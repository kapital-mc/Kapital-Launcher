package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"kapital/backend/models"
)

const adoptExe = "/Prism/prismlauncher.exe"

// adoptRig is a game rig whose game was started before the tracker: Prism
// (100) and its Java (200) are running, and the log says the game is up.
func adoptRig(t *testing.T) (*gameRig, context.Context) {
	t.Helper()
	r := newGameRig(t)
	r.procs.add(100, 1, "prismlauncher.exe", r.play.Add(-time.Minute))
	r.procs.add(200, 100, "javaw.exe", r.play)
	r.log.write("[00:30:39] [main/INFO]: ModLauncher running\n" + render + "Backend library: LWJGL\n" +
		render + "Reloading ResourceManager\n" + render + "Sound engine started\n")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return r, ctx
}

func chapterCandidate(id, dir string) []AdoptCandidate {
	return []AdoptCandidate{{ChapterID: id, InstanceDir: dir}}
}

func TestAdoptFollowsARunningGameAndStopEndsOnlyItsJava(t *testing.T) {
	r, ctx := adoptRig(t)
	got := r.tracker.Adopt(ctx, adoptExe, chapterCandidate("frangfurd", r.log.dir))
	if !reflect.DeepEqual(got, []string{"frangfurd"}) {
		t.Fatalf("adopted %v", got)
	}
	// The first state the view hears is where the game is, not a start.
	if phases := r.phases(); len(phases) == 0 || phases[0] != models.GamePhaseRunning {
		t.Fatalf("phases = %v, want running first", phases)
	}
	if !r.tracker.Active("frangfurd") {
		t.Fatal("an adopted game does not hold Play")
	}
	if err := r.tracker.Stop("frangfurd"); err != nil {
		t.Fatal(err)
	}
	if terms := r.procs.terminated(); !reflect.DeepEqual(terms, []termCall{{200, false}}) {
		t.Fatalf("terminated %v, want the game's Java only", terms)
	}
	r.procs.end(200, 1)
	r.untilPhase(models.GamePhaseCrashed)
	if s := r.tracker.Latest("frangfurd"); s.Reason != models.GameFailStopped {
		t.Fatalf("reason = %q", s.Reason)
	}
	// Prism was not started by this launcher: never asked to close, never ended.
	time.Sleep(20 * time.Millisecond)
	if closes, terms := r.procs.closed(), r.procs.terminated(); len(closes) != 0 || len(terms) != 1 {
		t.Fatalf("closes %v, terms %v", closes, terms)
	}
	if c := r.tracker.console("frangfurd"); c != nil {
		t.Fatal("an adopted Prism has a console record, which quitting would close")
	}
}

func TestAdoptFollowsTheGameToItsEnd(t *testing.T) {
	r, ctx := adoptRig(t)
	if got := r.tracker.Adopt(ctx, adoptExe, chapterCandidate("frangfurd", r.log.dir)); len(got) != 1 {
		t.Fatalf("adopted %v", got)
	}
	r.log.append(render + "Stopping!\n")
	r.untilPhase(models.GamePhaseStopping)
	r.procs.end(200, 0)
	r.untilPhase(models.GamePhaseClosed)
	if d := r.tracker.Durations("frangfurd"); len(d) != 0 {
		t.Fatalf("an adopted run recorded launch times: %v", d)
	}
}

func TestAdoptLeavesWhatIsNotARunningGame(t *testing.T) {
	cases := map[string]func(r *gameRig){
		"the log says Stopping!":          func(r *gameRig) { r.log.append(render + "Stopping!\n") },
		"the log is older than the Java":  func(r *gameRig) { r.log.age(time.Hour) },
		"no log":                          func(r *gameRig) { removeOrFail(r.t, r.log.path) },
		"the Java is another Prism's":     func(r *gameRig) { r.procs.procs[1].PPID = 300 },
		"Prism runs from another place":   func(r *gameRig) { r.procs.images = map[int]string{100: "/Elsewhere/prismlauncher.exe"} },
		"the Java's start cannot be read": func(r *gameRig) { delete(r.procs.started, 200) },
		"process lookup is not available": func(r *gameRig) { r.procs.listErr = errGameProcUnsupported },
		"the chapter is already followed": func(r *gameRig) { r.start() },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r, ctx := adoptRig(t)
			change(r)
			if got := r.tracker.Adopt(ctx, adoptExe, chapterCandidate("frangfurd", r.log.dir)); len(got) != 0 {
				t.Fatalf("adopted %v", got)
			}
			if name != "the chapter is already followed" && r.procs.waited(200) {
				t.Fatal("a Java that was not adopted is waited on")
			}
		})
	}
}

func TestAdoptPicksTheChapterPrismLaunched(t *testing.T) {
	r, ctx := adoptRig(t)
	other := newGameLog(t)
	other.write("[00:30:39] [main/INFO]: ModLauncher running\n")
	chapters := []AdoptCandidate{{ChapterID: "lichdenstein", InstanceDir: other.dir}, {ChapterID: "frangfurd", InstanceDir: r.log.dir}}

	// Both logs fit and nothing says which one Prism launched: neither.
	if got := r.tracker.Adopt(ctx, adoptExe, chapters); len(got) != 0 {
		t.Fatalf("adopted %v from an ambiguous match", got)
	}
	writeLastLaunch(t, r.log.dir, r.play.Add(time.Second))
	writeLastLaunch(t, other.dir, r.play.Add(-time.Hour))
	if got := r.tracker.Adopt(ctx, adoptExe, chapters); !reflect.DeepEqual(got, []string{"frangfurd"}) {
		t.Fatalf("adopted %v, want the chapter launched at the Java's start", got)
	}
}

func writeLastLaunch(t *testing.T, dir string, at time.Time) {
	t.Helper()
	cfg := fmt.Sprintf("[General]\nlastLaunchTime=%d\ntotalTimePlayed=60\n", at.UnixMilli())
	if err := os.WriteFile(filepath.Join(dir, "instance.cfg"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
}

func removeOrFail(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

// waited is whether anything has waited on pid.
func (f *fakeProcs) waited(pid int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.exits[pid]
	return ok
}
