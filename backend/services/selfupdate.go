package services

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"kapital/backend/models"
)

// The launcher's own update (ADR-6 amendment, 2026-10-21). Once the download
// site says a newer release is out (CheckUpdate), Update downloads that
// release's build for this system from its GitHub release, checks it against
// the release's checksums.txt, whose ed25519 signature must verify against a
// key built into this copy (updatekeys.go), and stages it in a hidden folder
// beside the running copy. Restart swaps the two with renames, rolling back on
// a failure, and the new copy is started once this one has quit:
//
//	macOS    the whole Kapital Launcher.app, unpacked by ditto and checked by codesign
//	Windows  kapital-launcher.exe, which Windows lets a running process rename
//
// The new copy removes the staging folder, with the old copy in it, at start.
// The download is written by the launcher, not a browser, so it carries no
// quarantine flag or Mark of the Web: the signature is the only gate.

// EventLauncherUpdate carries a models.LauncherUpdateProgress per step.
const EventLauncherUpdate = "launcher:update"

const (
	// aislop-ignore-next-line ai-slop/hardcoded-url -- fixed on purpose: only this repository's releases are installed
	launcherReleaseBase = "https://github.com/sandrogekeler/Kapital-Launcher/releases/download/"
	// checksumsName and signatureName are the release's checksum file and its
	// signature, written by the release workflow (scripts/update-sign).
	checksumsName      = "checksums.txt"
	signatureName      = "checksums.txt.sig"
	maxChecksums       = 64 << 10
	maxSignature       = 1 << 10
	maxLauncherAsset   = 200 << 20 // today's builds are 10 to 30 MB
	launcherFetchLimit = 30 * time.Second
	launcherStall      = 60 * time.Second
	launcherDownload   = time.Hour
	// stagingPrefix names the hidden folder an update is staged in, beside
	// the running copy; any folder of this name there is a leftover.
	stagingPrefix = ".kapital-update-"
)

// errNoUpdateKey is a build with no key to check an update against.
var errNoUpdateKey = errors.New("this build has no update key, so it cannot install updates itself")

// LauncherUpdater downloads, stages and applies a newer build of the launcher.
type LauncherUpdater struct {
	goos         string
	base         string
	client       *http.Client
	allowedHosts []string
	keys         []ed25519.PublicKey
	// executable is the running program's path; a test swaps it.
	executable func() (string, error)
	stall      time.Duration
	// platform holds what differs per system: the asset, the target, the
	// checks of an unpacked build, and how it is started.
	platform updatePlatform

	mu     sync.Mutex
	busy   bool
	staged *stagedUpdate
}

// updatePlatform is the per-system half of the updater.
type updatePlatform struct {
	// asset is the release file for this system.
	asset func(tag string) string
	// target is what an update replaces, found from the running executable:
	// the .app bundle on macOS, the exe on Windows.
	target func(exe string) (string, error)
	// prepare turns the verified download into the replacement, inside dir.
	prepare func(ctx context.Context, dir, download string) (string, error)
	// relaunch starts the copy at target and returns without waiting.
	relaunch func(target string) error
	// placed runs after a swap; Windows records the version for Apps and features.
	placed func(target, tag string)
	// hide hides the staging folder where a dot does not.
	hide func(dir string)
}

// stagedUpdate is a download that passed every check and waits for Restart.
type stagedUpdate struct {
	tag         string
	dir         string
	target      string
	replacement string
}

// NewLauncherUpdater is the updater for this system and this build's keys.
func NewLauncherUpdater() *LauncherUpdater {
	return newLauncherUpdater(runtime.GOOS, launcherReleaseBase, updateKeys(), systemUpdatePlatform())
}

func newLauncherUpdater(goos, base string, keys []ed25519.PublicKey, p updatePlatform) *LauncherUpdater {
	u := &LauncherUpdater{
		goos: goos, base: base, keys: keys, platform: p,
		allowedHosts: prismDownloadHosts, executable: os.Executable, stall: launcherStall,
	}
	u.client = &http.Client{CheckRedirect: u.checkRedirect}
	return u
}

// Target is what an update would replace, or why this copy cannot update
// itself, in a sentence for the player.
func (u *LauncherUpdater) Target() (string, error) {
	if len(u.keys) == 0 {
		return "", errNoUpdateKey
	}
	exe, err := u.executable()
	if err != nil {
		return "", fmt.Errorf("the launcher could not find its own program: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	target, err := u.platform.target(exe)
	if err != nil {
		return "", err
	}
	if err := checkWritable(filepath.Dir(target)); err != nil {
		return "", fmt.Errorf("the launcher cannot write to %s, where it is installed", filepath.Dir(target))
	}
	return target, nil
}

// checkWritable creates and removes a file in dir.
func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, stagingPrefix+"probe-*")
	if err != nil {
		return err
	}
	name := f.Name()
	return errors.Join(f.Close(), os.Remove(name))
}

