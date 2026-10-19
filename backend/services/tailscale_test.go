package services

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func fakeCLI(goos string, files map[string]bool, path string, run func(args ...string) ([]byte, error)) *TailscaleCLI {
	return &TailscaleCLI{
		goos: goos,
		stat: func(p string) (os.FileInfo, error) {
			if files[p] {
				return fakeFile{}, nil
			}
			return nil, os.ErrNotExist
		},
		lookPath: func(string) (string, error) {
			if path == "" {
				return "", errors.New("not found")
			}
			return path, nil
		},
		getenv: func(k string) string {
			if k == "ProgramFiles" {
				return `C:\Program Files`
			}
			return ""
		},
		run: func(_ context.Context, _ string, args ...string) ([]byte, error) { return run(args...) },
	}
}

func TestTailscaleCLIFindsTheOfficialApp(t *testing.T) {
	none := func(...string) ([]byte, error) { return nil, nil }
	mac := fakeCLI("darwin", map[string]bool{"/Applications/Tailscale.app/Contents/MacOS/Tailscale": true}, "", none)
	if got := mac.Exe(); got != "/Applications/Tailscale.app/Contents/MacOS/Tailscale" {
		t.Errorf("macOS: %q", got)
	}
	win := fakeCLI("windows", map[string]bool{`C:\Program Files\Tailscale\tailscale.exe`: true}, "", none)
	if got := win.Exe(); got != `C:\Program Files\Tailscale\tailscale.exe` {
		t.Errorf("Windows: %q", got)
	}
	linux := fakeCLI("linux", nil, "/usr/bin/tailscale", none)
	if got := linux.Exe(); got != "/usr/bin/tailscale" {
		t.Errorf("PATH: %q", got)
	}
	if fakeCLI("linux", nil, "", none).Exe() != "" {
		t.Error("found a Tailscale that is not there")
	}
	if linux.CanInstall() || !mac.CanInstall() || !win.CanInstall() {
		t.Error("CanInstall is wrong")
	}
}

func TestTailscaleDetectReadsTheBackendState(t *testing.T) {
	cases := map[string]struct {
		out   string
		err   error
		state string
	}{
		"running":          {`{"BackendState":"Running","Version":"1.104.1-t123-g456","Peer":{}}`, nil, TailscaleRunning},
		"needs login":      {`{"BackendState":"NeedsLogin","Version":"1.104.1"}`, errors.New("exit 1"), TailscaleNeedsLogin},
		"needs approval":   {`{"BackendState":"NeedsMachineAuth"}`, nil, TailscaleNeedsApproval},
		"starting":         {`{"BackendState":"Starting"}`, nil, TailscaleStarting},
		"stopped":          {`{"BackendState":"Stopped"}`, errors.New("exit 1"), TailscaleStopped},
		"no daemon":        {"failed to connect to local tailscaled", errors.New("exit 1"), TailscaleStopped},
		"not json at all":  {"", nil, TailscaleStopped},
		"an unknown state": {`{"BackendState":"Weird"}`, nil, TailscaleStopped},
	}
	for name, tc := range cases {
		cli := fakeCLI("linux", nil, "/usr/bin/tailscale", func(args ...string) ([]byte, error) {
			if args[0] == "version" {
				return []byte("1.100.0\n  tailscale commit: abc\n"), nil
			}
			return []byte(tc.out), tc.err
		})
		info, _ := cli.Detect(context.Background())
		if info.State != tc.state {
			t.Errorf("%s: state %q want %q", name, info.State, tc.state)
		}
	}
	running, _ := fakeCLI("linux", nil, "/usr/bin/tailscale", func(...string) ([]byte, error) {
		return []byte(`{"BackendState":"Running","Version":"1.104.1-t123-g456"}`), nil
	}).Detect(context.Background())
	if running.Version != "1.104.1" {
		t.Errorf("version %q", running.Version)
	}
	stopped, _ := fakeCLI("linux", nil, "/usr/bin/tailscale", func(args ...string) ([]byte, error) {
		if args[0] == "version" {
			return []byte("1.100.0\n"), nil
		}
		return nil, errors.New("exit 1")
	}).Detect(context.Background())
	if stopped.Version != "1.100.0" {
		t.Errorf("stopped version %q", stopped.Version)
	}
	missing, _ := fakeCLI("windows", nil, "", nil).Detect(context.Background())
	if missing.State != TailscaleMissing || !missing.CanInstall {
		t.Errorf("missing: %+v", missing)
	}
}

func TestOpenLoginURLOpensOnlyTailscalesPage(t *testing.T) {
	var opened []string
	open := func(u string) error { opened = append(opened, u); return nil }
	out := "\nTo authenticate, visit:\n\n\thttps://login.tailscale.com/a/abc123XYZ\n\nSuccess.\n" +
		"https://login.tailscale.com/a/second\n"
	ok, err := openLoginURL(strings.NewReader(out), open)
	if !ok || err != nil || len(opened) != 1 || opened[0] != "https://login.tailscale.com/a/abc123XYZ" {
		t.Fatalf("opened %v, %v %v", opened, ok, err)
	}
	opened = nil
	for _, bad := range []string{
		"visit https://login.tailscale.com.evil.example/a/abc",
		"visit http://login.tailscale.com/a/abc",
		"visit https://evil.example/https://login.tailscale.com/",
		"visit https://evil.example/?u=https://login.tailscale.com/a/abc",
		"visit https://login.tailscale.com/a/abc?next=https://evil.example",
		"visit https://login.tailscale.com/admin",
	} {
		if ok, _ := openLoginURL(strings.NewReader(bad), open); ok || len(opened) != 0 {
			t.Errorf("opened %q from %q", opened, bad)
		}
	}
	if isTailscaleLoginURL("https://user@login.tailscale.com/a/x") {
		t.Error("a URL with user info passed")
	}
}

func TestTailscaleSignInWithoutTailscale(t *testing.T) {
	cli := fakeCLI("linux", nil, "", nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := cli.SignIn(ctx, func(string) error { return nil }); err == nil {
		t.Fatal("signed in without Tailscale")
	}
}
