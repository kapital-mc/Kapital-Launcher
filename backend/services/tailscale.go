package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"kapital/backend/models"
)

// Tailscale is the player's own Tailscale, as its command line reports it.
// The launcher never runs Tailscale's network itself: it finds the CLI the
// official app installs, asks it for its state, and on the player's click
// starts its sign-in, whose page opens in the browser. It reads no account
// data and changes no Tailscale setting.

// EventTailscaleState carries a models.TailscaleInfo whenever it changes after
// an install, a sign-in or Open Tailscale.
const EventTailscaleState = "tailscale:state"

// Tailscale states, models.TailscaleInfo.State.
const (
	TailscaleMissing       = "missing"
	TailscaleStopped       = "stopped"
	TailscaleStarting      = "starting"
	TailscaleNeedsLogin    = "needsLogin"
	TailscaleNeedsApproval = "needsApproval"
	TailscaleRunning       = "running"
)

const (
	tailscaleStatusTimeout = 8 * time.Second
	tailscaleLoginTimeout  = 10 * time.Minute
	maxTailscaleStatus     = 4 << 20 // the status lists every peer
)

// tailscaleLoginURL is the sign-in page `tailscale login` prints. Only an
// address of this shape is opened in the browser.
var tailscaleLoginURL = regexp.MustCompile(`https://login\.tailscale\.com/a/[A-Za-z0-9]+`)

// TailscaleCLI finds and runs Tailscale's command line. Its OS calls are
// injected so it is tested on a machine with no Tailscale.
type TailscaleCLI struct {
	goos     string
	lookPath func(string) (string, error)
	stat     func(string) (os.FileInfo, error)
	getenv   func(string) string
	run      func(ctx context.Context, exe string, args ...string) ([]byte, error)
}

// NewTailscaleCLI returns the CLI for goos with the real OS calls.
func NewTailscaleCLI(goos string) *TailscaleCLI {
	return &TailscaleCLI{
		goos:     goos,
		lookPath: exec.LookPath,
		stat:     os.Stat,
		getenv:   os.Getenv,
		run: func(ctx context.Context, exe string, args ...string) ([]byte, error) {
			return tailscaleCommand(ctx, exe, args...).Output()
		},
	}
}

// tailscaleCommand is the CLI as a process: an argument array, its console
// hidden on Windows, and TAILSCALE_BE_CLI set so the macOS app's executable
// acts as the CLI instead of opening its window.
func tailscaleCommand(ctx context.Context, exe string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(), "TAILSCALE_BE_CLI=1")
	hideSyncConsole(cmd)
	return cmd
}

// cliPaths are where the official app puts the CLI, then PATH.
func (t *TailscaleCLI) cliPaths() []string {
	switch t.goos {
	case "windows":
		// The MSI's default INSTALLDIR (tailscale.com/kb/1189); the file name is [verify].
		return []string{t.getenv("ProgramFiles") + `\Tailscale\tailscale.exe`}
	case "darwin":
		// The Standalone and App Store apps carry the CLI in their executable;
		// /usr/local/bin/tailscale is the launcher the app's settings install.
		return []string{"/Applications/Tailscale.app/Contents/MacOS/Tailscale", "/usr/local/bin/tailscale"}
	}
	return nil
}

// Exe is the CLI's path, "" when Tailscale is not installed.
func (t *TailscaleCLI) Exe() string {
	for _, p := range t.cliPaths() {
		if info, err := t.stat(p); err == nil && info.Mode().IsRegular() {
			return p
		}
	}
	if p, err := t.lookPath("tailscale"); err == nil {
		return p
	}
	return ""
}

// CanInstall is whether the launcher can get Tailscale on this OS.
func (t *TailscaleCLI) CanInstall() bool { return t.goos == "windows" || t.goos == "darwin" }

