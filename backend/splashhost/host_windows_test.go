//go:build windows

package splashhost

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/wailsapp/go-webview2/webviewloader"
	"golang.org/x/sys/windows"

	"kapital/backend/testwindows"
)

const (
	// ciColdOpenTimeout is how long a test on CI lets a card open in a data
	// folder WebView2 has never used. The hosted Windows runner has no GPU,
	// and since its 20261007 image (Edge 154) the first start in a new folder,
	// which builds the shader caches (GrShaderCache, DawnGraphiteCache) in
	// software, has run past openTimeout. A second start in the same
	// folder is warm and still has to open within openTimeout, as do all
	// starts off CI.
	ciColdOpenTimeout = 30 * time.Second
	// browserExitTimeout bounds the wait for WebView2's processes to end once
	// every card is closed. They end within a second on a desktop; a leak
	// never ends, and fails the test when the bound runs out.
	browserExitTimeout = 30 * time.Second
	webviewExe         = "msedgewebview2.exe"
)

// coldOpenTimeout is the bound for a card's first start in a new data folder.
func coldOpenTimeout() time.Duration {
	if os.Getenv("CI") != "" {
		return ciColdOpenTimeout
	}
	return openTimeout
}

// realWindowDir gates a test that opens a real card and gives it a data
// folder. Before the folder is removed it waits for the WebView2 processes the
// test started to end, so a card that failed mid-start leaves a failure that
// says so, not a locked cache file in TempDir's cleanup.
func realWindowDir(t *testing.T) string {
	t.Helper()
	testwindows.Require(t, "opens a real WebView2 window")
	if v, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString(""); err == nil {
		t.Logf("WebView2 runtime %s", v)
	}
	dir := t.TempDir()
	// Registered after TempDir's own cleanup, so it runs before it.
	t.Cleanup(func() {
		if err := waitForBrowsersToExit(browserExitTimeout); err != nil {
			t.Error(err)
		}
	})
	return dir
}

// waitForBrowsersToExit waits until no WebView2 process this test binary
// started is running: the browser process, a child of this one, and its own
// children, which are found through the browser even after it has ended.
func waitForBrowsersToExit(within time.Duration) error {
	deadline := time.Now().Add(within)
	tracked := map[uint32]windows.Handle{}
	defer func() {
		for _, h := range tracked {
			windows.CloseHandle(h) //nolint:errcheck // a handle only waited on
		}
	}()
	for {
		if err := trackWebviewProcesses(tracked); err != nil {
			return err
		}
		var running []uint32
		for pid, h := range tracked {
			ev, err := windows.WaitForSingleObject(h, 0)
			if err != nil {
				return fmt.Errorf("waiting for WebView2 process %d: %w", pid, err)
			}
			if ev == uint32(windows.WAIT_TIMEOUT) {
				running = append(running, pid)
			}
		}
		if len(running) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("WebView2 processes %v still run %s after the card closed", running, within)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// trackWebviewProcesses adds to tracked, with a handle that keeps its id from
// being reused, every WebView2 process whose parent is this process or one
// already tracked.
func trackWebviewProcesses(tracked map[uint32]windows.Handle) error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return fmt.Errorf("listing processes: %w", err)
	}
	defer windows.CloseHandle(snap) //nolint:errcheck // a snapshot only read
	type proc struct{ pid, parent uint32 }
	var procs []proc
	e := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), webviewExe) {
			procs = append(procs, proc{e.ProcessID, e.ParentProcessID})
		}
	}
	self := uint32(os.Getpid())
	for added := true; added; {
		added = false
		for _, p := range procs {
			if _, ok := tracked[p.pid]; ok {
				continue
			}
			if _, ours := tracked[p.parent]; !ours && p.parent != self {
				continue
			}
			h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, p.pid)
			if err != nil {
				continue // it has ended already
			}
			tracked[p.pid] = h
			added = true
		}
	}
	return nil
}

// removeOnceFree removes the folder, retrying briefly for a file the system
// itself (the search indexer, an antivirus scan) may still have open.
func removeOnceFree(t *testing.T, dir string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := os.RemoveAll(dir)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the data folder is still held after WebView2 ended: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// findCardWindow is the card's top-level window, or 0 when none exists.
func findCardWindow() uintptr {
	class, err := windows.UTF16PtrFromString(cardClass)
	if err != nil {
		return 0
	}
	find := user32.NewProc("FindWindowW")
	return call(find, uintptr(unsafe.Pointer(class)), 0)
}

func windowVisible(hwnd uintptr) bool {
	return call(user32.NewProc("IsWindowVisible"), hwnd) != 0
}

// testPage is a page that says what it sees: it posts {"action":"leave"} when
// it loads, echoes every state it is given, and reports what a fetch of a file
// that is there and of one that is not come back as.
const testPage = `<!doctype html><meta charset="utf-8"><title>t</title><script>
const post = (o) => window.chrome.webview.postMessage(JSON.stringify(o));
window.kapitalSplash = { update: (s) => post({ action: "copyLog", state: s }) };
post({ action: "leave" });
fetch("data.json").then((r) => r.json()).then((j) => post({ action: "openFolder", got: j }));
fetch("missing.json").then((r) => post({ action: "openFolder", missing: r.status }));
</script>`

func testAssets(name string) ([]byte, string, bool) {
	switch name {
	case "splash.html":
		return []byte(testPage), "text/html; charset=utf-8", true
	case "data.json":
		return []byte(`{"n":42}`), "application/json", true
	}
	return nil, "", false
}

// collector gathers the page's messages.
type collector struct {
	mu   sync.Mutex
	msgs []string
}

func (c *collector) add(m string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, m)
}

