package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"kapital/backend/models"
)

const (
	playLedgerFile = "playtime.json"
	latestLogName  = "latest.log"
)

// PlayLedger keeps, per chapter, what each finished game log added up to
// (issue 341): numbers in the app data dir, so a log is unpacked once and the
// time survives the log being deleted. Its keys are a log's file name and size.
type PlayLedger struct {
	dataDir string
	mu      sync.Mutex
}

// NewPlayLedger makes a ledger in the app data dir.
func NewPlayLedger(dataDir string) *PlayLedger {
	return &PlayLedger{dataDir: dataDir}
}

// Count returns a chapter's seconds in a world with AFK left out: every log the
// ledger holds, after adding the finished ones that are new, and latest.log,
// measured fresh because it is still growing or will be archived. An instance
// with no game folder has only what the ledger already holds.
func (l *PlayLedger) Count(chapterID, instanceDir string) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ledger := l.load()
	runs := ledger[chapterID]
	if runs == nil {
		runs = map[string]playRun{}
	}
	latest, changed, err := countPlayLogs(instanceDir, runs)
	if err != nil {
		return 0, err
	}
	if changed {
		ledger[chapterID] = runs
		l.save(ledger)
	}
	total := latest
	for _, run := range runs {
		total += run.Seconds
	}
	return total, nil
}

// countPlayLogs adds the finished logs that are not in runs and returns
// latest.log's seconds and whether runs changed. A log that cannot be read (the
// game may be packing it) is left out and asked about again next time.
func countPlayLogs(instanceDir string, runs map[string]playRun) (int64, bool, error) {
	game := gameFolder(instanceDir)
	if game == "" {
		return 0, false, nil
	}
	root, err := os.OpenRoot(game)
	if err != nil {
		return 0, false, fmt.Errorf("open game folder: %w", err)
	}
	defer closeReadOnly(root)
	files, err := listRunFiles(root, models.RunLogKindLog)
	if err != nil {
		return 0, false, err
	}
	sortOldestFirst(files)
	var latest int64
	changed := false
	before := playRun{}
	for i, f := range files {
		key := fmt.Sprintf("%s:%d", f.name, f.size)
		if run, ok := runs[key]; ok && f.name != latestLogName {
			before = run
			continue
		}
		run, err := measurePlayFile(root, f.name, before, playerFromEarlier(root, files[:i]))
		if err != nil {
			slog.Warn("play time log not read", "log", f.name, "error", err)
			before = playRun{}
			continue
		}
		before = run
		if f.name == latestLogName {
			latest = run.Seconds
			continue
		}
		runs[key] = run
		changed = true
	}
	return latest, changed, nil
}

// sortOldestFirst orders logs by when they ended, latest.log last. The times are
// what says it: the number in an archive's name does not follow the runs.
func sortOldestFirst(files []runFile) {
	sort.SliceStable(files, func(i, j int) bool {
		a, b := files[i], files[j]
		if a.name == latestLogName || b.name == latestLogName {
			return b.name == latestLogName && a.name != latestLogName
		}
		return newerFirst(b, a)
	})
}

func measurePlayFile(root *os.Root, name string, before playRun, lookup func() string) (playRun, error) {
	rd, closeAll, err := openRunStream(root, filepath.Join("logs", name), strings.HasSuffix(name, ".gz"))
	if err != nil {
		return playRun{}, err
	}
	defer closeAll()
	return measurePlay(rd, before, lookup)
}

// playerFromEarlier finds the player's name in the logs before this one, newest
// first, for a log that does not name them.
func playerFromEarlier(root *os.Root, earlier []runFile) func() string {
	return func() string {
		for i := len(earlier) - 1; i >= 0; i-- {
			name := earlier[i].name
			if player := headPlayer(root, filepath.Join("logs", name), strings.HasSuffix(name, ".gz")); player != "" {
				return player
			}
		}
		return ""
	}
}

func (l *PlayLedger) path() string { return filepath.Join(l.dataDir, playLedgerFile) }

func (l *PlayLedger) load() map[string]map[string]playRun {
	ledger := map[string]map[string]playRun{}
	raw, err := os.ReadFile(l.path())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("play ledger read", "error", err)
		}
		return ledger
	}
	if err := json.Unmarshal(raw, &ledger); err != nil || ledger == nil {
		// Measured again from the logs that still exist.
		slog.Warn("play ledger parse", "error", err)
		return map[string]map[string]playRun{}
	}
	return ledger
}

// save writes the ledger. A failure costs reading the same logs again next time.
func (l *PlayLedger) save(ledger map[string]map[string]playRun) {
	raw, err := json.Marshal(ledger)
	if err != nil {
		slog.Warn("play ledger encode", "error", err)
		return
	}
	if err := writeFileAtomic(l.path(), raw, 0o600); err != nil {
		slog.Warn("play ledger write", "error", err)
	}
}
