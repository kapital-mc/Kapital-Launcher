package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kapital/backend/models"
)

// The snapshot of a server's Distant Horizons terrain (issue 136) is published
// by Konnekt (kollektiv-mc/Konnekt, docs/distant-horizons.md) as three kinds of
// file under one address:
//
//	<base>/latest.json                       the pointer: which index is current
//	<base>/snapshots/<stamp>/index.json      the snapshot: folders, sizes, hashes
//	<base>/parts/<sha256>                    8 MiB pieces shared by every snapshot
//
// All three are untrusted input like the manifest (SECURITY_CHECKLIST S4.10).
// The manifest names the pointer; the index's address and the parts' are
// resolved against it, never taken whole from a file, so neither can move the
// download to another host. The folder names the index carries become paths in
// the instance, so they are held to Distant Horizons' own name rules.

const (
	// dhIndexFormat is the one index and pointer format this build reads.
	dhIndexFormat = 1
	// dhPointerMax and dhIndexMax bound the two small files.
	dhPointerMax = 4 << 10
	dhIndexMax   = 4 << 20
	// dhMaxLevels is how many dimensions one snapshot may carry.
	dhMaxLevels = 64
	// dhMaxPart is the largest part accepted. Konnekt writes 8 MiB.
	dhMaxPart = 64 << 20
	// dhMaxTotal is the largest snapshot accepted, every dimension together.
	dhMaxTotal = 64 << 30
	// dhNameMax is Distant Horizons' length limit for a server or level key
	// (LevelInitMessage.MAX_LENGTH, core 3.3.3).
	dhNameMax = 150
	// dhDatabaseName is the database file in each level folder.
	dhDatabaseName = "DistantHorizons.sqlite"
	// dhServerData is the folder under the game folder where a client keeps
	// the terrain of the servers it joined.
	dhServerData = "Distant_Horizons_server_data"
	// dhJarPrefix starts the Distant Horizons jar's name.
	dhJarPrefix = "DistantHorizons-"
)

