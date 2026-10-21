package services

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"time"

	"kapital/backend/models"
)

// Picking up a game that is already running when the launcher starts: after a
// restart into an update, or a launcher closed and opened again while the game
// ran (ADR-2, nineteenth amendment). Without it the chapter shows Play while
// its game is up, and a second Play hands Prism a launch of an instance that is
// running.
//
// A game is adopted only when it is a Java whose parent is the detected Prism
// (by name and image path, as OtherPrismOpen matches it), and only for the
// chapter whose game log began after that Java started and has not said
// "Stopping!". Prism records lastLaunchTime in the instance's instance.cfg as it
// hands the running Java its launch (LauncherPartLaunch.cpp), which settles
// which chapter it is when more than one log fits. The run that follows ends
// only that Java on Stop: the Prism was not started by this launcher and is
// never asked to close or ended (S3.9).

const (
	// adoptLogSlack is how much earlier than the game's Java its log may have
	// been begun, for a process start time read coarsely.
	adoptLogSlack = 2 * time.Second
	// adoptLaunchBefore and adoptLaunchAfter bound lastLaunchTime around the
	// Java's start: Prism writes it once the Java is up and asks to launch.
	adoptLaunchBefore = 5 * time.Second
	adoptLaunchAfter  = 2 * time.Minute
)

// AdoptCandidate is a chapter whose game may already be running: its id and
// its Prism instance folder.
type AdoptCandidate struct {
	ChapterID   string
	InstanceDir string
}

// runningJava is a Java found under the detected Prism.
type runningJava struct {
	pid     int
	started time.Time
}

// adoptMatch is a chapter whose log fits a Java, read up to where the game is.
type adoptMatch struct {
	chapter  AdoptCandidate
	follower *logFollower
	phase    string
	launched bool
}

// Adopt follows the games of these chapters that are already running from the
// Prism at exe, each as a run of its own, and returns the chapters it took up.
// A chapter already followed, a Java whose start time cannot be read and a
// Java that fits more than one chapter are left alone. An OS with no process
// lookup adopts nothing.
func (t *GameTracker) Adopt(ctx context.Context, exe string, chapters []AdoptCandidate) []string {
	if exe == "" || len(chapters) == 0 {
		return nil
	}
	procs, err := t.os.list()
	if err != nil {
		slog.Info("running game lookup unavailable", "error", err)
		return nil
	}
	claimed := map[string]bool{}
	var adopted []string
	for _, g := range t.runningJavas(procs, exe) {
		m, ok := t.matchChapter(g, chapters, claimed)
		if !ok {
			continue
		}
		claimed[m.chapter.ChapterID] = true
		if err := t.adopt(ctx, exe, g, m); err != nil {
			slog.Info("running game not followed", "chapter", m.chapter.ChapterID, "error", err)
			continue
		}
		slog.Info("running game followed", "chapter", m.chapter.ChapterID, "phase", m.phase, "pid", g.pid)
		adopted = append(adopted, m.chapter.ChapterID)
	}
	return adopted
}

// runningJavas is every Java whose parent is the Prism at exe, newest first,
// so that two games of one Prism meet their chapters in launch order.
func (t *GameTracker) runningJavas(procs []procInfo, exe string) []runningJava {
	prisms := matchPrismProcesses(procs, t.os.image, exe, 0)
	var out []runningJava
	for _, p := range procs {
		if !isJavaName(p.Name) || !slices.Contains(prisms, p.PPID) {
			continue
		}
		started, known := t.os.started(p.PID)
		if !known {
			continue
		}
		out = append(out, runningJava{pid: p.PID, started: started})
	}
	slices.SortFunc(out, func(a, b runningJava) int { return b.started.Compare(a.started) })
	return out
}

// matchChapter is the one chapter a Java is running: of the chapters not yet
// followed whose log began after the Java and is still going, the only one, or
// the only one whose lastLaunchTime falls at the Java's start.
func (t *GameTracker) matchChapter(g runningJava, chapters []AdoptCandidate, claimed map[string]bool) (adoptMatch, bool) {
	now := t.now()
	var fits []adoptMatch
	for _, c := range chapters {
		if claimed[c.ChapterID] || c.InstanceDir == "" || t.Active(c.ChapterID) {
			continue
		}
		f := newLogFollower(c.InstanceDir, GameLogSnapshot{}, g.started.Add(-adoptLogSlack))
		phases, _ := f.poll(now) //nolint:errcheck // not an error: a first poll has no earlier log to restart from
		if !f.Fresh() {
			continue
		}
		phase := models.GamePhaseStarting
		for _, p := range phases {
			if phaseRank[p] > phaseRank[phase] {
				phase = p
			}
		}
		if phase == models.GamePhaseStopping {
			continue
		}
		fits = append(fits, adoptMatch{chapter: c, follower: f, phase: phase, launched: launchedAt(c.InstanceDir, g.started)})
	}
	if len(fits) == 1 {
		return fits[0], true
	}
	var launched []adoptMatch
	for _, m := range fits {
		if m.launched {
			launched = append(launched, m)
		}
	}
	if len(launched) == 1 {
		return launched[0], true
	}
	if len(fits) > 1 {
		slog.Info("running game fits more than one chapter, not followed", "pid", g.pid, "chapters", len(fits))
	}
	return adoptMatch{}, false
}

// launchedAt is whether the instance's lastLaunchTime falls at a Java started
// at started. Only that key and totalTimePlayed are read (ReadPlayTime).
func launchedAt(instanceDir string, started time.Time) bool {
	pt, err := ReadPlayTime(filepath.Join(instanceDir, "instance.cfg"), "")
	if err != nil || pt.LastLaunchMs == 0 {
		return false
	}
	at := time.UnixMilli(pt.LastLaunchMs)
	return !at.Before(started.Add(-adoptLaunchBefore)) && !at.After(started.Add(adoptLaunchAfter))
}

// adopt follows one matched game from the phase its log has reached, bound to
// its Java from the start.
func (t *GameTracker) adopt(ctx context.Context, exe string, g runningJava, m adoptMatch) error {
	req := TrackRequest{
		ChapterID:   m.chapter.ChapterID,
		InstanceDir: m.chapter.InstanceDir,
		// No Prism: the one the game runs under was not started by this
		// launcher. With pid 0 it is never alive to Stop (prismAlive), never
		// recorded for a console (holdPrismConsole) and never searched for
		// another Java (findGame).
		PrismExe:  filepath.Base(exe),
		StartedAt: g.started,
		Adopted:   true,
	}
	r, err := t.beginAt(ctx, req, m.follower, m.phase)
	if err != nil {
		return err
	}
	r.bind(g.pid)
	t.start(r)
	return nil
}
