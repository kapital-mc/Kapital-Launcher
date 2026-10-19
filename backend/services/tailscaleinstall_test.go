package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kapital/backend/models"
)

type fakePackages struct {
	srv       *httptest.Server
	installer []byte
	digest    string
	index     string
}

func newFakePackages(t *testing.T) *fakePackages {
	t.Helper()
	p := &fakePackages{installer: []byte(strings.Repeat("tailscale", 100000))}
	sum := sha256.Sum256(p.installer)
	p.digest = hex.EncodeToString(sum[:])
	p.index = `{"MSIs":{"amd64":"tailscale-setup-1.104.1-amd64.msi"},"MSIsVersion":"1.104.1",` +
		`"MacZips":{"universal-package":"Tailscale-1.104.1-macos.pkg"},"MacZipsVersion":"1.104.1"}`
	p.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/stable/" && r.URL.Query().Get("mode") == "json":
			fmt.Fprint(w, p.index)
		case strings.HasSuffix(r.URL.Path, ".sha256"):
			fmt.Fprint(w, p.digest+"\n")
		case strings.HasPrefix(r.URL.Path, "/stable/"):
			w.Header().Set("Content-Length", fmt.Sprint(len(p.installer)))
			if r.Method == http.MethodGet {
				w.Write(p.installer) //nolint:errcheck // test server
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *fakePackages) installerFor(t *testing.T, goos string) (*TailscaleInstaller, *[]string) {
	t.Helper()
	ti := NewTailscaleInstaller(t.TempDir(), goos, "amd64")
	ti.base = p.srv.URL + "/stable/"
	ti.client = p.srv.Client()
	var ran []string
	ti.verify = func(context.Context, string) error { return nil }
	ti.runInstaller = func(_ context.Context, path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ran = append(ran, filepath.Base(path)+fmt.Sprint(len(data)))
		return nil
	}
	return ti, &ran
}

func TestTailscaleLatestPicksThisOSsInstaller(t *testing.T) {
	p := newFakePackages(t)
	win, _ := p.installerFor(t, "windows")
	rel, err := win.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.File != "tailscale-setup-1.104.1-amd64.msi" || rel.Version != "1.104.1" ||
		rel.Size != int64(len(p.installer)) || rel.Digest != "sha256:"+p.digest || rel.URL != win.base+rel.File {
		t.Fatalf("%+v", rel)
	}
	mac, _ := p.installerFor(t, "darwin")
	if rel, err := mac.Latest(context.Background()); err != nil || rel.File != "Tailscale-1.104.1-macos.pkg" {
		t.Fatalf("macOS: %+v %v", rel, err)
	}
	linux, _ := p.installerFor(t, "linux")
	if _, err := linux.Latest(context.Background()); err == nil {
		t.Fatal("an installer for Linux")
	}
}

func TestTailscaleLatestRefusesAnOddIndex(t *testing.T) {
	p := newFakePackages(t)
	ti, _ := p.installerFor(t, "windows")
	for name, index := range map[string]string{
		"a path":           `{"MSIs":{"amd64":"../evil-1.104.1.msi"},"MSIsVersion":"1.104.1"}`,
		"another version":  `{"MSIs":{"amd64":"tailscale-setup-9.9.9-amd64.msi"},"MSIsVersion":"1.104.1"}`,
		"not an installer": `{"MSIs":{"amd64":"tailscale-1.104.1.exe"},"MSIsVersion":"1.104.1"}`,
		"no version":       `{"MSIs":{"amd64":"tailscale-setup-amd64.msi"},"MSIsVersion":""}`,
	} {
		p.index = index
		if _, err := ti.Latest(context.Background()); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	p.index = `{"MSIs":{"amd64":"tailscale-setup-1.104.1-amd64.msi"},"MSIsVersion":"1.104.1"}`
	p.digest = "not a digest"
	if _, err := ti.Latest(context.Background()); err == nil {
		t.Error("accepted a release without a digest")
	}
}

func TestTailscaleInstallVerifiesThenRuns(t *testing.T) {
	p := newFakePackages(t)
	ti, ran := p.installerFor(t, "windows")
	rel, err := ti.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var phases []string
	if err := ti.Install(context.Background(), rel, func(pr models.TailscaleInstallProgress) {
		if len(phases) == 0 || phases[len(phases)-1] != pr.Phase {
			phases = append(phases, pr.Phase)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(phases, ",") != "downloading,verifying,installing,done" {
		t.Errorf("phases %v", phases)
	}
	if len(*ran) != 1 || (*ran)[0] != rel.File+fmt.Sprint(len(p.installer)) {
		t.Errorf("ran %v", *ran)
	}
	if _, err := os.Stat(filepath.Join(ti.dir, rel.File)); !errors.Is(err, os.ErrNotExist) {
		t.Error("the installer was left behind")
	}
}

func TestTailscaleInstallRefusesAMismatch(t *testing.T) {
	p := newFakePackages(t)
	ti, ran := p.installerFor(t, "windows")
	rel, err := ti.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	bad := rel
	bad.Digest = "sha256:" + strings.Repeat("0", 64)
	var last models.TailscaleInstallProgress
	if err := ti.Install(context.Background(), bad, func(p models.TailscaleInstallProgress) { last = p }); err == nil || last.Phase != "failed" {
		t.Fatalf("a wrong digest: %v %+v", err, last)
	}
	elsewhere := rel
	elsewhere.URL = "https://evil.example/" + rel.File
	if err := ti.Install(context.Background(), elsewhere, func(models.TailscaleInstallProgress) {}); err == nil {
		t.Fatal("downloaded from another host")
	}
	ti.verify = func(context.Context, string) error { return errors.New("unsigned") }
	if err := ti.Install(context.Background(), rel, func(models.TailscaleInstallProgress) {}); err == nil {
		t.Fatal("ran an installer that failed its signature check")
	}
	if len(*ran) != 0 {
		t.Fatalf("ran %v", *ran)
	}
}
