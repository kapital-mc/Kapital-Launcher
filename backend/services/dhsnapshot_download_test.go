package services

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// dhFixture is a published snapshot on a TLS test server, laid out as Konnekt
// lays it out, and an instance to download it into. Parts are 4 bytes so a
// level has several.
type dhFixture struct {
	srv      *httptest.Server
	source   *DHSource
	instance string
	pointer  string
	index    DHIndex
	parts    map[string][]byte
	levels   map[string][]byte

	mu sync.Mutex
	// fetched counts the GETs of each part; cut and corrupt make a part's
	// next answers short or wrong, as many times as their counts say.
	fetched map[string]int
	cut     map[string]int
	corrupt map[string]int
	// served, when set, hears each part answered.
	served func()
}

const dhPartLen = 4

func newDHFixture(t *testing.T, levels map[string]string) *dhFixture {
	t.Helper()
	f := &dhFixture{
		parts: map[string][]byte{}, levels: map[string][]byte{},
		fetched: map[string]int{}, cut: map[string]int{}, corrupt: map[string]int{},
		index: DHIndex{
			Format: 1, CreatedAt: "2026-10-13T12:03:41Z", PartsBase: "../../parts/",
			DHVersion: "3.3.3", ServerFolder: "1790374988_frangfurd",
		},
	}
	for _, folder := range slices.Sorted(maps.Keys(levels)) {
		data := []byte(levels[folder])
		f.levels[folder] = data
		l := DHLevel{Folder: folder, Size: int64(len(data)), SHA256: sha(data)}
		for off := 0; off < len(data); off += dhPartLen {
			chunk := data[off:min(off+dhPartLen, len(data))]
			f.parts[sha(chunk)] = chunk
			l.Parts = append(l.Parts, DHPart{SHA256: sha(chunk), Size: int64(len(chunk))})
		}
		f.index.Levels = append(f.index.Levels, l)
	}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	client := f.srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	f.source = &DHSource{client: client, stall: 2 * time.Second, backoff: time.Millisecond}
	f.pointer = f.srv.URL + "/dh/latest.json"
	f.instance = t.TempDir()
	if err := os.Mkdir(filepath.Join(f.instance, "minecraft"), 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *dhFixture) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/dh/latest.json":
		writeJSON(w, DHPointer{Format: 1, CreatedAt: f.index.CreatedAt, Index: "snapshots/20261013T120341Z/index.json"})
	case r.URL.Path == "/dh/snapshots/20261013T120341Z/index.json":
		writeJSON(w, f.index)
	case strings.HasPrefix(r.URL.Path, "/dh/parts/"):
		f.servePart(w, strings.TrimPrefix(r.URL.Path, "/dh/parts/"))
	default:
		http.NotFound(w, r)
	}
}

func (f *dhFixture) servePart(w http.ResponseWriter, name string) {
	f.mu.Lock()
	data, ok := f.parts[name]
	f.fetched[name]++
	cut, corrupt := f.cut[name] > 0, f.corrupt[name] > 0
	if cut {
		f.cut[name]--
	}
	if corrupt {
		f.corrupt[name]--
	}
	served := f.served
	f.mu.Unlock()
	if served != nil {
		defer served()
	}
	switch {
	case !ok:
		w.WriteHeader(http.StatusNotFound)
	case cut:
		// A connection that drops mid-part: the length promised, half sent.
		w.Header().Set("Content-Length", "4")
		_, _ = w.Write(data[:len(data)/2]) //nolint:errcheck // the client sees the cut
	case corrupt:
		_, _ = w.Write([]byte(strings.Repeat("x", len(data)))) //nolint:errcheck // the client checks the hash
	default:
		_, _ = w.Write(data) //nolint:errcheck // the client checks the hash
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data) //nolint:errcheck // test server
}

func (f *dhFixture) fetches() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.fetched {
		n += c
	}
	return n
}

