package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"kapital/backend/models"
	"kapital/backend/services"
)

// The server's Distant Horizons terrain as a one-time download (issue 136,
// ADR-2 sixteenth amendment): the chapter's settings page reads what is
// published with GetDHSnapshot, and DownloadDHSnapshot fetches it into the
// instance on the player's request, with dh:download events for its progress.

// terrainDownloads is the app's side of it: where snapshots are read from, and
// the download under way for each chapter, at most one at a time in all.
type terrainDownloads struct {
	source *services.DHSource
	// emit, when set, takes the dh:download events in place of the window; a
	// test sets it.
	emit func(models.DHDownloadProgress)

	mu      sync.Mutex
	chapter string
	cancel  context.CancelFunc
}

func newTerrainDownloads() *terrainDownloads {
	return &terrainDownloads{source: services.NewDHSource()}
}

// begin claims the one download slot for a chapter, returning its context and
// the release (safe to call twice), or an error while another download runs.
func (t *terrainDownloads) begin(parent context.Context, chapterID string) (context.Context, func(), error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		return nil, nil, errors.New("another terrain download is under way; wait for it or cancel it first")
	}
	ctx, cancel := context.WithCancel(parent)
	t.chapter, t.cancel = chapterID, cancel
	return ctx, sync.OnceFunc(func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		cancel()
		t.chapter, t.cancel = "", nil
	}), nil
}

func (t *terrainDownloads) busy(chapterID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cancel != nil && t.chapter == chapterID
}

// running is whether any chapter's terrain is downloading.
func (t *terrainDownloads) running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cancel != nil
}

// stop cancels the chapter's download, reporting whether one was running.
func (t *terrainDownloads) stop(chapterID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel == nil || t.chapter != chapterID {
		return false
	}
	t.cancel()
	return true
}

// dhLook is a chapter's snapshot as read now, with the instance it would go in.
type dhLook struct {
	info        models.DHSnapshotInfo
	snap        services.DHSnapshot
	instanceDir string
}

// GetDHSnapshot reads the chapter's published terrain and what its instance
// already has (issue 136). One bounded GET of the snapshot's pointer and one of
// its index, on the host the manifest names; nothing is written. A chapter
// with no snapshot, no instance or an unreadable snapshot is an answer with a
// Reason, never an error.
func (a *App) GetDHSnapshot(chapterID string) (models.DHSnapshotInfo, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.DHSnapshotInfo{}, fmt.Errorf("no chapter %q", chapterID)
	}
	return a.lookDH(chapter).info, nil
}

// lookDH fills the answer step by step, stopping at the first reason a
// download could not start.
func (a *App) lookDH(chapter models.Chapter) dhLook {
	look := dhLook{info: models.DHSnapshotInfo{ChapterID: chapter.ID, Downloading: a.terrain.busy(chapter.ID)}}
	if chapter.Pack.DHSnapshot == nil {
		look.info.Reason = fmt.Sprintf("No terrain is published for %s yet.", chapter.Name)
		return look
	}
	_, cfg, err := a.chapterInstance(chapter.ID)
	if err != nil {
		look.info.Reason = fmt.Sprintf("Install %s first.", chapter.Name)
		return look
	}
	look.instanceDir = filepath.Dir(cfg)
	gameDir := services.GameFolder(look.instanceDir)
	if gameDir == "" {
		look.info.Reason = fmt.Sprintf("Play %s once first, so the pack is downloaded.", chapter.Name)
		return look
	}
	snap, err := a.terrain.source.Fetch(a.context(), *chapter.Pack.DHSnapshot)
	if err != nil {
		slog.Warn("dh snapshot not read", "chapter", chapter.ID, "error", err)
		look.info.Reason = "The published terrain could not be read. Try again later."
		return look
	}
	look.snap = snap
	look.info.CreatedAt = snap.Index.CreatedAt
	look.info.SizeBytes = snap.Index.Total()
	look.info.Dimensions = len(snap.Index.Levels)
	look.info.ServerDHVersion = snap.Index.DHVersion
	look.info.ExistingBytes = services.DHExistingBytes(look.instanceDir, snap.Index)
	look.info.PartialBytes = services.DHPartialBytes(look.instanceDir, snap)
	tracker, guess := a.chapterRunning(chapter, look.instanceDir)
	look.info.Running = tracker || guess
	look.info.Reason = a.dhVersionReason(gameDir, snap.Index, &look.info)
	switch {
	case look.info.Reason != "":
	case look.info.Running:
		look.info.Reason = "Close the game first: Distant Horizons keeps its terrain open while it runs."
	case !look.info.Downloading:
		look.info.Available = true
	}
	return look
}