// Detect asks the CLI for Tailscale's state. A CLI that cannot reach its
// daemon, or answers with something unreadable, is a Tailscale that is not
// running.
func (t *TailscaleCLI) Detect(ctx context.Context) models.TailscaleInfo {
	info := models.TailscaleInfo{State: TailscaleMissing, CanInstall: t.CanInstall()}
	exe := t.Exe()
	if exe == "" {
		return info
	}
	ctx, cancel := context.WithTimeout(ctx, tailscaleStatusTimeout)
	defer cancel()
	// `status` exits non-zero when Tailscale is stopped but still prints its
	// JSON, so the output is read whatever the exit.
	out, err := t.run(ctx, exe, "status", "--json")
	info.State, info.Version = parseTailscaleStatus(out)
	if info.State == TailscaleStopped && err != nil {
		info.Version = t.version(ctx, exe)
	}
	return info
}

// version is `tailscale version`'s first line, for a Tailscale whose status
// could not be read.
func (t *TailscaleCLI) version(ctx context.Context, exe string) string {
	out, err := t.run(ctx, exe, "version")
	if err != nil {
		return ""
	}
	first := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return shortTailscaleVersion(first)
}

// parseTailscaleStatus reads the two fields of `tailscale status --json` the
// launcher uses: BackendState and Version. Everything else (peers, users,
// addresses) is ignored and never kept.
func parseTailscaleStatus(out []byte) (state, version string) {
	if len(out) > maxTailscaleStatus {
		return TailscaleStopped, ""
	}
	var doc struct {
		BackendState string
		Version      string
	}
	if json.Unmarshal(out, &doc) != nil {
		return TailscaleStopped, ""
	}
	version = shortTailscaleVersion(doc.Version)
	switch doc.BackendState {
	case "Running":
		return TailscaleRunning, version
	case "NeedsLogin":
		return TailscaleNeedsLogin, version
	case "NeedsMachineAuth":
		return TailscaleNeedsApproval, version
	case "Starting":
		return TailscaleStarting, version
	}
	return TailscaleStopped, version
}

// shortTailscaleVersion drops the build suffix: "1.104.1-t1234-g5678" is "1.104.1".
func shortTailscaleVersion(v string) string {
	v = strings.SplitN(strings.TrimSpace(v), "-", 2)[0]
	if len(v) > 32 {
		return ""
	}
	return v
}

// SignIn runs `tailscale login`, hands the sign-in page it prints to open
// (the browser) and waits for the sign-in to finish, at most
// tailscaleLoginTimeout. A page that is not Tailscale's own login address is
// never opened.
func (t *TailscaleCLI) SignIn(ctx context.Context, open func(string) error) error {
	exe := t.Exe()
	if exe == "" {
		return errors.New("Tailscale is not installed")
	}
	ctx, cancel := context.WithTimeout(ctx, tailscaleLoginTimeout)
	defer cancel()
	cmd := tailscaleCommand(ctx, exe, "login")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Tailscale's sign-in: %w", err)
	}
	opened, scanErr := openLoginURL(pipe, open)
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return errors.New("the Tailscale sign-in was not finished in time")
		}
		return fmt.Errorf("Tailscale's sign-in: %w", err)
	}
	if scanErr != nil && !opened {
		return scanErr
	}
	return nil
}

// openLoginURL reads the CLI's output until its end, opening the first
// sign-in page in it. It reports whether it opened one.
func openLoginURL(r io.Reader, open func(string) error) (bool, error) {
	opened := false
	var openErr error
	sc := bufio.NewScanner(io.LimitReader(r, 1<<20))
	for sc.Scan() {
		if opened {
			continue
		}
		link := tailscaleLoginURL.FindString(sc.Text())
		if link == "" || !isTailscaleLoginURL(link) {
			continue
		}
		opened = true
		// The page's address is a one-time key to this machine's sign-in, so
		// an error that might carry it is not passed on to the log.
		if open(link) != nil {
			openErr = errors.New("could not open Tailscale's sign-in page")
		}
	}
	return opened, errors.Join(openErr, sc.Err())
}

func isTailscaleLoginURL(link string) bool {
	u, err := url.Parse(link)
	return err == nil && u.Scheme == "https" && u.Host == "login.tailscale.com" && u.User == nil
}
