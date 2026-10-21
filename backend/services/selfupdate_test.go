package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kapital/backend/models"
)

const testTag = "v1.2.0"

// fakeRelease serves a release's files from memory, as GitHub's download URLs would.
type fakeRelease struct {
	files    map[string][]byte
	requests []string
}

func (f *fakeRelease) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests = append(f.requests, r.URL.Path)
	body, ok := f.files[strings.TrimPrefix(r.URL.Path, "/dl/"+testTag+"/")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write(body) //nolint:errcheck // a test server
}

// updaterFixture is an updater over a fake release, a fake platform that
// swaps a plain file, and a running "launcher" in a temp folder.
type updaterFixture struct {
	u        *LauncherUpdater
	release  *fakeRelease
	target   string
	key      ed25519.PrivateKey
	relaunch []string
}

func newUpdaterFixture(t *testing.T) *updaterFixture {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "launcher.exe")
	if err := os.WriteFile(target, []byte("old build"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := &updaterFixture{release: &fakeRelease{files: map[string][]byte{}}, target: target, key: private}
	srv := httptest.NewTLSServer(f.release)
	t.Cleanup(srv.Close)
	p := updatePlatform{
		asset:    func(tag string) string { return "launcher-" + tag + ".bin" },
		target:   func(exe string) (string, error) { return exe, nil },
		prepare:  func(_ context.Context, _, download string) (string, error) { return download, nil },
		relaunch: func(path string) error { f.relaunch = append(f.relaunch, path); return nil },
	}
	f.u = newLauncherUpdater("test", srv.URL+"/dl/", []ed25519.PublicKey{public}, p)
	f.u.client = srv.Client()
	f.u.client.CheckRedirect = f.u.checkRedirect
	f.u.allowedHosts = []string{"127.0.0.1"}
	f.u.executable = func() (string, error) { return target, nil }
	f.publish([]byte("new build"))
	return f
}

// publish puts a build on the fake release with signed checksums.
func (f *updaterFixture) publish(build []byte) {
	sum := sha256.Sum256(build)
	sums := []byte(hex.EncodeToString(sum[:]) + "  launcher-" + testTag + ".bin\n" +
		strings.Repeat("0", 64) + "  checksums-of-something-else.zip\n")
	f.release.files["launcher-"+testTag+".bin"] = build
	f.release.files[checksumsName] = sums
	f.release.files[signatureName] = []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(f.key, sums)) + "\n")
}

func (f *updaterFixture) download(t *testing.T) ([]models.LauncherUpdateProgress, error) {
	t.Helper()
	var steps []models.LauncherUpdateProgress
	err := f.u.Download(context.Background(), testTag, func(p models.LauncherUpdateProgress) { steps = append(steps, p) })
	return steps, err
}

func TestUpdateDownloadsStagesAndSwapsIn(t *testing.T) {
	f := newUpdaterFixture(t)
	steps, err := f.download(t)
	if err != nil {
		t.Fatal(err)
	}
	if last := steps[len(steps)-1]; last.Phase != "ready" || last.Version != testTag {
		t.Fatalf("last step = %+v, want ready", last)
	}
	if got := readFile(t, f.target); got != "old build" {
		t.Fatalf("the download replaced the launcher before Restart: %q", got)
	}
	if f.u.Ready() != testTag {
		t.Fatalf("Ready() = %q", f.u.Ready())
	}
	target, err := f.u.Apply()
	if err != nil {
		t.Fatal(err)
	}
	if target != f.target || readFile(t, f.target) != "new build" {
		t.Fatalf("after Apply the launcher holds %q", readFile(t, f.target))
	}
	if f.u.Ready() != "" {
		t.Fatal("an applied update is still staged")
	}
	// The next start removes the staging folder with the old copy in it.
	f.u.CleanUp(context.Background())
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(f.target), stagingPrefix+"*")); len(left) != 0 {
		t.Fatalf("leftovers after CleanUp: %v", left)
	}
}