// dhVersionReason refuses a snapshot from a newer Distant Horizons than the
// instance's: the client could not read the database.
func (a *App) dhVersionReason(gameDir string, ix services.DHIndex, info *models.DHSnapshotInfo) string {
	mods, err := services.ListMods(gameDir)
	if err != nil {
		slog.Warn("dh snapshot: mods not listed", "error", err)
	}
	info.ClientDHVersion = services.ClientDHVersion(mods)
	if info.ClientDHVersion == "" {
		return "Distant Horizons is not in this instance. Play once so the pack is up to date."
	}
	newer, err := services.DHVersionNewer(ix.DHVersion, info.ClientDHVersion)
	if err != nil || newer {
		return fmt.Sprintf("The server runs Distant Horizons %s, newer than this pack's %s. Update the pack first.", ix.DHVersion, info.ClientDHVersion)
	}
	return ""
}

// DownloadDHSnapshot downloads the chapter's published terrain into its
// instance and returns the snapshot read again. The chapter resolves through
// the manifest, and every folder the download writes comes from the validated
// index (services.ParseDHIndex). It is refused under a developer preview, while
// the game runs (asked again just before anything is placed), while another
// download runs, and when the instance already holds terrain for the server and
// replace is false. Progress is dh:download events. A cancelled download
// answers errDHCancelled, and the next one continues from what was verified.
func (a *App) DownloadDHSnapshot(chapterID string, replace bool) (models.DHSnapshotInfo, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.DHSnapshotInfo{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.DHSnapshotInfo{}, err
	}
	look := a.lookDH(chapter)
	if look.info.Running {
		return look.info, a.refuseIfRunning(chapter, look.instanceDir)
	}
	if !look.info.Available {
		return look.info, errors.New(look.info.Reason)
	}
	if look.info.ExistingBytes > 0 && !replace {
		return look.info, errors.New("the instance already has terrain for this server; confirm to replace it")
	}
	ctx, done, err := a.terrain.begin(a.trackContext(), chapter.ID)
	if err != nil {
		return look.info, err
	}
	defer done()
	emit := func(phase string, received, total int64) {
		a.emitDH(models.DHDownloadProgress{ChapterID: chapter.ID, Phase: phase, Received: received, Total: total})
	}
	emit("checking", look.info.PartialBytes, look.info.SizeBytes)
	err = a.terrain.source.Download(ctx, services.DHDownloadRequest{
		InstanceDir: look.instanceDir,
		Snapshot:    look.snap,
		Progress:    emit,
		BeforePlace: func() error { return a.refuseIfRunning(chapter, look.instanceDir) },
	})
	cancelled := ctx.Err() != nil
	done()
	switch {
	case err != nil && cancelled:
		slog.Info("dh snapshot cancelled", "chapter", chapter.ID)
		emit("cancelled", 0, look.info.SizeBytes)
		return a.lookDH(chapter).info, errDHCancelled
	case err != nil:
		slog.Error("dh snapshot", "chapter", chapter.ID, "error", err)
		a.emitDH(models.DHDownloadProgress{ChapterID: chapter.ID, Phase: "failed", Error: err.Error()})
		return a.lookDH(chapter).info, err
	default:
		slog.Info("dh snapshot placed", "chapter", chapter.ID, "bytes", look.info.SizeBytes, "createdAt", look.info.CreatedAt)
	}
	return a.lookDH(chapter).info, nil
}

// errDHCancelled is what a download the player cancelled answers.
var errDHCancelled = errors.New("the download was cancelled; what was already downloaded is kept for next time")

// CancelDHSnapshot cancels the chapter's terrain download, reporting whether
// one was running. What was verified stays for the next download.
func (a *App) CancelDHSnapshot(chapterID string) (bool, error) {
	if _, ok := a.chapter(chapterID); !ok {
		return false, fmt.Errorf("no chapter %q", chapterID)
	}
	return a.terrain.stop(chapterID), nil
}

// emitDH is one dh:download event.
func (a *App) emitDH(p models.DHDownloadProgress) {
	if a.terrain.emit != nil {
		a.terrain.emit(p)
		return
	}
	if a.ctx != nil {
		wailsrt.EventsEmit(a.ctx, services.EventDHDownload, p)
	}
}