func (f *dhFixture) snapshot(t *testing.T) DHSnapshot {
	t.Helper()
	snap, err := f.source.Fetch(t.Context(), f.pointer)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func (f *dhFixture) placed(folder string) string {
	return filepath.Join(f.instance, "minecraft", dhServerData, f.index.ServerFolder, folder, dhDatabaseName)
}

func (f *dhFixture) download(t *testing.T, ctx context.Context, phases *[]string) error {
	t.Helper()
	return f.source.Download(ctx, DHDownloadRequest{
		InstanceDir: f.instance,
		Snapshot:    f.snapshot(t),
		Progress: func(phase string, received, total int64) {
			if phases != nil && (len(*phases) == 0 || (*phases)[len(*phases)-1] != phase) {
				*phases = append(*phases, phase)
			}
		},
	})
}

var twoLevels = map[string]string{
	"ff@minecraft@@overworld":   "overworld terrain, many chunks",
	"ff@northstar@@earth_orbit": "orbit",
}

func TestDHDownloadPlacesEveryLevelAndCleansUp(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	// The player's own database, with a write-ahead log that must not be
	// replayed into the new one.
	own := f.placed("ff@minecraft@@overworld")
	if err := os.MkdirAll(filepath.Dir(own), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{own, own + "-wal", own + "-shm"} {
		if err := os.WriteFile(name, []byte("player's own"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := DHExistingBytes(f.instance, f.index); got != 3*int64(len("player's own")) {
		t.Errorf("existing bytes %d", got)
	}
	var phases []string
	if err := f.download(t, t.Context(), &phases); err != nil {
		t.Fatal(err)
	}
	for folder, data := range f.levels {
		got, err := os.ReadFile(f.placed(folder))
		if err != nil || string(got) != string(data) {
			t.Errorf("%s: %q %v", folder, got, err)
		}
	}
	for _, gone := range []string{own + "-wal", own + "-shm", filepath.Join(f.instance, dhStaging)} {
		if _, err := os.Stat(gone); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s is still there: %v", gone, err)
		}
	}
	if strings.Join(phases, ",") != "downloading,verifying,placing,done" {
		t.Errorf("phases %v", phases)
	}
}

func TestDHDownloadRetriesACutConnectionAndABadPart(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	first := f.index.Levels[0].Parts[0].SHA256
	second := f.index.Levels[0].Parts[1].SHA256
	f.cut[first] = 1
	f.corrupt[second] = 2
	if err := f.download(t, t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if f.fetched[first] != 2 || f.fetched[second] != 3 {
		t.Errorf("fetched %d and %d times", f.fetched[first], f.fetched[second])
	}
}

func TestDHDownloadFailsAPartThatNeverMatchesAndResumes(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	bad := f.index.Levels[0].Parts[2].SHA256
	f.corrupt[bad] = dhAttempts
	err := f.download(t, t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "does not match its hash") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(f.placed("ff@minecraft@@overworld")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a level was placed before the whole snapshot was verified")
	}
	snap := f.snapshot(t)
	// The overworld's first two parts are kept; the orbit, after it, was not reached.
	if got, want := DHPartialBytes(f.instance, snap), int64(2*dhPartLen); got != want {
		t.Errorf("partial %d, want %d", got, want)
	}
	before := f.fetches()
	if err := f.download(t, t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if got, want := f.fetches()-before, len(f.parts)-2; got != want {
		t.Errorf("the resume fetched %d parts, want %d", got, want)
	}
}

func TestDHDownloadCancelledKeepsWhatWasVerified(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	snap := f.snapshot(t)
	f.served = func() {
		if f.fetches() == 3 {
			cancel()
		}
	}
	err := f.source.Download(ctx, DHDownloadRequest{InstanceDir: f.instance, Snapshot: snap})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if got := DHPartialBytes(f.instance, snap); got == 0 || got >= snap.Index.Total() {
		t.Errorf("partial %d of %d", got, snap.Index.Total())
	}
}

func TestDHDownloadPlacesNothingWhileTheGameRuns(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	snap := f.snapshot(t)
	running := errors.New("Frangfurd is starting or running")
	err := f.source.Download(t.Context(), DHDownloadRequest{
		InstanceDir: f.instance, Snapshot: snap, BeforePlace: func() error { return running },
	})
	if !errors.Is(err, running) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(f.placed("ff@minecraft@@overworld")); !errors.Is(err, os.ErrNotExist) {
		t.Error("placed while running")
	}
	// Once the game is closed, nothing is fetched again.
	before := f.fetches()
	if err := f.download(t, t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if f.fetches() != before {
		t.Errorf("fetched %d parts again", f.fetches()-before)
	}
}

func TestDHDownloadStartsOverForAnotherSnapshot(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	snap := f.snapshot(t)
	stale := filepath.Join(f.instance, dhStaging, "ff@minecraft@@old.sqlite")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	j := dhJournal{Format: 1, Index: strings.Repeat("0", 64), Done: map[string]int{"ff@minecraft@@overworld": 3}}
	root, err := os.OpenRoot(f.instance)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test
	if err := j.save(root); err != nil {
		t.Fatal(err)
	}
	if got := DHPartialBytes(f.instance, snap); got != 0 {
		t.Errorf("another snapshot's journal counted %d bytes", got)
	}
	if err := f.download(t, t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if f.fetches() != len(f.parts) {
		t.Errorf("fetched %d parts, the snapshot has %d", f.fetches(), len(f.parts))
	}
}

func TestDHDownloadRefusesAWholeFileThatDoesNotMatch(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	f.index.Levels[0].SHA256 = strings.Repeat("0", 64)
	err := f.download(t, t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "did not match the snapshot") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.instance, dhStaging, "ff@minecraft@@overworld.sqlite")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the bad file was kept")
	}
}

func TestDHDownloadNeedsAGameFolder(t *testing.T) {
	f := newDHFixture(t, twoLevels)
	if err := os.Remove(filepath.Join(f.instance, "minecraft")); err != nil {
		t.Fatal(err)
	}
	if err := f.download(t, t.Context(), nil); err == nil {
		t.Error("downloaded into an instance with no game folder")
	}
}

func TestDHFetchFollowsNoRedirectAndRefusesABigPointer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/moved/latest.json":
			http.Redirect(w, r, "https://evil.example/latest.json", http.StatusFound)
		case "/big/latest.json":
			_, _ = w.Write([]byte(strings.Repeat(" ", dhPointerMax+1))) //nolint:errcheck // test server
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := srv.Client()
	client.CheckRedirect = NewDHSource().client.CheckRedirect
	s := &DHSource{client: client, stall: time.Second}
	for _, p := range []string{"/moved/latest.json", "/big/latest.json", "/gone/latest.json"} {
		if _, err := s.Fetch(t.Context(), srv.URL+p); err == nil {
			t.Errorf("%s: fetched", p)
		}
	}
	if _, err := s.Fetch(t.Context(), strings.Replace(srv.URL, "https", "http", 1)+"/dh/latest.json"); err == nil {
		t.Error("fetched over http")
	}
}