func (c *collector) has(sub string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range c.msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestARealWindowLoadsItsPageTalksBackAndLeavesNothing runs three cycles in the
// same data folder, which is what a player's three starts are: each opens the
// card, gets a message from the page, pushes a state to it and closes, and
// each close must leave no window and no browser holding the folder. Only the
// first, cold, start gets longer on CI; the warm ones keep openTimeout.
func TestARealWindowLoadsItsPageTalksBackAndLeavesNothing(t *testing.T) {
	dataDir := realWindowDir(t)
	for cycle := 1; cycle <= 3; cycle++ {
		got := &collector{}
		h := New()
		if cycle == 1 {
			h.(*windowsHost).openWithin = coldOpenTimeout()
		}
		start := time.Now()
		err := h.Open(Rect{X: 200, Y: 200, W: 320, H: 200}, Page{
			Assets:     testAssets,
			Entry:      "splash.html",
			OnMessage:  got.add,
			DataDir:    dataDir,
			Background: [3]uint8{10, 20, 30},
		})
		if errors.Is(err, errNoRuntime) {
			t.Skipf("no WebView2 runtime: %v", err)
		}
		if err != nil {
			h.Close()
			t.Fatalf("cycle %d: open: %v", cycle, err)
		}
		t.Logf("cycle %d: open took %s", cycle, time.Since(start).Round(time.Millisecond))

		hwnd := findCardWindow()
		if hwnd == 0 || !windowVisible(hwnd) {
			t.Fatalf("cycle %d: no visible card window after Open (hwnd %#x)", cycle, hwnd)
		}
		waitFor(t, "the page's first message", func() bool { return got.has(`"action":"leave"`) })
		waitFor(t, "a file of the build, fetched by the page", func() bool { return got.has(`"n":42`) })
		waitFor(t, "a 404 for a file that is not in the build", func() bool { return got.has(`"missing":404`) })

		h.Update([]byte(`{"n":1}`))
		h.Update([]byte(`{"n":2}`))
		waitFor(t, "the newest state, echoed back by the page", func() bool { return got.has(`"n":2`) && got.has(`"state"`) })

		closed := make(chan struct{})
		go func() { h.Close(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Fatalf("cycle %d: Close did not return", cycle)
		}
		if findCardWindow() != 0 {
			t.Fatalf("cycle %d: the window is still there after Close", cycle)
		}
		h.Close() // twice is safe
		h.Update([]byte(`{"n":3}`))
	}

	// The browser process that held the folder ends shortly after the last
	// Close; if the card let go of nothing it would hold the folder until the
	// launcher exits.
	if err := waitForBrowsersToExit(browserExitTimeout); err != nil {
		t.Fatal(err)
	}
	removeOnceFree(t, dataDir)
}

func TestAStateThatArrivesBeforeTheWindowHasLoadedIsNotLost(t *testing.T) {
	dataDir := realWindowDir(t)
	got := &collector{}
	h := New()
	defer h.Close()
	h.(*windowsHost).openWithin = coldOpenTimeout()
	opened := make(chan error, 1)
	go func() {
		opened <- h.Open(Rect{X: 200, Y: 200, W: 320, H: 200}, Page{
			Assets:    testAssets,
			Entry:     "splash.html",
			OnMessage: got.add,
			DataDir:   dataDir,
		})
	}()
	// The tracker pushes as soon as its goroutine has the host, which can be
	// while Open is still waiting for the page: the state waits for it.
	wh := h.(*windowsHost)
	waitFor(t, "Open to start", func() bool {
		wh.mu.Lock()
		defer wh.mu.Unlock()
		return wh.card != nil
	})
	h.Update([]byte(`{"n":7}`))
	if err := <-opened; err != nil {
		if errors.Is(err, errNoRuntime) {
			t.Skipf("no WebView2 runtime: %v", err)
		}
		t.Fatal(err)
	}
	waitFor(t, "the state, echoed back by the page", func() bool { return got.has(`"n":7`) })
}

// TestACloseWhileTheWebviewStartsLeavesTheProcessRunning closes the card while
// WebView2 is still creating it, which is what Open's own timeout does on a
// slow start and what a game that is up before the card does. Destroying the
// window then aborts the creation, and go-webview2 ends the whole process on
// that, so this test failing is the test binary exiting with no FAIL line.
func TestACloseWhileTheWebviewStartsLeavesTheProcessRunning(t *testing.T) {
	dataDir := realWindowDir(t)
	h := New()
	opened := make(chan error, 1)
	go func() {
		opened <- h.Open(Rect{X: 200, Y: 200, W: 320, H: 200}, Page{
			Assets:  testAssets,
			Entry:   "splash.html",
			DataDir: dataDir,
		})
	}()
	wh := h.(*windowsHost)
	waitFor(t, "the webview to start", func() bool {
		wh.mu.Lock()
		c := wh.card
		wh.mu.Unlock()
		return c != nil && c.isEmbedding()
	})
	h.Close()

	err := <-opened
	if errors.Is(err, errNoRuntime) {
		t.Skipf("no WebView2 runtime: %v", err)
	}
	if err == nil {
		t.Fatal("Open succeeded on a card closed while it started")
	}
	select {
	case <-wh.card.done:
	case <-time.After(30 * time.Second):
		t.Fatal("the card's thread did not finish after the webview started")
	}
	if findCardWindow() != 0 {
		t.Fatal("the window is still there after the card's thread finished")
	}
	if err := waitForBrowsersToExit(browserExitTimeout); err != nil {
		t.Fatal(err)
	}
	removeOnceFree(t, dataDir)
}
