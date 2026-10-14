package services

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"kapital/backend/design"
	"kapital/backend/models"
)

func TestOnMacOSTheCardCoversTheScreenFromPlayUntilTheHandover(t *testing.T) {
	f := newCardFixture("darwin")
	f.activateOK = true
	f.begin(t)
	if !f.card.HoldsGameWindow() {
		t.Fatal("macOS keeps the game out of sight, with the cover")
	}
	// Prism's "Please wait" dialogs come before the game's Java and float above
	// ordinary windows: the cover is up from Play (seen on the iMac, 2026-10-15).
	if !f.host().Covered() {
		t.Fatalf("the cover goes up with the card: %v", f.host().Calls())
	}
	hold := f.card.GameHold("frangfurd")
	if hold == nil {
		t.Fatal("macOS gives the tracker the card's cover as its hold")
	}
	holder, err := hold(200)
	if err != nil {
		t.Fatal(err)
	}
	if !f.host().Covered() {
		t.Fatalf("the game's Java keeps the cover up: %v", f.host().Calls())
	}
	// The game's window being made no longer closes the card on macOS.
	for _, phase := range []string{models.GamePhaseMods, models.GamePhaseWindow, models.GamePhaseResources} {
		if !f.card.Observe(game(phase)) {
			t.Fatalf("the card shows at %s", phase)
		}
	}
	if f.host().Closes() != 0 || !f.host().Covered() {
		t.Fatalf("the card stays covering until the handover: %v", f.host().Calls())
	}

	// The tracker's handover: the cover goes and the game comes to the front,
	// then the card closes.
	rep := holder.Release(true)
	if f.host().Covered() || !reflect.DeepEqual(f.activated, []int{200}) || !rep.Foreground {
		t.Fatalf("the handover lowers the cover and activates the game: calls %v, activated %v, report %+v",
			f.host().Calls(), f.activated, rep)
	}
	f.card.Handover("frangfurd")
	if f.host().Closes() != 1 || f.card.Showing("frangfurd") {
		t.Fatalf("the card closes at the handover: %v", f.host().Calls())
	}
	if f.unminimises() != 0 {
		t.Fatal("the launcher waits for the game to end")
	}
	if again := holder.Release(true); again != rep || len(f.activated) != 1 {
		t.Fatalf("a second release repeats nothing: %+v, activated %v", again, f.activated)
	}
	f.card.Observe(game(models.GamePhaseClosed))
	if f.unminimises() != 1 {
		t.Fatalf("the launcher comes back: %v", f.launcher.calls)
	}
}

func TestOnMacOSTheCardIsCentredOnTheScreenNotTheLauncher(t *testing.T) {
	f := newCardFixture("darwin")
	f.begin(t)
	opens := f.host().Opens()
	if len(opens) != 1 || !opens[0].OnScreen || opens[0].W != design.SplashWidth || opens[0].H != design.SplashHeight {
		t.Fatalf("the card asks for the screen's middle: %+v", opens)
	}
	w := newCardFixture("windows")
	w.begin(t)
	if got := w.host().Opens(); len(got) != 1 || got[0].OnScreen {
		t.Fatalf("Windows still centres on the launcher: %+v", got)
	}
}

// coverFixture is a macOS card whose wait for the game is short enough to see.
func coverFixture(t *testing.T, wait time.Duration) *cardFixture {
	t.Helper()
	f := newCardFixture("darwin")
	f.card.cfg.CoverWait = wait
	f.begin(t)
	return f
}

// eventually waits up to two seconds for cond.
func eventually(t *testing.T, cond func() bool) bool {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if cond() {
			return true
		}
	}
	return cond()
}

func TestOnMacOSACoverWithNoGameInTimeIsLoweredAndTheJavaRaisesItAgain(t *testing.T) {
	f := coverFixture(t, 20*time.Millisecond)
	if !eventually(t, func() bool { return !f.host().Covered() }) {
		t.Fatalf("no game in time lowers the cover, so Prism can be reached: %v", f.host().Calls())
	}
	if f.host().Closes() != 0 || !f.card.Showing("frangfurd") {
		t.Fatal("the card stays while the cover is down")
	}
	if _, err := f.card.GameHold("frangfurd")(200); err != nil {
		t.Fatal(err)
	}
	if !f.host().Covered() {
		t.Fatalf("the game's Java raises the cover again: %v", f.host().Calls())
	}
}