var (
	// dhIndexRef is the only index address a pointer may name, relative to it.
	dhIndexRef = regexp.MustCompile(`^snapshots/[0-9]{8}T[0-9]{6}Z/index\.json$`)
	// dhPartsBase is the parts folder relative to the index: up to the
	// snapshot's prefix, then parts/. Nothing absolute, no scheme, no host.
	dhPartsBase = regexp.MustCompile(`^(\.\./){1,4}parts/$`)
	// dhServerFolder is "<serverId>_<serverKey>", the folder every client keeps
	// the server's terrain in once the server sets its serverKey.
	dhServerFolder = regexp.MustCompile(`^-?[0-9]+_[A-Za-z0-9_-]+$`)
	// dhLevelFolder is "<levelKeyPrefix>@<namespace>@@<path>", one dimension.
	dhLevelFolder = regexp.MustCompile(`^[A-Za-z0-9_-]+@[A-Za-z0-9_-]+@@[A-Za-z0-9_-]+$`)
	dhHash        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// dhVersionShape is a Distant Horizons version: numbers with dots, and an
	// optional suffix that the comparison ignores.
	dhVersionShape = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+){1,3})(?:[-+][A-Za-z0-9.]+)?$`)
	// dhJarVersion reads the version from the jar's name,
	// DistantHorizons-3.3.3-1.21.1-fabric-neoforge.jar.
	dhJarVersion = regexp.MustCompile(`^DistantHorizons-([0-9]+(?:\.[0-9]+){1,3})[-+.]`)
)

// DHPointer is latest.json.
type DHPointer struct {
	Format    int    `json:"format"`
	CreatedAt string `json:"createdAt"`
	Index     string `json:"index"`
}

// DHPart is one part of a database: its hash, which is also its name in the
// parts folder, and its length.
type DHPart struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// DHLevel is one dimension's database: the level folder it goes in on the
// client, its size and whole-file hash, and its parts in order.
type DHLevel struct {
	Folder string   `json:"folder"`
	Size   int64    `json:"size"`
	SHA256 string   `json:"sha256"`
	Parts  []DHPart `json:"parts"`
}

// DHIndex is index.json, format 1.
type DHIndex struct {
	Format       int       `json:"format"`
	CreatedAt    string    `json:"createdAt"`
	PartsBase    string    `json:"partsBase"`
	DHVersion    string    `json:"dhVersion"`
	ServerFolder string    `json:"serverFolder"`
	Levels       []DHLevel `json:"levels"`
}

// Total is the snapshot's size, every database together.
func (ix DHIndex) Total() int64 {
	var n int64
	for _, l := range ix.Levels {
		n += l.Size
	}
	return n
}

// checkDHSnapshotURL is the manifest's rule for pack.dhSnapshot: checkURL's
// https on the allowlist, a path that ends in /latest.json with no dot
// segment, and no query or fragment, so the addresses resolved from it stay
// plain paths on the same host.
func checkDHSnapshotURL(field, raw string) error {
	if err := checkURL(field, raw); err != nil {
		return err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("manifest: %s: %w", field, err)
	}
	if u.RawQuery != "" || u.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("manifest: %s: %q carries a query or a fragment", field, raw)
	}
	if !strings.HasSuffix(u.Path, "/latest.json") || path.Clean(u.Path) != u.Path || strings.Contains(u.Path, "//") {
		return fmt.Errorf("manifest: %s: %q is not a plain path ending in /latest.json", field, raw)
	}
	return nil
}

// decodeStrict decodes one JSON value, refusing unknown fields and anything
// after it: a field this build does not know is a newer format.
func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("data after the value")
	}
	return nil
}

// ParseDHPointer decodes and checks latest.json.
func ParseDHPointer(data []byte) (DHPointer, error) {
	var p DHPointer
	if err := decodeStrict(data, &p); err != nil {
		return DHPointer{}, fmt.Errorf("snapshot pointer: %w", err)
	}
	if p.Format != dhIndexFormat {
		return DHPointer{}, fmt.Errorf("snapshot pointer: format %d, this build reads %d", p.Format, dhIndexFormat)
	}
	if !dhIndexRef.MatchString(p.Index) {
		return DHPointer{}, fmt.Errorf("snapshot pointer: index %q is not snapshots/<stamp>/index.json", p.Index)
	}
	return p, nil
}

// ParseDHIndex decodes and checks index.json, refusing the whole index on any
// violation (issue 136): an unknown format, a folder outside Distant Horizons'
// name rules, a hash that is not 64 hex characters, sizes that do not add up,
// or a snapshot over the caps.
func ParseDHIndex(data []byte) (DHIndex, error) {
	var ix DHIndex
	if err := decodeStrict(data, &ix); err != nil {
		return DHIndex{}, fmt.Errorf("snapshot index: %w", err)
	}
	if err := validateDHIndex(ix); err != nil {
		return DHIndex{}, fmt.Errorf("snapshot index: %w", err)
	}
	return ix, nil
}

func validateDHIndex(ix DHIndex) error {
	switch {
	case ix.Format != dhIndexFormat:
		return fmt.Errorf("format %d, this build reads %d", ix.Format, dhIndexFormat)
	case !validDHTime(ix.CreatedAt):
		return fmt.Errorf("createdAt %q is not an RFC 3339 time", ix.CreatedAt)
	case !dhPartsBase.MatchString(ix.PartsBase):
		return fmt.Errorf("partsBase %q is not a relative parts folder", ix.PartsBase)
	case !dhVersionShape.MatchString(ix.DHVersion):
		return fmt.Errorf("dhVersion %q is not a version", ix.DHVersion)
	case !dhServerFolder.MatchString(ix.ServerFolder) || len(ix.ServerFolder) > dhNameMax:
		return fmt.Errorf("serverFolder %q is not <serverId>_<serverKey>", ix.ServerFolder)
	case len(ix.Levels) == 0 || len(ix.Levels) > dhMaxLevels:
		return fmt.Errorf("%d levels, between 1 and %d are accepted", len(ix.Levels), dhMaxLevels)
	}
	seen := map[string]bool{}
	var total int64
	for i, l := range ix.Levels {
		if err := validateDHLevel(l); err != nil {
			return fmt.Errorf("levels[%d]: %w", i, err)
		}
		if seen[l.Folder] {
			return fmt.Errorf("levels[%d]: folder %q appears twice", i, l.Folder)
		}
		seen[l.Folder] = true
		total += l.Size
		if total > dhMaxTotal {
			return fmt.Errorf("the snapshot is over %d bytes", int64(dhMaxTotal))
		}
	}
	return nil
}

func validateDHLevel(l DHLevel) error {
	switch {
	case !dhLevelFolder.MatchString(l.Folder) || len(l.Folder) > dhNameMax+1:
		return fmt.Errorf("folder %q is not <levelKeyPrefix>@<namespace>@@<path>", l.Folder)
	case !dhHash.MatchString(l.SHA256):
		return fmt.Errorf("sha256 %q is not 64 lowercase hex characters", l.SHA256)
	case l.Size <= 0 || l.Size > dhMaxTotal:
		return fmt.Errorf("size %d is out of range", l.Size)
	case len(l.Parts) == 0:
		return fmt.Errorf("no parts")
	}
	var sum int64
	for j, p := range l.Parts {
		if !dhHash.MatchString(p.SHA256) {
			return fmt.Errorf("parts[%d]: sha256 %q is not 64 lowercase hex characters", j, p.SHA256)
		}
		if p.Size <= 0 || p.Size > dhMaxPart {
			return fmt.Errorf("parts[%d]: size %d is out of range", j, p.Size)
		}
		sum += p.Size
		if sum > l.Size {
			return fmt.Errorf("the parts add up to more than the level's %d bytes", l.Size)
		}
	}
	if sum != l.Size {
		return fmt.Errorf("the parts add up to %d bytes, the level says %d", sum, l.Size)
	}
	return nil
}

func validDHTime(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// dhIndexURL resolves the pointer's index against the pointer's own address.
func dhIndexURL(pointer *url.URL, p DHPointer) (*url.URL, error) {
	return dhResolve(pointer, p.Index)
}

// dhPartURL is a part's address: the index's partsBase resolved against the
// index's own address, then the part's hash.
func dhPartURL(index *url.URL, ix DHIndex, part DHPart) (*url.URL, error) {
	return dhResolve(index, ix.PartsBase+part.SHA256)
}

// dhResolve resolves a relative reference that passed its shape check, and
// refuses a result that left the base's scheme or host, or climbed above the
// root, which the shape checks already rule out.
func dhResolve(base *url.URL, ref string) (*url.URL, error) {
	r, err := url.Parse(ref)
	if err != nil {
		return nil, err
	}
	if r.IsAbs() || r.Host != "" || strings.HasPrefix(ref, "/") {
		return nil, fmt.Errorf("snapshot: %q is not a relative address", ref)
	}
	u := base.ResolveReference(r)
	if u.Scheme != "https" || u.Host != base.Host || u.User != nil || u.RawQuery != "" {
		return nil, fmt.Errorf("snapshot: %q leaves the snapshot's address", ref)
	}
	return u, nil
}

// ClientDHVersion is the Distant Horizons version of the instance's jar, read
// from its name whether it is switched on or off, or "" when there is none.
func ClientDHVersion(mods []models.ModFile) string {
	for _, m := range mods {
		if v := dhJarVersion.FindStringSubmatch(m.Name); v != nil {
			return v[1]
		}
	}
	return ""
}

// DHVersionNewer reports whether version a is newer than b, comparing the
// numbers only. A version that is not one is an error.
func DHVersionNewer(a, b string) (bool, error) {
	pa, err := dhVersionParts(a)
	if err != nil {
		return false, err
	}
	pb, err := dhVersionParts(b)
	if err != nil {
		return false, err
	}
	for i := range max(len(pa), len(pb)) {
		x, y := dhVersionAt(pa, i), dhVersionAt(pb, i)
		if x != y {
			return x > y, nil
		}
	}
	return false, nil
}

func dhVersionParts(v string) ([]int, error) {
	m := dhVersionShape.FindStringSubmatch(v)
	if m == nil {
		return nil, fmt.Errorf("%q is not a Distant Horizons version", v)
	}
	var parts []int
	for _, s := range strings.Split(m[1], ".") {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("%q is not a Distant Horizons version", v)
		}
		parts = append(parts, n)
	}
	return parts, nil
}

func dhVersionAt(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}
	return 0
}
