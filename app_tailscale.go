package main

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"kapital/backend/models"
	"kapital/backend/services"
)

// tailscaleDownloadPage is where Get Tailscale goes on an OS the launcher
// cannot install it on.
const tailscaleDownloadPage = "https://tailscale.com/download"

// How long, and how often, Go looks at Tailscale's state after the player
// did something that changes it (an install, a sign-in, Open Tailscale), so
// the settings page follows with no polling of its own.
const (
	tailscaleWatchEvery = 3 * time.Second
	tailscaleWatchFor   = 15 * time.Minute
)

// tailscaleState is the App's side of the player's Tailscale: the CLI, the
// installer and what the last look found.
type tailscaleState struct {
	cli       *services.TailscaleCLI
	installer *services.TailscaleInstaller
	// open starts the Tailscale app; a test swaps it.
	open func() error
	// emitState and emitInstall take the events in place of the window; a
	// test sets them.
	emitState   func(models.TailscaleInfo)
	emitInstall func(models.TailscaleInstallProgress)

	mu        sync.Mutex
	signingIn bool
	watching  bool
	last      models.TailscaleInfo
}

func newTailscaleState(dataDir string) *tailscaleState {
	return &tailscaleState{
		cli:       services.NewTailscaleCLI(runtime.GOOS),
		installer: services.NewTailscaleInstaller(dataDir, runtime.GOOS, runtime.GOARCH),
		open:      services.OpenTailscaleApp,
	}
}

// GetTailscale reports whether Tailscale is installed and connected. It runs
// Tailscale's CLI, so the settings page asks when it opens and after a click,
// and the watch after an install or a sign-in emits the rest.
func (a *App) GetTailscale() (models.TailscaleInfo, error) {
	return a.lookAtTailscale(a.context()), nil
}

// GetTailscaleRelease reads the installer InstallTailscale would download,
// for the popup to show before anything is fetched.
func (a *App) GetTailscaleRelease() (models.TailscaleRelease, error) {
	return a.tailscale.installer.Latest(a.context())
}

// InstallTailscale downloads and runs Tailscale's installer after the player
// approved it. The release is read again here rather than taken from the
// caller, so what is run is always what Tailscale published. Progress arrives
// as tailscale:install events, and the state as tailscale:state once the
// installer is done.
func (a *App) InstallTailscale() error {
	ts := a.tailscale
	rel, err := ts.installer.Latest(a.context())
	if err != nil {
		a.emitTailscaleInstall(models.TailscaleInstallProgress{Phase: "failed", Error: err.Error()})
		return err
	}
	if err := ts.installer.Install(a.context(), rel, a.emitTailscaleInstall); err != nil {
		slog.Error("install tailscale", "version", rel.Version, "error", err)
		return err
	}
	slog.Info("tailscale installed", "version", rel.Version)
	a.watchTailscale()
	return nil
}

// SignInTailscale starts Tailscale's sign-in and returns: the sign-in page
// opens in the browser, and the state follows as tailscale:state events.
func (a *App) SignInTailscale() error {
	ts := a.tailscale
	ts.mu.Lock()
	if ts.signingIn {
		ts.mu.Unlock()
		return nil
	}
	ts.signingIn = true
	ts.mu.Unlock()
	a.watchTailscale()
	go func() {
		err := ts.cli.SignIn(a.context(), a.openURL)
		ts.mu.Lock()
		ts.signingIn = false
		ts.mu.Unlock()
		if err != nil {
			slog.Warn("tailscale sign-in", "error", err)
		}
		a.lookAtTailscale(a.context())
	}()
	return nil
}

// OpenTailscale opens the Tailscale app, for a Tailscale that is installed
// but not running: on macOS its first start asks for its system extension.
// Where the launcher cannot install Tailscale, it opens the download page.
func (a *App) OpenTailscale() error {
	ts := a.tailscale
	if ts.cli.Exe() == "" {
		return a.openURL(tailscaleDownloadPage)
	}
	if err := ts.open(); err != nil {
		return err
	}
	a.watchTailscale()
	return nil
}

