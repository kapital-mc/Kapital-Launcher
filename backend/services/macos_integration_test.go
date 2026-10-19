//go:build darwin && macintegration

package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
)

// Checks on a real Mac that the unit tests cannot make (#30): the managed
// Prism installed from its real release and verified by codesign, the logging
// rules copied out of its bundle, its --version read through detection, the
// data roots Prism itself creates, and the launcher's sync copy run from outside
// its bundle. The macOS workflow (.github/workflows/macos.yml) runs them on an
// Apple Silicon and an Intel runner. They download Prism from GitHub and start
// it, so they sit behind the macintegration tag and outside `go test ./...`.
//
// Inputs, from the workflow:
//
//	KAPITAL_IT_PRISM_VERSION  the release tag, e.g. 11.1.1
//	KAPITAL_IT_PRISM_DIGEST   sha256:<hex> of the macOS zip, as GitHub lists it
//	KAPITAL_IT_PRISM_SIZE     the zip's size in bytes
//	KAPITAL_IT_APP            the built "Kapital Launcher.app" (sync copy test)
//	KAPITAL_IT_DISPOSABLE     1 on a runner: the tests may write to the home
//	                          folder (~/Applications, Prism's default root)

func itEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skip(name + " is not set")
	}
	return v
}

func itDisposable(t *testing.T) {
	t.Helper()
	if os.Getenv("KAPITAL_IT_DISPOSABLE") != "1" {
		t.Skip("writes to the home folder; only on a disposable runner")
	}
}