func TestUpdateRefusesWhatTheKeyDidNotSign(t *testing.T) {
	cases := map[string]func(f *updaterFixture){
		"build swapped after signing": func(f *updaterFixture) {
			f.release.files["launcher-"+testTag+".bin"] = []byte("evil build")
		},
		"checksums edited": func(f *updaterFixture) {
			f.release.files[checksumsName] = append(f.release.files[checksumsName], []byte("x")...)
		},
		"another key": func(f *updaterFixture) {
			_, other, _ := ed25519.GenerateKey(rand.Reader) //nolint:errcheck // a test key
			f.key = other
			f.publish([]byte("new build"))
		},
		"no signature": func(f *updaterFixture) { delete(f.release.files, signatureName) },
		"no asset line": func(f *updaterFixture) {
			sums := []byte(strings.Repeat("a", 64) + "  other.bin\n")
			f.release.files[checksumsName] = sums
			f.release.files[signatureName] = []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(f.key, sums)))
		},
	}
	for name, spoil := range cases {
		t.Run(name, func(t *testing.T) {
			f := newUpdaterFixture(t)
			spoil(f)
			steps, err := f.download(t)
			if err == nil {
				t.Fatal("the download was staged")
			}
			if last := steps[len(steps)-1]; last.Phase != "failed" {
				t.Fatalf("last step = %+v, want failed", last)
			}
			if f.u.Ready() != "" || readFile(t, f.target) != "old build" {
				t.Fatal("a refused update was staged or placed")
			}
			if left, _ := filepath.Glob(filepath.Join(filepath.Dir(f.target), stagingPrefix+"*")); len(left) != 0 {
				t.Fatalf("a refused download left %v", left)
			}
		})
	}
}

func TestUpdateWithoutAKeyOffersTheSiteOnly(t *testing.T) {
	f := newUpdaterFixture(t)
	f.u.keys = nil
	if _, err := f.u.Target(); !errors.Is(err, errNoUpdateKey) {
		t.Fatalf("Target() error = %v, want errNoUpdateKey", err)
	}
	if _, err := f.download(t); err == nil {
		t.Fatal("a build without a key downloaded an update")
	}
	if len(f.release.requests) != 0 {
		t.Fatalf("asked the release for %v", f.release.requests)
	}
}

func TestUpdateRefusesATagThatIsNotARelease(t *testing.T) {
	f := newUpdaterFixture(t)
	for _, tag := range []string{"1.2.0", "v1.2.0/../../x", "latest", "v1.2.0-dev"} {
		if err := f.u.Download(context.Background(), tag, func(models.LauncherUpdateProgress) {}); err == nil {
			t.Errorf("Download(%q) went ahead", tag)
		}
	}
	if len(f.release.requests) != 0 {
		t.Fatalf("asked the release for %v", f.release.requests)
	}
}

func TestUpdateRefusesAnotherHost(t *testing.T) {
	f := newUpdaterFixture(t)
	f.u.allowedHosts = []string{"github.com"}
	if _, err := f.download(t); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("err = %v, want a refusal", err)
	}
}

func TestUpdateApplyRollsBackWhenThePlacementFails(t *testing.T) {
	f := newUpdaterFixture(t)
	if _, err := f.download(t); err != nil {
		t.Fatal(err)
	}
	// The staged file vanishing makes the second rename fail.
	if err := os.Remove(f.u.staged.replacement); err != nil {
		t.Fatal(err)
	}
	if _, err := f.u.Apply(); err == nil {
		t.Fatal("Apply succeeded without a replacement")
	}
	if readFile(t, f.target) != "old build" {
		t.Fatal("the running launcher was not put back")
	}
}

func TestUpdateApplyWithNothingStaged(t *testing.T) {
	f := newUpdaterFixture(t)
	if _, err := f.u.Apply(); err == nil {
		t.Fatal("Apply succeeded with nothing staged")
	}
}

func TestUpdateTargetNeedsAWritableFolder(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	f := newUpdaterFixture(t)
	dir := filepath.Dir(f.target)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:errcheck // restores the temp dir for its removal
	if _, err := f.u.Target(); err == nil || !strings.Contains(err.Error(), "cannot write") {
		t.Fatalf("Target() error = %v, want the folder named", err)
	}
}

func TestChecksumFor(t *testing.T) {
	d := strings.Repeat("ab", 32)
	sums := []byte(d + "  a.dmg\n" + strings.ToUpper(d) + " *b.exe\r\n" + "zz  c.zip\n")
	if got, err := checksumFor(sums, "a.dmg"); err != nil || got != d {
		t.Errorf("a.dmg = %q, %v", got, err)
	}
	if got, err := checksumFor(sums, "b.exe"); err != nil || got != d {
		t.Errorf("b.exe = %q, %v", got, err)
	}
	if _, err := checksumFor(sums, "c.zip"); err == nil {
		t.Error("a malformed digest was accepted")
	}
	if _, err := checksumFor(sums, "d.zip"); err == nil {
		t.Error("a missing file was found")
	}
}

func TestUpdateKeysParse(t *testing.T) {
	saved := UpdatePublicKeys
	t.Cleanup(func() { UpdatePublicKeys = saved })
	public, _, _ := ed25519.GenerateKey(rand.Reader) //nolint:errcheck // a test key
	UpdatePublicKeys = []string{"not base64!", base64.StdEncoding.EncodeToString(public)}
	if keys := updateKeys(); len(keys) != 1 || !keys[0].Equal(public) {
		t.Fatalf("updateKeys() = %v", keys)
	}
}
