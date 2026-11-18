package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// The mod count beside a chapter comes from the pack it syncs from, not from
// the manifest, so a pack release needs no launcher release. pack.toml's
// [index] table names index.toml beside it and that file's SHA-256; the
// index lists every file of the pack, and each entry directly under mods/ is
// one mod (a .pw.toml for a downloaded one, a .jar the pack carries itself).
// The index is fetched only when its hash is one the launcher has not counted
// yet in this run, and is refused unless it matches that hash.
const (
	// maxIndexTOML bounds index.toml: Luxemburg's 202 mods are about 95 KB.
	maxIndexTOML = 1 << 20
	// maxModCounts bounds the counts kept: one per index hash seen.
	maxModCounts = 32
)

var indexFileName = regexp.MustCompile(`^[0-9A-Za-z_-][0-9A-Za-z._-]{0,63}\.toml$`)

// packIndexRef is pack.toml's [index] table: the index's file name beside
// pack.toml and its SHA-256 in lowercase hex.
type packIndexRef struct {
	file string
	hash string
}

// scanPackIndex reads pack.toml's [index] table. Only a plain file name beside
// pack.toml and a SHA-256 hash are accepted; anything else is refused.
func scanPackIndex(raw []byte) (packIndexRef, error) {
	var ref packIndexRef
	var format string
	sc := bufio.NewScanner(bytes.NewReader(raw))
	inIndex := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inIndex = line == "[index]"
			continue
		}
		if !inIndex {
			continue
		}
		key, rawValue, ok := strings.Cut(line, "=")
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if !ok || err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "file":
			ref.file = value
		case "hash-format":
			format = value
		case "hash":
			ref.hash = strings.ToLower(value)
		}
	}
	if err := sc.Err(); err != nil {
		return packIndexRef{}, fmt.Errorf("pack.toml: %w", err)
	}
	if !indexFileName.MatchString(ref.file) {
		return packIndexRef{}, fmt.Errorf("pack.toml: index file %q is not a name beside it", ref.file)
	}
	if format != packwizHashSHA256 || !sha256Hex.MatchString(ref.hash) {
		return packIndexRef{}, errors.New("pack.toml: the index hash is not SHA-256")
	}
	return ref, nil
}

// countIndexMods counts index.toml's entries directly under mods/: one
// `file = "mods/<name>"` line each. A file in a folder under mods/ is not a mod.
func countIndexMods(raw []byte) int {
	n := 0
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		key, rawValue, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || strings.TrimSpace(key) != "file" {
			continue
		}
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if err != nil {
			continue
		}
		if name, ok := strings.CutPrefix(value, "mods/"); ok && name != "" && !strings.Contains(name, "/") {
			n++
		}
	}
	return n
}

// siblingURL is a file beside a pack.toml: the same URL with its last path
// segment replaced, so the scheme, host and folder stay the pack's own. A URL
// that does not end in /pack.toml, or carries user info, a query or a
// fragment, has none.
func siblingURL(packURL, name string) (string, error) {
	u, err := url.Parse(packURL)
	if err != nil {
		return "", err
	}
	dir, ok := strings.CutSuffix(u.Path, "/pack.toml")
	if !ok || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.Contains(packURL, "#") {
		return "", errors.New("the pack URL does not end in /pack.toml, or carries more than a path")
	}
	u.Path, u.RawPath = dir+"/"+name, ""
	return u.String(), nil
}

// packMods is the number of mods in the pack whose pack.toml is packRaw, read
// from its index under the rule the pack's URL passed. A count is kept per
// index hash, so an unchanged pack is not fetched again.
func (c *InstanceCreator) packMods(ctx context.Context, packURL string, check func(field, raw string) error, packRaw []byte) (int, error) {
	ref, err := scanPackIndex(packRaw)
	if err != nil {
		return 0, err
	}
	c.countsMu.Lock()
	n, ok := c.modCounts[ref.hash]
	c.countsMu.Unlock()
	if ok {
		return n, nil
	}
	rawURL, err := siblingURL(packURL, ref.file)
	if err != nil {
		return 0, fmt.Errorf("index: %w", err)
	}
	if err := check("index.toml", rawURL); err != nil {
		return 0, err
	}
	raw, err := c.fetchFile(ctx, "index.toml", rawURL, check, maxIndexTOML)
	if err != nil {
		return 0, err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != ref.hash {
		return 0, errors.New("index.toml does not match the hash pack.toml names")
	}
	n = countIndexMods(raw)
	c.countsMu.Lock()
	if c.modCounts == nil || len(c.modCounts) >= maxModCounts {
		c.modCounts = map[string]int{}
	}
	c.modCounts[ref.hash] = n
	c.countsMu.Unlock()
	return n, nil
}
