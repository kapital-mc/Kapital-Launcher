package services

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

// What Stop ends besides Prism, and the Prism Play closes first (ADR-2,
// eighteenth amendment).
//
// Prism runs the pack sync as its pre-launch command: the launcher's own copy,
// which runs packwiz in a Java of its own. Ending Prism does not end them (on
// Windows a child outlives its parent, on macOS it is handed to launchd), so a
// Stop during the download left it running. Stop therefore ends every process
// below the launcher's Prism that started after Play, found by parent pid in the
// same process list the game's Java is found in, and never by name.
//
// A Prism that is already open takes the launch from the one Play starts, and
// that one is not the launcher's to stop. So Play first asks such a Prism to
// close, as its own close button does, when nothing runs under it: no game, no
// download. One that is busy, or does not close in time, is left alone and takes
// the launch as before.

// launchDescendants is every process below root, at any depth, that started at
// or after since: what Prism started for this launch. A process whose start
// cannot be read is left out, with everything below it, because a pid seen as a
// parent may be one the OS gave to another process.
func launchDescendants(procs []procInfo, root int, since time.Time, started func(pid int) (time.Time, bool)) []int {
	var out []int
	parents := []int{root}
	for len(parents) > 0 {
		var next []int
		for _, p := range procs {
			if !slices.Contains(parents, p.PPID) || p.PID == root || slices.Contains(out, p.PID) {
				continue
			}
			created, known := started(p.PID)
			if !known || created.Before(since) {
				continue
			}
			out = append(out, p.PID)
			next = append(next, p.PID)
		}
		parents = next
	}
	return out
}

// endLaunchChildren ends what the launcher's Prism started for this launch (the
// pack sync and its Java, a Java download), except the game's Java, which Stop
// ends on its own. Called by Stop before Prism is asked to close and again before
// it is ended. It never fails the stop.
func (r *gameRun) endLaunchChildren() {
	if !r.prismAlive() || r.procUnavailable {
		return
	}
	procs, err := r.t.os.list()
	if err != nil {
		slog.Info("stop: process lookup unavailable", "chapter", r.req.ChapterID, "error", err)
		return
	}
	for _, pid := range launchDescendants(procs, r.req.Prism.PID, r.req.StartedAt, r.t.os.started) {
		if pid == r.bound || !endable(pid) {
			continue
		}
		if err := r.t.os.terminate(pid, true); err != nil {
			slog.Warn("stop: end a process prism started", "chapter", r.req.ChapterID, "pid", pid, "error", err)
			continue
		}
		slog.Info("stop: ended", "chapter", r.req.ChapterID, "target", "prism child", "pid", pid)
	}
}

// CloseIdlePrisms asks every Prism running from exe to close, as its close
// button does, when it is idle: nothing runs under it, and it is not the Prism of
// a run the launcher is following. It waits up to stopForce for them to exit and
// reports whether one is still open. Nothing is ended: a Prism that stays is the
// player's, and takes the launch. An empty exe, or an OS with no process lookup,
// closes nothing.
func (t *GameTracker) CloseIdlePrisms(exe string) bool {
	if exe == "" {
		return false
	}
	procs, err := t.os.list()
	if err != nil {
		slog.Info("could not tell whether Prism is open", "error", err)
		return false
	}
	live := t.livePrisms()
	var asked []int
	open := false
	for _, pid := range matchPrismProcesses(procs, t.os.image, exe, 0) {
		if slices.Contains(live, pid) {
			continue
		}
		if !endable(pid) || hasChild(procs, pid) {
			slog.Info("prism is open and busy, left alone", "pid", pid)
			open = true
			continue
		}
		if err := t.os.askClose(pid); err != nil {
			slog.Info("prism is open and took no close request", "pid", pid, "error", err)
			open = true
			continue
		}
		slog.Info("prism is open, asked to close before play", "pid", pid)
		asked = append(asked, pid)
	}
	ctx, cancel := context.WithTimeout(context.Background(), t.stopForce)
	defer cancel()
	for _, pid := range asked {
		if _, _, err := t.os.wait(ctx, pid); err != nil {
			slog.Info("prism did not close before play", "pid", pid, "error", err)
			open = true
		}
	}
	return open
}

// livePrisms is the pids of the Prisms the launcher started for runs that are
// still going.
func (t *GameTracker) livePrisms() []int {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []int
	for _, c := range t.consoles {
		if !c.over.Load() {
			out = append(out, c.pid)
		}
	}
	for _, r := range t.live {
		out = append(out, r.req.Prism.PID)
	}
	return out
}

// hasChild is whether any process is a child of pid.
func hasChild(procs []procInfo, pid int) bool {
	for _, p := range procs {
		if p.PPID == pid && p.PID != pid {
			return true
		}
	}
	return false
}
