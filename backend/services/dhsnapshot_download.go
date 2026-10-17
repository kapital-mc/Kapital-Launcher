package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// The download of a snapshot into a chapter's instance (issue 136, ADR-2
// sixteenth amendment). Everything it writes goes through an os.Root on the
// instance folder:
//
//	<instance>/kapital-dh-download/journal.json       which parts are verified
//	<instance>/kapital-dh-download/<level>.sqlite     a database being assembled
//	<instance>/<game>/Distant_Horizons_server_data/<serverFolder>/<level>/DistantHorizons.sqlite
//
// The staging folder sits beside the game folder, out of the game's sight and
// on the same volume, so placing a database is a rename. Parts are written at
// their offsets into the level's staged file and the journal counts a part
// only once its bytes matched its hash, so a cancelled or failed download
// continues from there. A level moves into place only after its whole file
// matched the index's hash, and only once BeforePlace (the running check) says
// so: Distant Horizons holds its database open while the game runs.

const (
	dhStaging     = "kapital-dh-download"
	dhJournalName = "journal.json"
	// dhSpareSpace is kept free beyond the download itself.
	dhSpareSpace = 256 << 20
	// dhProgressEvery is how many bytes pass between two progress reports.
	dhProgressEvery = 1 << 20
)

// dhSidecars are SQLite's files beside a database. A write-ahead log left from
// the player's own database would be replayed into the downloaded one, so they
// go when it is placed.
var dhSidecars = []string{"-wal", "-shm", "-journal"}

// DHDownloadRequest is one download.
type DHDownloadRequest struct {
	// InstanceDir is the chapter's instance folder, the one holding instance.cfg.
	InstanceDir string
	Snapshot    DHSnapshot
	// Progress hears each phase and, while downloading, the bytes so far of
	// the whole snapshot.
	Progress func(phase string, received, total int64)
	// BeforePlace is asked just before the databases move into the game
	// folder; an error stops there and keeps what was downloaded.
	BeforePlace func() error
}

// dhJournal is what survives a cancelled download: the snapshot it belongs
// to, by its index's hash, and how many parts of each level are verified.
type dhJournal struct {
	Format int            `json:"format"`
	Index  string         `json:"index"`
	Done   map[string]int `json:"done"`
}

