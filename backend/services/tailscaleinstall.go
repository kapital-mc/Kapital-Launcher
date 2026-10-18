package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"kapital/backend/models"
)

// TailscaleInstaller gets Tailscale for a player who has none, on approval
// only: the official installer from Tailscale's own package server, checked
// against the SHA-256 published beside it (and on macOS against Tailscale's
// Developer ID), then run. The installer is Tailscale's and asks for what it
// needs itself: an administrator's approval on Windows, and on macOS the
// Installer's steps and then Tailscale's system extension and VPN prompts.

// EventTailscaleInstall carries a models.TailscaleInstallProgress per step.
const EventTailscaleInstall = "tailscale:install"

const (
	tailscalePackages      = "https://pkgs.tailscale.com/stable/"
	tailscaleHost          = "pkgs.tailscale.com"
	maxTailscaleIndex      = 1 << 20
	maxTailscaleInstaller  = 300 << 20 // today's are 22 to 40 MB
	tailscaleAPITimeout    = 15 * time.Second
	tailscaleDownloadLimit = 2 * time.Hour
	tailscaleDownloadStall = 60 * time.Second
)

var (
	tailscaleVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	tailscaleFile    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}\.(msi|pkg)$`)
	sha256Hex        = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// TailscaleInstaller downloads and runs Tailscale's installer.
type TailscaleInstaller struct {
	goos, goarch string
	dir          string // where the installer is downloaded
	base         string // tailscalePackages, a test server in tests
	client       *http.Client
	stall        time.Duration
	// runInstaller starts the downloaded installer and waits for it.
	runInstaller func(ctx context.Context, path string) error
	// verify checks the installer's signature where the OS can.
	verify func(ctx context.Context, path string) error
	mu     sync.Mutex
}

// NewTailscaleInstaller downloads into <dataDir>/tailscale.
func NewTailscaleInstaller(dataDir, goos, goarch string) *TailscaleInstaller {
	t := &TailscaleInstaller{
		goos: goos, goarch: goarch,
		dir:          filepath.Join(dataDir, "tailscale"),
		base:         tailscalePackages,
		stall:        tailscaleDownloadStall,
		runInstaller: runTailscaleInstaller,
		verify:       verifyTailscaleInstaller,
	}
	t.client = &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return t.checkHost(req.URL)
	}}
	return t
}

func (t *TailscaleInstaller) checkHost(u *url.URL) error {
	base, err := url.Parse(t.base)
	if err != nil || u.Scheme != base.Scheme || u.Host != base.Host {
		return fmt.Errorf("refusing to fetch Tailscale from %s", u.Redacted())
	}
	return nil
}

// installerFile picks this OS's installer from the package index: the MSI
// for the CPU on Windows, the Standalone .pkg on macOS.
func installerFile(goos, goarch string, msis, macs map[string]string) (string, error) {
	switch goos {
	case "windows":
		if f, ok := msis[goarch]; ok {
			return f, nil
		}
	case "darwin":
		if f, ok := macs["universal-package"]; ok {
			return f, nil
		}
	}
	return "", fmt.Errorf("Tailscale has no installer the launcher can run on %s/%s", goos, goarch)
}

// Latest reads Tailscale's stable package index and the installer's size and
// published digest, without downloading it.
func (t *TailscaleInstaller) Latest(ctx context.Context) (models.TailscaleRelease, error) {
	ctx, cancel := context.WithTimeout(ctx, tailscaleAPITimeout)
	defer cancel()
	var index struct {
		MSIs           map[string]string
		MSIsVersion    string
		MacZips        map[string]string
		MacZipsVersion string
	}
	body, _, err := t.get(ctx, http.MethodGet, t.base+"?mode=json", maxTailscaleIndex)
	if err != nil {
		return models.TailscaleRelease{}, fmt.Errorf("read Tailscale's packages: %w", err)
	}
	if err := json.Unmarshal(body, &index); err != nil {
		return models.TailscaleRelease{}, fmt.Errorf("read Tailscale's packages: %w", err)
	}
	file, err := installerFile(t.goos, t.goarch, index.MSIs, index.MacZips)
	if err != nil {
		return models.TailscaleRelease{}, err
	}
	version := index.MSIsVersion
	if t.goos == "darwin" {
		version = index.MacZipsVersion
	}
	if !tailscaleVersion.MatchString(version) || !tailscaleFile.MatchString(file) || !strings.Contains(file, version) {
		return models.TailscaleRelease{}, fmt.Errorf("Tailscale's package index names an unexpected installer %q", file)
	}
	rel := models.TailscaleRelease{Version: version, File: file, URL: t.base + file}
	if _, rel.Size, err = t.get(ctx, http.MethodHead, rel.URL, 0); err != nil {
		return models.TailscaleRelease{}, fmt.Errorf("read the size of %s: %w", file, err)
	}
	if rel.Size <= 0 || rel.Size > maxTailscaleInstaller {
		return models.TailscaleRelease{}, fmt.Errorf("%s is %d bytes, outside the expected size", file, rel.Size)
	}
	sum, _, err := t.get(ctx, http.MethodGet, rel.URL+".sha256", 256)
	if err != nil {
		return models.TailscaleRelease{}, fmt.Errorf("read the digest of %s: %w", file, err)
	}
	digest := strings.ToLower(strings.Fields(string(sum) + " ")[0])
	if !sha256Hex.MatchString(digest) {
		return models.TailscaleRelease{}, fmt.Errorf("%s has no SHA-256 digest; refusing an unverifiable download", file)
	}
	rel.Digest = "sha256:" + digest
	return rel, nil
}

// get fetches one address on the package server: a body of at most limit
// bytes for GET, the length for HEAD.
func (t *TailscaleInstaller) get(ctx context.Context, method, address string, limit int64) ([]byte, int64, error) {
	req, err := http.NewRequestWithContext(ctx, method, address, nil)
	if err != nil {
		return nil, 0, err
	}
	if err := t.checkHost(req.URL); err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", prismUserAgent)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	if resp.StatusCode != http.StatusOK {
		return nil, 0, errors.New(resp.Status)
	}
	if method == http.MethodHead {
		return nil, resp.ContentLength, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err == nil && int64(len(body)) > limit {
		err = errors.New("response too large")
	}
	return body, int64(len(body)), err
}

// Install downloads rel, checks it, runs it and waits for the installer to
// finish, reporting each step. The download is removed afterwards.
func (t *TailscaleInstaller) Install(ctx context.Context, rel models.TailscaleRelease, progress func(models.TailscaleInstallProgress)) (err error) {
	if !t.mu.TryLock() {
		return errors.New("Tailscale is already being installed")
	}
	defer t.mu.Unlock()
	defer func() {
		if err != nil {
			progress(models.TailscaleInstallProgress{Phase: "failed", Error: err.Error()})
		}
	}()
	if rel.URL != t.base+rel.File || !tailscaleFile.MatchString(rel.File) {
		return fmt.Errorf("refusing to download Tailscale from %q", rel.URL)
	}
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(t.dir, rel.File)
	defer removeQuietly(path)
	if err := t.download(ctx, rel, path, progress); err != nil {
		return err
	}
	progress(models.TailscaleInstallProgress{Phase: "verifying", Received: rel.Size, Total: rel.Size})
	if err := t.verify(ctx, path); err != nil {
		return err
	}
	progress(models.TailscaleInstallProgress{Phase: "installing", Received: rel.Size, Total: rel.Size})
	if err := t.runInstaller(ctx, path); err != nil {
		return err
	}
	progress(models.TailscaleInstallProgress{Phase: "done", Received: rel.Size, Total: rel.Size})
	return nil
}

func (t *TailscaleInstaller) download(ctx context.Context, rel models.TailscaleRelease, path string, progress func(models.TailscaleInstallProgress)) error {
	want := strings.TrimPrefix(rel.Digest, "sha256:")
	if !sha256Hex.MatchString(want) || rel.Size <= 0 || rel.Size > maxTailscaleInstaller {
		return errors.New("refusing a Tailscale download without a digest and a size")
	}
	ctx, cancel := context.WithTimeout(ctx, tailscaleDownloadLimit)
	defer cancel()
	stalled := time.AfterFunc(t.stall, cancel)
	defer stalled.Stop()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", prismUserAgent)
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("download Tailscale: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download Tailscale: %s", resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	counter := &progressWriter{total: rel.Size, alive: func() { stalled.Reset(t.stall) }, report: func(p models.PrismInstallProgress) {
		progress(models.TailscaleInstallProgress{Phase: "downloading", Received: p.Received, Total: p.Total})
	}}
	n, err := io.Copy(io.MultiWriter(f, h, counter), io.LimitReader(resp.Body, rel.Size+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("download Tailscale: %w", err)
	}
	if n != rel.Size {
		return fmt.Errorf("Tailscale's installer is %d bytes, the package server said %d", n, rel.Size)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("Tailscale's installer does not match its published digest (got %s)", got)
	}
	return nil
}