// Ready is the release staged for Restart, or "" when none is.
func (u *LauncherUpdater) Ready() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.staged == nil {
		return ""
	}
	return u.staged.tag
}

// Download fetches, checks and stages release tag, reporting each step. One
// runs at a time; a staged update of the same tag is kept and reported ready.
func (u *LauncherUpdater) Download(ctx context.Context, tag string, progress func(models.LauncherUpdateProgress)) (err error) {
	u.mu.Lock()
	if u.busy {
		u.mu.Unlock()
		return errors.New("the update is already downloading")
	}
	if u.staged != nil && u.staged.tag == tag {
		u.mu.Unlock()
		progress(models.LauncherUpdateProgress{Phase: "ready", Version: tag})
		return nil
	}
	u.busy = true
	u.mu.Unlock()
	defer func() {
		u.mu.Lock()
		u.busy = false
		u.mu.Unlock()
		if err != nil {
			progress(models.LauncherUpdateProgress{Phase: "failed", Version: tag, Error: err.Error()})
		}
	}()

	staged, err := u.stage(ctx, tag, progress)
	if err != nil {
		return err
	}
	u.mu.Lock()
	previous := u.staged
	u.staged = staged
	u.mu.Unlock()
	if previous != nil {
		removeQuietly(previous.dir)
	}
	slog.Info("launcher update staged", "version", tag)
	progress(models.LauncherUpdateProgress{Phase: "ready", Version: tag})
	return nil
}

// stage is Download's work: the checksums and their signature, the asset,
// and the system's own checks, all inside a fresh staging folder.
func (u *LauncherUpdater) stage(ctx context.Context, tag string, progress func(models.LauncherUpdateProgress)) (*stagedUpdate, error) {
	if _, ok := parseRelease(tag); !ok || !strings.HasPrefix(tag, "v") {
		return nil, fmt.Errorf("refusing launcher version %q", tag)
	}
	target, err := u.Target()
	if err != nil {
		return nil, err
	}
	asset := u.platform.asset(tag)
	sums, err := u.fetchSmall(ctx, tag, checksumsName, maxChecksums)
	if err != nil {
		return nil, err
	}
	sig, err := u.fetchSmall(ctx, tag, signatureName, maxSignature)
	if err != nil {
		return nil, err
	}
	if err := verifyChecksums(u.keys, sums, sig); err != nil {
		return nil, err
	}
	want, err := checksumFor(sums, asset)
	if err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp(filepath.Dir(target), stagingPrefix)
	if err != nil {
		return nil, fmt.Errorf("prepare the update: %w", err)
	}
	if u.platform.hide != nil {
		u.platform.hide(dir)
	}
	download := filepath.Join(dir, asset)
	if err := u.download(ctx, tag, asset, want, download, progress); err != nil {
		removeQuietly(dir)
		return nil, err
	}
	progress(models.LauncherUpdateProgress{Phase: "verifying", Version: tag})
	replacement, err := u.platform.prepare(ctx, dir, download)
	if err != nil {
		removeQuietly(dir)
		return nil, err
	}
	return &stagedUpdate{tag: tag, dir: dir, target: target, replacement: replacement}, nil
}

// fetchSmall reads one small file of the release whole.
func (u *LauncherUpdater) fetchSmall(ctx context.Context, tag, name string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, launcherFetchLimit)
	defer cancel()
	resp, err := u.get(ctx, tag, name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", name, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("the release's %s is larger than expected", name)
	}
	return body, nil
}

// get requests one file of the release and checks the answer is a 200.
func (u *LauncherUpdater) get(ctx context.Context, tag, name string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.base+tag+"/"+name, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", prismUserAgent)
	if err := u.checkHost(req.URL); err != nil {
		return nil, err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", name, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() //nolint:errcheck,gosec // the status is the answer
		return nil, fmt.Errorf("download %s: %s", name, resp.Status)
	}
	return resp, nil
}