// Download fetches what the staging folder lacks, checks every level against
// its hash, and places the levels in the game folder.
func (s *DHSource) Download(ctx context.Context, req DHDownloadRequest) error {
	gameDir := gameFolder(req.InstanceDir)
	if gameDir == "" {
		return errors.New("the instance has no game folder yet: play once so the pack is downloaded")
	}
	root, err := os.OpenRoot(req.InstanceDir)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck // nothing was written through the handle itself
	ix := req.Snapshot.Index
	j, err := openDHJournal(root, req.Snapshot.IndexHash)
	if err != nil {
		return err
	}
	total := ix.Total()
	p := &dhProgress{report: req.Progress, total: total, received: j.verified(ix)}
	if err := checkDHSpace(req.InstanceDir, total-p.received); err != nil {
		return err
	}
	p.phase("downloading")
	for _, l := range ix.Levels {
		if err := s.downloadLevel(ctx, root, req.Snapshot, l, &j, p); err != nil {
			return err
		}
	}
	p.phase("verifying")
	for _, l := range ix.Levels {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := verifyStaged(root, l, &j); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.BeforePlace != nil {
		if err := req.BeforePlace(); err != nil {
			return err
		}
	}
	p.phase("placing")
	for _, l := range ix.Levels {
		if err := placeDHLevel(root, filepath.Base(gameDir), ix.ServerFolder, l); err != nil {
			return err
		}
	}
	if err := root.RemoveAll(dhStaging); err != nil {
		slog.Warn("dh snapshot: staging not removed", "error", err)
	}
	p.phase("done")
	return nil
}

// downloadLevel fetches the parts of one level the journal does not count yet.
func (s *DHSource) downloadLevel(ctx context.Context, root *os.Root, snap DHSnapshot, l DHLevel, j *dhJournal, p *dhProgress) (err error) {
	f, err := root.OpenFile(stagedLevel(l), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	var off int64
	for i, part := range l.Parts {
		if i < j.Done[l.Folder] {
			off += part.Size
			continue
		}
		u, err := dhPartURL(snap.IndexURL, snap.Index, part)
		if err != nil {
			return err
		}
		if err := s.fetchPart(ctx, u, part, f, off, p.add); err != nil {
			return err
		}
		off += part.Size
		j.Done[l.Folder] = i + 1
		if err := j.save(root); err != nil {
			return err
		}
	}
	return f.Truncate(l.Size)
}

// verifyStaged checks a level's assembled file against the index's hash. A
// file that does not match is removed and its parts are fetched again next time.
func verifyStaged(root *os.Root, l DHLevel, j *dhJournal) error {
	got, err := hashRootFile(root, stagedLevel(l))
	if err == nil && got == l.SHA256 {
		return nil
	}
	if rerr := root.Remove(stagedLevel(l)); rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
		err = errors.Join(err, rerr)
	}
	j.Done[l.Folder] = 0
	if serr := j.save(root); serr != nil {
		err = errors.Join(err, serr)
	}
	if err != nil {
		slog.Warn("dh snapshot: a level could not be checked", "level", l.Folder, "error", err)
	} else {
		slog.Warn("dh snapshot: a level did not match its hash", "level", l.Folder)
	}
	return fmt.Errorf("the downloaded %s did not match the snapshot; download again to fetch it fresh", l.Folder)
}

func hashRootFile(root *os.Root, name string) (sum string, err error) {
	f, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// placeDHLevel moves a verified level into the game folder, over whatever
// database was there, removing that database's SQLite side files first.
func placeDHLevel(root *os.Root, game, serverFolder string, l DHLevel) error {
	dir := filepath.Join(game, dhServerData, serverFolder, l.Folder)
	if err := root.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	db := filepath.Join(dir, dhDatabaseName)
	for _, ext := range dhSidecars {
		if err := root.Remove(db + ext); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return root.Rename(stagedLevel(l), db)
}

func stagedLevel(l DHLevel) string { return filepath.Join(dhStaging, l.Folder+".sqlite") }

// openDHJournal reads the staging folder's journal. One from another
// snapshot, or none, starts over with an empty staging folder: a part's place
// in a file belongs to the index that laid it out.
func openDHJournal(root *os.Root, indexHash string) (dhJournal, error) {
	if j, ok := readDHJournal(root); ok && j.Index == indexHash {
		return j, nil
	}
	if err := root.RemoveAll(dhStaging); err != nil {
		return dhJournal{}, err
	}
	if err := root.MkdirAll(dhStaging, 0o755); err != nil {
		return dhJournal{}, err
	}
	j := dhJournal{Format: 1, Index: indexHash, Done: map[string]int{}}
	return j, j.save(root)
}

func readDHJournal(root *os.Root) (dhJournal, bool) {
	data, err := root.ReadFile(filepath.Join(dhStaging, dhJournalName))
	if err != nil {
		return dhJournal{}, false
	}
	var j dhJournal
	if err := json.Unmarshal(data, &j); err != nil || j.Format != 1 || j.Done == nil {
		return dhJournal{}, false
	}
	return j, true
}

// save writes the journal beside a temporary name and renames it over.
func (j dhJournal) save(root *os.Root) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	name := filepath.Join(dhStaging, dhJournalName)
	if err := root.WriteFile(name+".tmp", data, 0o644); err != nil {
		return err
	}
	return root.Rename(name+".tmp", name)
}

// verified is the bytes of the snapshot the journal counts as fetched.
func (j dhJournal) verified(ix DHIndex) int64 {
	var n int64
	for _, l := range ix.Levels {
		for i, part := range l.Parts {
			if i < j.Done[l.Folder] {
				n += part.Size
			}
		}
	}
	return n
}

// DHPartialBytes is what an interrupted download of this snapshot left
// verified in the instance, 0 when none did.
func DHPartialBytes(instanceDir string, snap DHSnapshot) int64 {
	root, err := os.OpenRoot(instanceDir)
	if err != nil {
		return 0
	}
	defer root.Close() //nolint:errcheck // read only
	j, ok := readDHJournal(root)
	if !ok || j.Index != snap.IndexHash {
		return 0
	}
	return j.verified(snap.Index)
}

// DHExistingBytes is what the instance already holds in the snapshot's level
// folders: databases and their side files, which a download replaces.
func DHExistingBytes(instanceDir string, ix DHIndex) int64 {
	gameDir := gameFolder(instanceDir)
	if gameDir == "" {
		return 0
	}
	var n int64
	for _, l := range ix.Levels {
		db := filepath.Join(gameDir, dhServerData, ix.ServerFolder, l.Folder, dhDatabaseName)
		for _, ext := range append([]string{""}, dhSidecars...) {
			if fi, err := os.Lstat(db + ext); err == nil && fi.Mode().IsRegular() {
				n += fi.Size()
			}
		}
	}
	return n
}

// checkDHSpace refuses a download the disk cannot hold. Where free space
// cannot be read the download goes ahead and a full disk fails it instead.
func checkDHSpace(dir string, need int64) error {
	free, err := diskFree(dir)
	if err != nil {
		slog.Info("dh snapshot: free space unknown", "error", err)
		return nil
	}
	if free < need+dhSpareSpace {
		return fmt.Errorf("not enough free space: the download needs %s more and the disk has %s", gigabytes(need+dhSpareSpace), gigabytes(free))
	}
	return nil
}

func gigabytes(n int64) string { return fmt.Sprintf("%.1f GB", float64(n)/(1<<30)) }

// dhProgress turns bytes into progress reports, one per megabyte.
type dhProgress struct {
	report                func(phase string, received, total int64)
	received, total, last int64
}

func (p *dhProgress) phase(name string) {
	if p.report != nil {
		p.report(name, p.received, p.total)
	}
}

func (p *dhProgress) add(n int64) {
	p.received += n
	if n < 0 || p.received-p.last >= dhProgressEvery {
		p.last = p.received
		p.phase("downloading")
	}
}