// lookAtTailscale asks the CLI for the state, keeps it and emits it when it
// changed.
func (a *App) lookAtTailscale(ctx context.Context) models.TailscaleInfo {
	ts := a.tailscale
	info := ts.cli.Detect(ctx)
	ts.mu.Lock()
	info.SigningIn = ts.signingIn
	changed := info != ts.last
	ts.last = info
	ts.mu.Unlock()
	if changed {
		a.emitTailscaleState(info)
	}
	return info
}

// watchTailscale looks at the state every few seconds, one watch at a time,
// until Tailscale is connected or tailscaleWatchFor has passed.
func (a *App) watchTailscale() {
	ts := a.tailscale
	ts.mu.Lock()
	if ts.watching {
		ts.mu.Unlock()
		return
	}
	ts.watching = true
	ts.mu.Unlock()
	go func() {
		defer func() {
			ts.mu.Lock()
			ts.watching = false
			ts.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(a.context(), tailscaleWatchFor)
		defer cancel()
		tick := time.NewTicker(tailscaleWatchEvery)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			info := a.lookAtTailscale(ctx)
			if info.State == services.TailscaleRunning && !info.SigningIn {
				return
			}
		}
	}()
}

func (a *App) emitTailscaleState(info models.TailscaleInfo) {
	if a.tailscale.emitState != nil {
		a.tailscale.emitState(info)
		return
	}
	if a.ctx != nil {
		wailsrt.EventsEmit(a.ctx, services.EventTailscaleState, info)
	}
}

func (a *App) emitTailscaleInstall(p models.TailscaleInstallProgress) {
	if a.tailscale.emitInstall != nil {
		a.tailscale.emitInstall(p)
		return
	}
	if a.ctx != nil {
		wailsrt.EventsEmit(a.ctx, services.EventTailscaleInstall, p)
	}
}

// checkTailscaleSave is SaveSettings' part for the Tailscale switches: a
// choice of a Tailscale address whose switch is off is dropped (the chapter
// goes back to its default), and every switch must name a chapter with a
// Tailscale address.
func (a *App) checkTailscaleSave(settings *models.AppSettings) error {
	settings.ServerChoices = services.DropTailscaleChoices(a.manifest.Chapters, *settings)
	return services.ValidateTailscaleServers(a.manifest.Chapters, settings.TailscaleServers)
}

// syncTailscaleLists puts the Tailscale entry into, or takes it out of, the
// in-game server list of every chapter whose switch a save changed.
func (a *App) syncTailscaleLists(before, after models.AppSettings) {
	for _, c := range a.manifest.Chapters {
		if services.UsesTailscale(before, c.ID) != services.UsesTailscale(after, c.ID) {
			a.syncTailscaleList(after, c)
		}
	}
}

// syncTailscaleList makes the chapter's in-game server list match its
// Tailscale switch. It runs on a switch and again before every Play, so a list
// the game rewrote, or an instance installed after the switch, catches up. A
// failure is logged and changes nothing else: the list is a convenience.
func (a *App) syncTailscaleList(settings models.AppSettings, chapter models.Chapter) {
	addr, ok := services.TailscaleAddress(chapter)
	if !ok {
		return
	}
	dir := a.instanceDir(settings, a.realEngine(), chapter)
	if dir == "" {
		return
	}
	entry := services.ServerListEntry{Name: fmt.Sprintf("%s (%s)", chapter.Name, addr.Label), Address: addr.Address}
	wrote, err := services.SyncServerList(dir, entry, services.UsesTailscale(settings, chapter.ID))
	if err != nil {
		slog.Warn("tailscale server list", "chapter", chapter.ID, "error", err)
		return
	}
	if wrote {
		slog.Info("tailscale server list", "chapter", chapter.ID, "listed", services.UsesTailscale(settings, chapter.ID))
	}
}