// waitForFile polls for a path to exist, the way a person would watch Finder.
func waitForFile(path string, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// startAndStop starts Prism with args, waits until want exists, and ends it
// with SIGTERM, as Stop does. It reports whether want appeared.
func startAndStop(t *testing.T, exe string, want string, args ...string) bool {
	t.Helper()
	return startRunStop(t, exe, want, nil, args...)
}

// startRunStop is startAndStop with during called while Prism is up, once want
// exists.
func startRunStop(t *testing.T, exe string, want string, during func(), args ...string) bool {
	t.Helper()
	cmd := exec.Command(exe, args...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	seen := waitForFile(want, 90*time.Second)
	if seen && during != nil {
		during()
	}
	if err := terminateProcess(cmd.Process.Pid, false); err != nil {
		t.Error(err)
	}
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Error("Prism did not quit on SIGTERM within 20 s; killing it")
		if err := terminateProcess(cmd.Process.Pid, true); err != nil {
			t.Error(err)
		}
		<-done
	}
	return seen
}

func TestMacManagedPrism(t *testing.T) {
	version := itEnv(t, "KAPITAL_IT_PRISM_VERSION")
	digest := itEnv(t, "KAPITAL_IT_PRISM_DIGEST")
	size, err := strconv.ParseInt(itEnv(t, "KAPITAL_IT_PRISM_SIZE"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := portableAsset(runtime.GOOS, runtime.GOARCH, version)
	if err != nil {
		t.Fatal(err)
	}
	// The real data dir has a space in it (~/Library/Application Support).
	dataDir := filepath.Join(t.TempDir(), "Application Support", "KapitalLauncher")
	m := NewManagedPrism(dataDir, runtime.GOOS, runtime.GOARCH)
	rel := models.PrismRelease{
		Version: version, Asset: asset, URL: prismDownloadBase + version + "/" + asset,
		Size: size, Digest: strings.ToLower(digest),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var phases []string
	err = m.Install(ctx, rel, func(p models.PrismInstallProgress) {
		if len(phases) == 0 || phases[len(phases)-1] != p.Phase {
			phases = append(phases, p.Phase)
		}
	})
	if err != nil {
		t.Fatalf("install: %v (phases %v)", err, phases)
	}
	t.Logf("install phases: %v", phases)
	if !strings.Contains(strings.Join(phases, ","), "verifying") {
		t.Fatalf("codesign never ran: phases %v", phases)
	}

	got, exe, ok := m.Installed()
	if !ok || got != version {
		t.Fatalf("installed %q (ok %v), want %q", got, ok, version)
	}
	info, err := os.Stat(exe)
	if err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("executable %s: %v, mode %v", exe, err, info)
	}
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe)))

	t.Run("signature", func(t *testing.T) {
		// What verifySignature ran, said again with the detail the handover's
		// open question needs: who signed it, and its Team ID for a -R
		// requirement.
		out, err := exec.Command("/usr/bin/codesign", "-dv", "--verbose=2", app).CombinedOutput()
		if err != nil {
			t.Fatalf("codesign -dv: %v: %s", err, out)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "Authority=") || strings.HasPrefix(line, "TeamIdentifier=") ||
				strings.HasPrefix(line, "Identifier=") || strings.HasPrefix(line, "Format=") ||
				strings.HasPrefix(line, "Runtime Version=") {
				t.Log(line)
			}
		}
		// Prism's own designated requirement, beside the one the install holds
		// it to (verify_darwin.go, #226).
		if dr, err := exec.Command("/usr/bin/codesign", "-d", "-r-", app).CombinedOutput(); err == nil {
			t.Logf("designated requirement: %s", strings.TrimSpace(string(dr)))
		}
		if err := codesignVerify(context.Background(), app, prismRequirement); err != nil {
			t.Errorf("Prism does not meet the launcher's requirement: %v", err)
		}
		if !strings.Contains(string(out), "Authority=Developer ID Application") {
			t.Errorf("not Developer ID signed:\n%s", out)
		}
		if out, err := exec.Command("/usr/sbin/spctl", "--assess", "--type", "execute", "-vv", app).CombinedOutput(); err != nil {
			t.Errorf("Gatekeeper refuses the managed Prism: %v: %s", err, out)
		} else {
			t.Logf("spctl: %s", strings.TrimSpace(string(out)))
		}
	})

	t.Run("no quarantine", func(t *testing.T) {
		// The launcher downloads with Go's HTTP client, which sets no
		// quarantine attribute, so Gatekeeper has nothing to ask on first open.
		if out, err := exec.Command("/usr/bin/xattr", "-p", "com.apple.quarantine", app).CombinedOutput(); err == nil {
			t.Errorf("the bundle is quarantined: %s", out)
		}
	})

	t.Run("seeded root", func(t *testing.T) {
		for _, name := range []string{"prismlauncher.cfg", "prismlauncher_update.cfg"} {
			if _, err := os.Stat(filepath.Join(m.Root(), name)); err != nil {
				t.Error(err)
			}
		}
		rules, err := os.ReadFile(filepath.Join(m.Root(), prismLogRulesFile))
		if err != nil {
			t.Fatalf("qtlogging.ini was not copied out of Contents/Resources: %v", err)
		}
		if !strings.Contains(string(rules), "launcher.task.critical=true") {
			t.Fatalf("qtlogging.ini lacks the launcher's rule:\n%s", rules)
		}
	})

	t.Run("sparkle keys", func(t *testing.T) {
		// On macOS Prism updates through Sparkle, which reads its own keys, not
		// prismlauncher_update.cfg (ADR-11). Logged for the decision.
		plist := filepath.Join(app, "Contents", "Info.plist")
		for _, key := range []string{"SUFeedURL", "SUEnableAutomaticChecks", "SUAutomaticallyUpdate", "SUPublicEDKey"} {
			out, err := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :"+key, plist).CombinedOutput()
			if err != nil {
				t.Logf("Info.plist %s: absent", key)
				continue
			}
			t.Logf("Info.plist %s = %s", key, strings.TrimSpace(string(out)))
		}
	})

	t.Run("version through detection", func(t *testing.T) {
		p := NewPrismService("darwin")
		engine := p.Detect(ctx, models.AppSettings{PrismExecutable: ResolvePrismExecutable("darwin", app)})
		if !engine.Found || engine.Source != "settings" {
			t.Fatalf("detect from the picked bundle: %+v", engine)
		}
		if engine.Version != version {
			out, err := exec.Command(exe, "--version").CombinedOutput()
			t.Fatalf("version %q, want %q; prismlauncher --version said %q (%v)", engine.Version, version, out, err)
		}
	})

	t.Run("standard location", func(t *testing.T) {
		itDisposable(t)
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(home, "Applications", "Prism Launcher.app")
		if _, err := os.Stat(dest); err == nil {
			t.Skip(dest + " already exists")
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("/usr/bin/ditto", app, dest).CombinedOutput(); err != nil {
			t.Fatalf("ditto: %v: %s", err, out)
		}
		defer func() {
			if err := os.RemoveAll(dest); err != nil {
				t.Error(err)
			}
		}()
		engine := NewPrismService("darwin").Detect(ctx, models.AppSettings{})
		want := filepath.Join(dest, "Contents", "MacOS", "prismlauncher")
		if !engine.Found || engine.Executable != want || engine.Source != "standard-location" {
			t.Fatalf("detect in ~/Applications: %+v, want %s", engine, want)
		}
		if engine.Version != version {
			t.Fatalf("version %q, want %q", engine.Version, version)
		}
	})

	t.Run("managed root through --dir", func(t *testing.T) {
		// The launch passes the managed root with --dir; Prism's own log in
		// that root is what the tracker follows (#106).
		log := filepath.Join(m.Root(), "logs", "PrismLauncher-0.log")
		// While it runs, the look an update takes before it removes the
		// previous version's folder sees it, by name and image path (#227).
		seen := func() {
			open, err := NewGameTracker(t.TempDir(), nil).OtherPrismOpen("", exe)
			if err != nil || !open {
				t.Errorf("a running managed Prism is not seen as open: %v, %v", open, err)
			}
		}
		if !startRunStop(t, exe, log, seen, "--dir", m.Root()) {
			t.Fatalf("Prism started with --dir wrote no %s", log)
		}
	})

	t.Run("default data root", func(t *testing.T) {
		itDisposable(t)
		p := NewPrismService("darwin")
		root := p.DataRoot(models.AppSettings{}, models.EngineInfo{Found: true, Executable: exe, Source: "settings"})
		if root == "" {
			t.Fatal("no default root on darwin")
		}
		if _, err := os.Stat(root); err == nil {
			t.Skip(root + " already exists")
		}
		defer func() {
			if err := os.RemoveAll(root); err != nil {
				t.Error(err)
			}
		}()
		if !startAndStop(t, exe, filepath.Join(root, "logs")) {
			entries, err := os.ReadDir(filepath.Dir(root))
			names := []string{}
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("Prism started with no --dir created no %s; %s holds %v (%v)", root, filepath.Dir(root), names, err)
		}
	})

	t.Run("self-updated copy", func(t *testing.T) {
		// What Sparkle leaves (#221): managed.json names an older version
		// whose folder now holds the release. Made by renaming the folder and
		// rewriting the record, then put back.
		const older = "11.0.0"
		placed, olderDir := m.appDir(version), m.appDir(older)
		record := filepath.Join(m.dir, "managed.json")
		saved, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(placed, olderDir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Rename(olderDir, placed); err != nil {
				t.Error(err)
			}
			if err := os.WriteFile(record, saved, 0o644); err != nil {
				t.Error(err)
			}
		}()
		if err := os.WriteFile(record, []byte(`{"version":"`+older+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, olderExe, ok := m.Installed()
		if !ok {
			t.Fatal("the renamed copy is not seen as installed")
		}
		engine := NewPrismService("darwin").Detect(ctx, models.AppSettings{PrismExecutable: olderExe})
		if engine.Version != version {
			t.Fatalf("detection read %q from the renamed copy, want %q", engine.Version, version)
		}
		latest := models.PrismRelease{Version: version, Installed: older, UpdateAvailable: true}
		m.Reconcile(ctx, &latest, engine.Version)
		if latest.Installed != version || latest.UpdateAvailable {
			t.Fatalf("a self-updated copy at the latest release: %+v (it should verify and need nothing)", latest)
		}
	})
}

func TestMacSyncCopyRunsOutsideTheBundle(t *testing.T) {
	app := itEnv(t, "KAPITAL_IT_APP")
	exe := filepath.Join(app, "Contents", "MacOS", "kapital-launcher")
	if _, err := os.Stat(exe); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(t.TempDir(), "Application Support", "KapitalLauncher")
	sc := NewSyncCopyFrom(dataDir, "1.0.0", func() (string, error) { return exe, nil })
	copyPath, err := sc.Ensure()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(copyPath, " ") {
		t.Fatalf("the copy's path %q should carry the space the real one does", copyPath)
	}

	// Wails signs the bundle ad hoc, so the executable's signature also seals
	// the bundle's Info.plist and resources, and codesign refuses the copy on
	// its own ("invalid Info.plist", seen on the first run of this test). What
	// matters is that it still runs: the kernel checks the code pages, which
	// the copy keeps. Logged, and the runs below are the check.
	out, err := exec.Command("/usr/bin/codesign", "--verify", "--verbose", copyPath).CombinedOutput()
	t.Logf("codesign on the copy: %v: %s", err, strings.TrimSpace(string(out)))

	// Prism splits its pre-launch command on spaces outside double quotes; the
	// launcher's own reader must get the same path back.
	cmdLine := preLaunchCommand(copyPath, "https://example.invalid/pack.toml")
	if got, _, ok := parseSyncCommand(cmdLine); !ok || got != copyPath {
		t.Fatalf("parse %q: %q, %v", cmdLine, got, ok)
	}

	run := func(args ...string) (int, string) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, copyPath, args...)
		cmd.Env = []string{"HOME=" + os.Getenv("HOME"), "PATH=/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		var exit *exec.ExitError
		switch {
		case err == nil:
			return 0, string(out)
		case errors.As(err, &exit) && exit.ExitCode() >= 0:
			return exit.ExitCode(), string(out)
		default:
			// A signal (SIGKILL for a bad signature, SIGABRT from dyld) lands here.
			t.Fatalf("the copy did not run to an exit: %v\n%s", err, out)
			return 0, ""
		}
	}
	// No URL: the sync's own usage line, before anything of the app starts.
	if code, out := run(SyncFlag); code != SyncExitRefused || !strings.Contains(out, "usage:") {
		t.Fatalf("exit %d, output %q", code, out)
	}
	// A URL but none of Prism's INST_ variables: refused, still cleanly.
	if code, out := run(SyncFlag, "https://kapital-packs.alessandrogekeler.workers.dev/frangfurd/pack.toml"); code != SyncExitRefused || !strings.Contains(out, "refused") {
		t.Fatalf("exit %d, output %q", code, out)
	}
}

// TestMacLauncherStartsWithItsMenus starts the built bundle the way Finder
// would leave it running, checks that it stays up, and reads its menu bar
// through System Events (#225): App, Edit and Window, which carry the Cmd
// shortcuts. Reading another process's menus needs the Accessibility
// permission; where the runner has not granted it to osascript the menus are
// logged as unread and only the start is checked. It ends with the menu's own
// Quit when the menus could be read, else SIGTERM.
func TestMacLauncherStartsWithItsMenus(t *testing.T) {
	app := itEnv(t, "KAPITAL_IT_APP")
	itDisposable(t) // the app writes its settings and log to the real data dir
	exe := filepath.Join(app, "Contents", "MacOS", "kapital-launcher")
	var stderr strings.Builder
	cmd := exec.Command(exe)
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	ended := false
	defer func() {
		if ended {
			return
		}
		if err := terminateProcess(cmd.Process.Pid, true); err != nil {
			t.Log(err)
		}
		<-done
	}()

	// Up for 15 s: long enough for the window, the webview and the startup's
	// first reads to have run.
	select {
	case err := <-done:
		ended = true
		t.Fatalf("the launcher exited at start: %v\n%s", err, stderr.String())
	case <-time.After(15 * time.Second):
	}
	if !strings.Contains(stderr.String(), "msg=starting") {
		t.Errorf("no start line in the launcher's log:\n%s", stderr.String())
	}

	pid := strconv.Itoa(cmd.Process.Pid)
	script := `tell application "System Events" to get name of every menu bar item of menu bar 1 of (first process whose unix id is ` + pid + `)`
	out, err := exec.Command("/usr/bin/osascript", "-e", script).CombinedOutput()
	if err != nil {
		t.Logf("menus unread, osascript: %v: %s", err, strings.TrimSpace(string(out)))
		return
	}
	menus := strings.TrimSpace(string(out))
	t.Logf("menu bar: %s", menus)
	checkWindowChrome(t, pid, stderr.String())
	for _, want := range []string{"Edit", "Window"} {
		if !strings.Contains(menus, want) {
			t.Errorf("menu bar %q has no %s menu", menus, want)
		}
	}

	// Cmd+Q's own item, so a quit through the menu is seen to end the app.
	quit := `tell application "System Events" to tell (first process whose unix id is ` + pid + `) to click (first menu item of menu 1 of menu bar item 2 of menu bar 1 whose name starts with "Quit")`
	if out, err := exec.Command("/usr/bin/osascript", "-e", quit).CombinedOutput(); err != nil {
		t.Fatalf("click Quit: %v: %s", err, out)
	}
	select {
	case <-done:
		ended = true
	case <-time.After(20 * time.Second):
		t.Error("the launcher did not quit from its Quit menu item within 20 s")
	}
}

// windowChromeScript lists the launcher's first window and its title bar
// buttons as "<subrole> x y w h" lines, in screen points.
const windowChromeScript = `on box(e)
	tell application "System Events"
		set p to position of e
		set s to size of e
	end tell
	return (item 1 of p as string) & " " & (item 2 of p as string) & " " & (item 1 of s as string) & " " & (item 2 of s as string)
end box
tell application "System Events" to tell (first process whose unix id is %s)
	set frontmost to true
	set w to window 1
	set out to "window " & my box(w)
	repeat with b in (buttons of w)
		set out to out & linefeed & (subrole of b as string) & " " & my box(b)
	end repeat
	return out
end tell`

// clickScript posts a real left click at a screen point, so it goes through
// AppKit's hit testing as a person's would (System Events' own click presses an
// accessibility element instead).
const clickScript = `ObjC.import('CoreGraphics')
function run(argv) {
	const p = { x: Number(argv[0]), y: Number(argv[1]) }
	$.CGEventPost(0, $.CGEventCreateMouseEvent(null, 5, p, 0))
	delay(0.3)
	$.CGEventPost(0, $.CGEventCreateMouseEvent(null, 1, p, 0))
	$.CGEventPost(0, $.CGEventCreateMouseEvent(null, 2, p, 0))
}`

var chromeLogLine = regexp.MustCompile(`msg="window chrome" titlebar=([0-9.]+) controlsEnd=([0-9.]+)`)

// checkWindowChrome holds the header to where macOS put the traffic lights
// (ADR-10, 2026-10-19 amendment): the bar the launcher measured and logged
// must be the one System Events reports, and the header's first button, which
// sits in that bar, must still take a click. With KAPITAL_IT_SHOTS set it also
// keeps a picture of the window's top before and after the click.
func checkWindowChrome(t *testing.T, pid, stderr string) {
	t.Helper()
	m := chromeLogLine.FindStringSubmatch(stderr)
	if m == nil {
		t.Errorf("no window chrome line in the launcher's log:\n%s", stderr)
		return
	}
	titlebar, _ := strconv.ParseFloat(m[1], 64)    //nolint:errcheck // the pattern admits only digits and dots
	controlsEnd, _ := strconv.ParseFloat(m[2], 64) //nolint:errcheck // the pattern admits only digits and dots
	t.Logf("measured: titlebar %v, controls end %v", titlebar, controlsEnd)

	out, err := exec.Command("/usr/bin/osascript", "-e", strings.Replace(windowChromeScript, "%s", pid, 1)).CombinedOutput()
	if err != nil {
		t.Logf("window chrome unread: %v: %s", err, strings.TrimSpace(string(out)))
		return
	}
	boxes := map[string][4]float64{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		if len(f) != 5 {
			continue
		}
		var b [4]float64
		for i := range b {
			b[i], _ = strconv.ParseFloat(f[i+1], 64) //nolint:errcheck // a field that is not a number reads as 0 and fails the checks below
		}
		boxes[f[0]] = b
	}
	t.Logf("window chrome: %v", boxes)
	win, closeBox, zoom := boxes["window"], boxes["AXCloseButton"], boxes["AXFullScreenButton"]
	if centre := 2 * (closeBox[1] + closeBox[3]/2 - win[1]); math.Abs(centre-titlebar) > 2 {
		t.Errorf("the lights centre a %v pt bar, the launcher measured %v", centre, titlebar)
	}
	if end := zoom[0] + zoom[2] - win[0]; math.Abs(end-controlsEnd) > 3 {
		t.Errorf("the lights end %v pt in, the launcher measured %v", end, controlsEnd)
	}

	shot := func(name string) {
		dir := os.Getenv("KAPITAL_IT_SHOTS")
		if dir == "" {
			return
		}
		rect := fmt.Sprintf("%v,%v,400,120", win[0], win[1])
		if out, err := exec.Command("/usr/sbin/screencapture", "-x", "-R", rect, filepath.Join(dir, name+"-"+runtime.GOARCH+".png")).CombinedOutput(); err != nil {
			t.Logf("screencapture: %v: %s", err, out)
		}
	}
	shot("header")

	// The sidebar's button: 12 px past the lights (pl-3) and half of its 28 px
	// box, on the bar's middle. A click that lands saves the compact sidebar.
	x, y := win[0]+controlsEnd+12+14, win[1]+titlebar/2
	if out, err := exec.Command("/usr/bin/osascript", "-l", "JavaScript", "-e", clickScript, fmt.Sprint(x), fmt.Sprint(y)).CombinedOutput(); err != nil {
		t.Logf("click unsent: %v: %s", err, out)
		return
	}
	settings := filepath.Join(DataDir(), SettingsFileName)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b, err := os.ReadFile(settings); err == nil && strings.Contains(string(b), `"sidebarCompact": true`) {
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("a click on the sidebar's button at (%v, %v) did not compact the sidebar", x, y)
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	shot("header-clicked")
}