func TestOnMacOSACoverTheJavaReachedInTimeIsNotLoweredByTheWait(t *testing.T) {
	f := coverFixture(t, 30*time.Millisecond)
	if _, err := f.card.GameHold("frangfurd")(200); err != nil {
		t.Fatal(err)
	}
	time.Sleep(120 * time.Millisecond)
	if !f.host().Covered() {
		t.Fatalf("the wait leaves a cover the game holds: %v", f.host().Calls())
	}
}

func TestOnMacOSARunThatEndsBeforeTheJavaLowersTheCoverAndKeepsTheCard(t *testing.T) {
	f := newCardFixture("darwin")
	f.begin(t)
	failed := game(models.GamePhaseFailed)
	failed.Reason = models.GameFailPackSync
	if !f.card.Observe(failed) {
		t.Fatal("the card stays for the failure")
	}
	if f.host().Covered() || f.host().Closes() != 0 {
		t.Fatalf("the cover goes so the error and Prism can be reached, the card stays: %v", f.host().Calls())
	}
}

func TestOnMacOSTheHandoverKeepsTheCoverForItsDelay(t *testing.T) {
	f := newCardFixture("darwin")
	f.card.cfg.HandoverDelay = 80 * time.Millisecond
	f.activateOK = true
	f.begin(t)
	holder, err := f.card.GameHold("frangfurd")(200)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan WindowReport)
	start := time.Now()
	go func() { done <- holder.Release(true) }()
	time.Sleep(20 * time.Millisecond)
	if !f.host().Covered() {
		t.Fatal("the cover stays while the game draws its first own frame")
	}
	<-done
	if elapsed := time.Since(start); elapsed < 80*time.Millisecond {
		t.Fatalf("released after %v, before the delay", elapsed)
	}
	if f.host().Covered() || len(f.activated) != 1 {
		t.Fatalf("then the cover goes and the game is activated: %v, %v", f.host().Calls(), f.activated)
	}
	// A release because the run ended does not wait.
	g := newCardFixture("darwin")
	g.card.cfg.HandoverDelay = time.Hour
	g.begin(t)
	h, err := g.card.GameHold("frangfurd")(200)
	if err != nil {
		t.Fatal(err)
	}
	h.Release(false)
	if g.host().Covered() {
		t.Fatal("an ended run lowers the cover at once")
	}
}

func TestOnMacOSARunThatEndsBeforeTheHandoverLowersTheCardAndKeepsIt(t *testing.T) {
	f := newCardFixture("darwin")
	f.begin(t)
	holder, err := f.card.GameHold("frangfurd")(200)
	if err != nil {
		t.Fatal(err)
	}
	f.card.Observe(game(models.GamePhaseMods))

	// The tracker releases without the foreground before it emits the end.
	holder.Release(false)
	crashed := game(models.GamePhaseCrashed)
	if !f.card.Observe(crashed) {
		t.Fatal("the card stays for the crash")
	}
	if f.host().Covered() || f.host().Closes() != 0 || len(f.activated) != 0 {
		t.Fatalf("the cover goes, the card stays, nothing is activated: calls %v, activated %v",
			f.host().Calls(), f.activated)
	}
}

func TestTheCoverIsNotRaisedForACardThatIsGone(t *testing.T) {
	f := newCardFixture("darwin")
	f.begin(t)
	hold := f.card.GameHold("frangfurd")
	if _, ok := f.card.Leave(); !ok {
		t.Fatal("the card was up")
	}
	if _, err := hold(200); !errors.Is(err, errNoCardToCover) {
		t.Fatalf("a card the player left is not raised: %v", err)
	}
	if _, err := newCardFixture("darwin").card.GameHold("frangfurd")(200); !errors.Is(err, errNoCardToCover) {
		t.Fatalf("no card, no cover: %v", err)
	}
	// The cover went up with the card at Play; nothing raises it after the
	// card was left and closed.
	left := false
	for _, call := range f.host().Calls() {
		if call == "close" {
			left = true
		}
		if left && call == "cover on" {
			t.Fatalf("raised after it was left: %v", f.host().Calls())
		}
	}
}

func TestOnlyMacOSGivesTheTrackerAHoldOfTheCards(t *testing.T) {
	for goos, want := range map[string]bool{"darwin": true, "windows": false, "linux": false} {
		if got := newCardFixture(goos).card.GameHold("frangfurd") != nil; got != want {
			t.Errorf("%s: card hold %v, want %v", goos, got, want)
		}
	}
	if newCardFixture("linux").card.HoldsGameWindow() {
		t.Error("an OS with no card holds nothing")
	}
}
