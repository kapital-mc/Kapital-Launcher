package services

import (
	"errors"
	"reflect"
	"testing"

	"kapital/backend/models"
)

func TestOnMacOSTheCardCoversTheGameFromItsJavaUntilTheHandover(t *testing.T) {
	f := newCardFixture("darwin")
	f.activateOK = true
	f.begin(t)
	if !f.card.HoldsGameWindow() {
		t.Fatal("macOS keeps the game out of sight, with the cover")
	}
	hold := f.card.GameHold("frangfurd")
	if hold == nil {
		t.Fatal("macOS gives the tracker the card's cover as its hold")
	}
	if f.host().Covered() {
		t.Fatal("no cover before the game starts: Prism's dialogs must stay reachable")
	}

	holder, err := hold(200)
	if err != nil {
		t.Fatal(err)
	}
	if !f.host().Covered() {
		t.Fatalf("the game's Java raises the cover: %v", f.host().Calls())
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
	for _, call := range f.host().Calls() {
		if call == "cover on" {
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
