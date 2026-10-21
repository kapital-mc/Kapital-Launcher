package services

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// What Prism started for this launch is everything below it that began after
// Play, at any depth. A child older than Play, or whose start cannot be read,
// is not, and neither is anything below it.
func TestLaunchDescendantsAreWhatPrismStartedSincePlay(t *testing.T) {
	play := time.Now()
	started := map[int]time.Time{
		300: play.Add(time.Second),     // the pack sync
		301: play.Add(2 * time.Second), // its Java
		302: play.Add(-time.Minute),    // older than Play
		303: play.Add(3 * time.Second), // below the older one
		305: play.Add(time.Second),     // another program's
	}
	procs := []procInfo{
		{PID: 100, PPID: 1, Name: "prismlauncher.exe"},
		{PID: 300, PPID: 100, Name: "KapitalLauncher.exe"},
		{PID: 301, PPID: 300, Name: "javaw.exe"},
		{PID: 302, PPID: 100, Name: "conhost.exe"},
		{PID: 303, PPID: 302, Name: "javaw.exe"},
		{PID: 304, PPID: 100, Name: "javaw.exe"}, // start unreadable
		{PID: 305, PPID: 1, Name: "javaw.exe"},
	}
	lookup := func(pid int) (time.Time, bool) {
		at, ok := started[pid]
		return at, ok
	}
	if got := launchDescendants(procs, 100, play, lookup); !reflect.DeepEqual(got, []int{300, 301}) {
		t.Fatalf("got %v", got)
	}
}

// A Stop during the pack download ends the sync and its Java, which outlive
// Prism, and asks Prism to close.
func TestStopDuringTheDownloadEndsThePackSyncAndItsJava(t *testing.T) {
	r := newGameRig(t)
	r.procs.add(100, 1, "prismlauncher.exe", r.play)
	r.procs.add(300, 100, "KapitalLauncher.exe", r.play.Add(time.Second))
	r.procs.add(301, 300, "javaw.exe", r.play.Add(2*time.Second))
	r.start()
	if err := r.tracker.Stop("frangfurd"); err != nil {
		t.Fatal(err)
	}
	if got := r.procs.terminated(); !reflect.DeepEqual(got, []termCall{{300, true}, {301, true}}) {
		t.Fatalf("the sync and its Java, by pid: %v", got)
	}
	if got := r.procs.closed(); !reflect.DeepEqual(got, []int{100}) {
		t.Fatalf("Prism is asked to close: %v", got)
	}
	close(r.prism)
	r.wantEnded("failed")
}

// A Prism that is open takes the launch, so Play asks an idle one to close and
// waits for it. A busy one, with a game or a download under it, and the Prism
// of a run the launcher follows, are left alone; nothing is ended.
func TestCloseIdlePrismsAsksOnlyIdlePrismsToClose(t *testing.T) {
	r := newGameRig(t)
	r.tracker.stopForce = 50 * time.Millisecond
	exe := prismExe(t)
	r.procs.images = map[int]string{}
	for _, pid := range []int{100, 500, 600} {
		r.procs.add(pid, 1, "prismlauncher.exe", r.play.Add(-time.Hour))
		r.procs.images[pid] = exe
	}
	r.procs.add(601, 600, "javaw.exe", r.play)
	r.start() // the run's Prism is 100

	go r.procs.end(500, 0)
	if open := r.tracker.CloseIdlePrisms(exe); !open {
		t.Fatal("the busy Prism is still open")
	}
	if got := r.procs.closed(); !reflect.DeepEqual(got, []int{500}) {
		t.Fatalf("only the idle Prism is asked: %v", got)
	}
	if got := r.procs.terminated(); len(got) != 0 {
		t.Fatalf("nothing is ended: %v", got)
	}
}

// One that does not close in time is still open, and is not ended.
func TestCloseIdlePrismsSaysWhenOneStays(t *testing.T) {
	r := newGameRig(t)
	r.tracker.stopForce = 20 * time.Millisecond
	exe := prismExe(t)
	r.procs.add(500, 1, "prismlauncher.exe", r.play)
	r.procs.images = map[int]string{500: exe}
	if open := r.tracker.CloseIdlePrisms(exe); !open {
		t.Fatal("it stayed")
	}
	if !slices.Equal(r.procs.closed(), []int{500}) || len(r.procs.terminated()) != 0 {
		t.Fatalf("asked, never ended: %v %v", r.procs.closed(), r.procs.terminated())
	}
	if open := r.tracker.CloseIdlePrisms(""); open {
		t.Fatal("no executable, nothing to close")
	}
}