// download streams the asset to path, checking its size and SHA-256.
func (u *LauncherUpdater) download(ctx context.Context, tag, asset, want, path string, progress func(models.LauncherUpdateProgress)) error {
	ctx, cancel := context.WithTimeout(ctx, launcherDownload)
	defer cancel()
	stalled := time.AfterFunc(u.stall, cancel)
	defer stalled.Stop()
	resp, err := u.get(ctx, tag, asset)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	total := resp.ContentLength
	if total > maxLauncherAsset {
		return fmt.Errorf("the update is %d bytes, more than the launcher accepts", total)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	h := sha256.New()
	counter := &updateCounter{total: total, tag: tag, report: progress, alive: func() { stalled.Reset(u.stall) }}
	n, err := io.Copy(io.MultiWriter(f, h, counter), io.LimitReader(resp.Body, maxLauncherAsset+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	switch {
	case err != nil:
		return fmt.Errorf("download the update: %w", err)
	case n > maxLauncherAsset:
		return errors.New("the update is larger than the launcher accepts")
	case hex.EncodeToString(h.Sum(nil)) != want:
		return errors.New("the update does not match the release's signed checksum")
	}
	return nil
}

func (u *LauncherUpdater) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("too many redirects")
	}
	return u.checkHost(req.URL)
}

func (u *LauncherUpdater) checkHost(addr *url.URL) error {
	if addr.Scheme != "https" || !slices.Contains(u.allowedHosts, addr.Hostname()) {
		return fmt.Errorf("refusing to fetch the update from %s", addr.Redacted())
	}
	return nil
}

// verifyChecksums checks sig, base64 on one line, against every key.
func verifyChecksums(keys []ed25519.PublicKey, sums, sig []byte) error {
	if len(keys) == 0 {
		return errNoUpdateKey
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || len(raw) != ed25519.SignatureSize {
		return errors.New("the release's checksum signature is malformed")
	}
	for _, k := range keys {
		if ed25519.Verify(k, sums, raw) {
			return nil
		}
	}
	return errors.New("the release's checksums are not signed with this launcher's update key")
}

// checksumFor finds asset's SHA-256 in sha256sum's format: the hex digest,
// two spaces (or a space and a star) and the file name.
func checksumFor(sums []byte, asset string) (string, error) {
	for line := range strings.Lines(string(sums)) {
		digest, name, ok := strings.Cut(strings.TrimRight(line, "\r\n"), " ")
		if !ok || strings.TrimLeft(name, " *") != asset {
			continue
		}
		if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 {
			return "", fmt.Errorf("the checksum of %s is malformed", asset)
		}
		return strings.ToLower(digest), nil
	}
	return "", fmt.Errorf("the release has no %s", asset)
}

// Apply swaps the staged update into place and returns the path to start. On
// a failure the running copy is put back where it was and nothing is staged.
func (u *LauncherUpdater) Apply() (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.busy {
		return "", errors.New("the update is still downloading")
	}
	s := u.staged
	if s == nil {
		return "", errors.New("no update is ready")
	}
	u.staged = nil
	previous := filepath.Join(s.dir, "previous"+filepath.Ext(s.target))
	if err := os.Rename(s.target, previous); err != nil {
		removeQuietly(s.dir)
		return "", fmt.Errorf("move the running launcher aside: %w", err)
	}
	if err := os.Rename(s.replacement, s.target); err != nil {
		if rerr := os.Rename(previous, s.target); rerr != nil {
			slog.Error("launcher update: roll back", "target", s.target, "error", rerr)
			return "", fmt.Errorf("place the update: %w; putting the old launcher back also failed: %w", err, rerr)
		}
		removeQuietly(s.dir)
		return "", fmt.Errorf("place the update: %w", err)
	}
	if u.platform.placed != nil {
		u.platform.placed(s.target, s.tag)
	}
	slog.Info("launcher updated in place", "version", s.tag, "target", s.target)
	return s.target, nil
}

// Relaunch starts the copy at target without waiting for it.
func (u *LauncherUpdater) Relaunch(target string) error {
	return u.platform.relaunch(target)
}

// CleanUp removes what earlier updates left beside this copy: the old copy a
// restart swapped out and any download that was never applied. On Windows the
// old exe stays locked until the process that ran it has exited, so a removal
// is tried a few times.
func (u *LauncherUpdater) CleanUp(ctx context.Context) {
	exe, err := u.executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	target, err := u.platform.target(exe)
	if err != nil {
		return
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(target), stagingPrefix+"*"))
	if err != nil || len(leftovers) == 0 {
		return
	}
	for try := range 5 {
		leftovers = slices.DeleteFunc(leftovers, func(dir string) bool { return os.RemoveAll(dir) == nil })
		if len(leftovers) == 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(try+1) * time.Second):
		}
	}
	slog.Warn("launcher update: leftovers not removed", "paths", leftovers)
}

// updateCounter turns bytes written into download progress, a few per megabyte.
type updateCounter struct {
	total, n, last int64
	tag            string
	report         func(models.LauncherUpdateProgress)
	alive          func()
}

func (c *updateCounter) Write(b []byte) (int, error) {
	c.alive()
	c.n += int64(len(b))
	if c.n-c.last >= progressEvery || c.n == c.total {
		c.last = c.n
		c.report(models.LauncherUpdateProgress{Phase: "downloading", Received: c.n, Total: c.total, Version: c.tag})
	}
	return len(b), nil
}
