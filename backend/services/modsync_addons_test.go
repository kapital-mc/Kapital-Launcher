package services

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"kapital/backend/models"
)

// disabledAll is names with ".disabled" after each.
func disabledAll(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n + ".disabled"
	}
	return out
}

// sortedNames is the names sorted as modsOnDisk sorts them.
func sortedNames(names ...string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return out
}

func cameraOn(disabled []string) models.AppSettings {
	return models.AppSettings{
		Theme:         "dark",
		DisabledMods:  map[string][]string{"frangfurd": disabled},
		EnabledAddons: map[string][]string{"frangfurd": {"camera"}},
	}
}

// Off is every add-on's default: a player who never turned it on gets one run,
// the pack's, and nothing of the add-on.
func TestSyncLeavesAnAddonThatIsOffUndownloaded(t *testing.T) {
	f := newSyncFixture(t, nil, jade)
	if code := f.run(); code != 0 || len(f.runs) != 1 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
	if slices.Contains(f.runs[0], "--meta-file") {
		t.Fatalf("the pack's run names a record: %q", f.runs[0])
	}
}

// An add-on turned off after it was synced keeps its jars, put away, and is not
// synced again.
func TestSyncPutsAwayTheJarsOfAnAddonTurnedOff(t *testing.T) {
	f := newSyncFixture(t, []string{}, append([]string{jade}, cameraJars...)...)
	writeCameraRecord(t, f.game)
	if code := f.run(); code != 0 || len(f.runs) != 1 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
	want := sortedNames(append([]string{jade}, disabledAll(cameraJars)...)...)
	if got := modsOnDisk(t, f.game); !reflect.DeepEqual(got, want) {
		t.Fatalf("mods %q, want %q", got, want)
	}
	if !strings.Contains(f.say(), `add-on "camera" is off, 4 jars put away`) {
		t.Fatalf("lines:\n%s", f.say())
	}
}

// On: the add-on's jars are back before either run, its own run follows the
// pack's with its own record, and afterwards the jars stay, except one the
// player switched off in the advanced list.
func TestSyncRunsAnAddonThatIsOnAfterThePack(t *testing.T) {
	keptOff := cameraJars[1]
	f := newSyncFixture(t, nil, append([]string{jade}, disabledAll(cameraJars)...)...)
	f.settings(cameraOn([]string{keptOff}))
	writeCameraRecord(t, f.game)
	var seen [][]string
	f.during = func(game string) (int, error) {
		seen = append(seen, modsOnDisk(t, game))
		return 0, nil
	}
	if code := f.run(); code != 0 {
		t.Fatalf("exit %d\n%s", code, f.say())
	}
	if len(f.runs) != 2 {
		t.Fatalf("%d runs\n%s", len(f.runs), f.say())
	}
	pack := append([]string{f.java, f.game}, packwizSyncArgs(f.game, testPackURL)...)
	addon := append([]string{f.java, f.game}, packwizAddonSyncArgs(f.game, cameraPackURL, "kapital-addon-camera.json")...)
	if !reflect.DeepEqual(f.runs[0], pack) || !reflect.DeepEqual(f.runs[1], addon) {
		t.Fatalf("runs %q", f.runs)
	}
	// Both runs saw every add-on jar as a jar, so neither downloaded one again.
	all := sortedNames(append([]string{jade}, cameraJars...)...)
	for i, s := range seen {
		if !reflect.DeepEqual(s, all) {
			t.Fatalf("run %d saw %q, want %q", i, s, all)
		}
	}
	want := sortedNames(jade, cameraJars[0], keptOff+".disabled", cameraJars[2], cameraJars[3])
	if got := modsOnDisk(t, f.game); !reflect.DeepEqual(got, want) {
		t.Fatalf("after: %q, want %q", got, want)
	}
}

// The first time on, there is no record: the add-on's run is what downloads it.
func TestSyncInstallsAnAddonTurnedOnBeforeItsFirstSync(t *testing.T) {
	f := newSyncFixture(t, nil, jade)
	f.settings(cameraOn(nil))
	if code := f.run(); code != 0 || len(f.runs) != 2 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
	if !slices.Contains(f.runs[1], "--meta-file") || f.runs[1][len(f.runs[1])-1] != cameraPackURL {
		t.Fatalf("add-on run %q", f.runs[1])
	}
}

func TestSyncSkipsTheAddonWhenThePackFails(t *testing.T) {
	f := newSyncFixture(t, nil, jade)
	f.settings(cameraOn(nil))
	f.during = func(string) (int, error) { return 3, nil }
	if code := f.run(); code != 3 || len(f.runs) != 1 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
	if !strings.Contains(f.say(), `add-on "camera" waits for the pack sync to succeed`) {
		t.Fatalf("lines:\n%s", f.say())
	}
}

func TestSyncExitsWithTheAddonsCodeWhenOnlyItFails(t *testing.T) {
	f := newSyncFixture(t, nil, jade)
	f.settings(cameraOn(nil))
	runs := 0
	f.during = func(string) (int, error) {
		runs++
		if runs == 2 {
			return 5, nil
		}
		return 0, nil
	}
	if code := f.run(); code != 5 || len(f.runs) != 2 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
}

// Settings that cannot be read are every add-on off: its jars are put away.
func TestSyncTreatsUnreadableSettingsAsEveryAddonOff(t *testing.T) {
	f := newSyncFixture(t, nil, cameraJars...)
	f.rawSettings("{")
	writeCameraRecord(t, f.game)
	if code := f.run(); code != 0 || len(f.runs) != 1 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
	if got := modsOnDisk(t, f.game); !reflect.DeepEqual(got, sortedNames(disabledAll(cameraJars)...)) {
		t.Fatalf("mods %q", got)
	}
}

func TestSyncRunsNoAddonForAForeignInstance(t *testing.T) {
	f := newSyncFixture(t, nil, jade)
	f.settings(cameraOn(nil))
	f.env["INST_ID"] = "my-own-instance"
	if code := f.run(); code != 0 || len(f.runs) != 1 {
		t.Fatalf("exit %d, %d runs\n%s", code, len(f.runs), f.say())
	}
}
