//go:build darwin && macintegration

package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"kapital/backend/models"
)

// TestMacSelfUpdateSwapsTheBundle runs the launcher's own update on a real Mac
// with the bundle the workflow built (KAPITAL_IT_APP): the release zip made as
// build.yml makes it, served with signed checksums, unpacked by ditto, checked
// by codesign and swapped in by renames. On a runner (KAPITAL_IT_DISPOSABLE)
// the installed copy is in /Applications, where macOS's App Management would
// stop a swap it objects to.
func TestMacSelfUpdateSwapsTheBundle(t *testing.T) {
	app := itEnv(t, "KAPITAL_IT_APP")
	work := t.TempDir()

	// The release: the bundle zipped with ditto, as build.yml does.
	const tag = "v99.0.0"
	asset := "Kapital-Launcher-" + tag + "-macos-universal.zip"
	zip := filepath.Join(work, asset)
	runTool(t, "/usr/bin/ditto", "-c", "-k", "--keepParent", app, zip)
	build, err := os.ReadFile(zip)
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(build)
	sums := []byte(hex.EncodeToString(sum[:]) + "  " + asset + "\n")
	files := map[string][]byte{
		asset:         build,
		checksumsName: sums,
		signatureName: []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(private, sums))),
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[strings.TrimPrefix(r.URL.Path, "/dl/"+tag+"/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body) //nolint:errcheck // a test server
	}))
	t.Cleanup(srv.Close)

	// The installed copy that runs.
	installDir := filepath.Join(work, "Applications")
	if os.Getenv("KAPITAL_IT_DISPOSABLE") == "1" {
		installDir = "/Applications"
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(installDir, launcherBundleName)
	if err := os.RemoveAll(installed); err != nil {
		t.Fatal(err)
	}
	runTool(t, "/usr/bin/ditto", app, installed)
	t.Cleanup(func() { _ = os.RemoveAll(installed) }) //nolint:errcheck // a test copy
	marker := filepath.Join(installed, "Contents", "old-copy")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	u := newLauncherUpdater("darwin", srv.URL+"/dl/", []ed25519.PublicKey{public}, systemUpdatePlatform())
	u.client = srv.Client()
	u.client.CheckRedirect = u.checkRedirect
	u.allowedHosts = []string{"127.0.0.1"}
	u.executable = func() (string, error) {
		return filepath.Join(installed, "Contents", "MacOS", "kapital-launcher"), nil
	}

	if target, err := u.Target(); err != nil || target != installed {
		t.Fatalf("Target() = %q, %v", target, err)
	}
	var last models.LauncherUpdateProgress
	if err := u.Download(context.Background(), tag, func(p models.LauncherUpdateProgress) { last = p }); err != nil {
		t.Fatal(err)
	}
	if last.Phase != "ready" {
		t.Fatalf("last step = %+v", last)
	}
	target, err := u.Apply()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the old bundle is still in place")
	}
	runTool(t, "/usr/bin/codesign", "--verify", "--deep", "--strict", target)
	// Downloaded by the launcher, so not quarantined: no Open Anyway after an update.
	if out, err := exec.Command("/usr/bin/xattr", "-r", "-l", target).CombinedOutput(); err != nil || strings.Contains(string(out), "com.apple.quarantine") {
		t.Fatalf("the updated bundle carries quarantine: %s, %v", out, err)
	}
	u.CleanUp(context.Background())
	if left, _ := filepath.Glob(filepath.Join(installDir, stagingPrefix+"*")); len(left) != 0 {
		t.Fatalf("leftovers after CleanUp: %v", left)
	}
}

// A copy macOS translocated cannot update itself, and says what to do.
func TestMacSelfUpdateRefusesATranslocatedCopy(t *testing.T) {
	_, err := bundleTarget("/private/var/folders/x/AppTranslocation/ABC/d/Kapital Launcher.app/Contents/MacOS/kapital-launcher")
	if err == nil || !strings.Contains(err.Error(), "Applications") {
		t.Fatalf("err = %v", err)
	}
}

func runTool(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v: %s", name, args, err, out)
	}
}
