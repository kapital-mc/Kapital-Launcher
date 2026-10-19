package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
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
	// open starts the Tailscale app, and lookup resolves a chapter's
	// Tailscale address to compare with the peers; a test swaps them.
	open   func() error
	lookup services.LookupNetIP
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
		lookup:    services.DefaultLookupNetIP,
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
	if rel, ok := services.PreviewTailscaleRelease(a.previews.Launcher()); ok {
		return rel, nil
	}
	return a.tailscale.installer.Latest(a.context())
}

// InstallTailscale downloads and runs Tailscale's installer after the player
// approved it. The release is read again here rather than taken from the
// caller, so what is run is always what Tailscale published. Progress arrives
// as tailscale:install events, and the state as tailscale:state once the
// installer is done.
func (a *App) InstallTailscale() error {
	// Under a Tailscale preview (#124) the install is a made-up one that fails.
	if a.underTailscalePreview() {
		return a.playPreviewTailscaleInstall()
	}
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

// signInTailscale starts Tailscale's sign-in and returns: the sign-in page
// opens in the browser, and the state follows as tailscale:state events. What
// Go sees once it finished goes to after, when there is one.
func (a *App) signInTailscale(after func(models.TailscaleInfo)) {
	ts := a.tailscale
	ts.mu.Lock()
	if ts.signingIn {
		ts.mu.Unlock()
		return
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
		info := a.lookAtTailscale(a.context())
		if err == nil && after != nil {
			after(info)
		}
	}()
}

// ConnectTailscale takes the player one step towards reaching the chapter's
// server over Tailscale, from whatever Go sees now: a Tailscale that is not
// running is opened, one that is not signed in signs in, and a signed-in one
// that does not reach the server opens the server's share link
// (ServerAddress.invite) in the browser, where the player accepts it. The
// chapter id resolves through the manifest and the link is the manifest's,
// held to Tailscale's shape by ValidateManifest. Go watches the state after,
// and tailscale:state says when the server is reached.
func (a *App) ConnectTailscale(chapterID string) error {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return fmt.Errorf("no chapter %q", chapterID)
	}
	addr, ok := services.TailscaleAddress(chapter)
	if !ok {
		return fmt.Errorf("%s has no Tailscale address", chapter.Name)
	}
	if a.underTailscalePreview() {
		a.connectPreviewTailscale()
		return nil
	}
	info := a.lookAtTailscale(a.context())
	switch info.State {
	case services.TailscaleMissing:
		return errors.New("Tailscale is not installed")
	case services.TailscaleNeedsLogin:
		a.signInTailscale(func(after models.TailscaleInfo) {
			if after.State == services.TailscaleRunning && !slices.Contains(after.Connected, chapter.ID) {
				a.openTailscaleInvite(chapter, addr)
			}
		})
		return nil
	case services.TailscaleRunning:
		if slices.Contains(info.Connected, chapter.ID) {
			return nil
		}
		return a.openTailscaleInvite(chapter, addr)
	}
	return a.OpenTailscale()
}

// openTailscaleInvite opens the chapter's share link and watches for the
// server to appear. A server with no link is one whose owner gives access
// some other way.
func (a *App) openTailscaleInvite(chapter models.Chapter, addr models.ServerAddress) error {
	if addr.Invite == "" {
		return fmt.Errorf("%s has no Tailscale share link: ask the server's owner for access", chapter.Name)
	}
	if err := a.openURL(addr.Invite); err != nil {
		slog.Warn("tailscale invite", "chapter", chapter.ID, "error", err)
		return fmt.Errorf("could not open the share link for %s", chapter.Name)
	}
	slog.Info("tailscale invite opened", "chapter", chapter.ID)
	a.watchTailscale()
	return nil
}

// OpenTailscale opens the Tailscale app, for a Tailscale that is installed
// but not running: on macOS its first start asks for its system extension.
// Where the launcher cannot install Tailscale, it opens the download page.
func (a *App) OpenTailscale() error {
	if a.underTailscalePreview() {
		return nil
	}
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

// lookAtTailscale asks the CLI for the state and which chapters' servers it
// reaches, keeps it and emits it when it changed. Under a Tailscale preview
// the answer is the preview's, and nothing is asked or emitted.
func (a *App) lookAtTailscale(ctx context.Context) models.TailscaleInfo {
	if info, ok := a.previewTailscale(); ok {
		return info
	}
	ts := a.tailscale
	info, peers := ts.cli.Detect(ctx)
	info.Connected = services.TailscaleConnected(ctx, a.manifest.Chapters, peers, ts.lookup)
	ts.mu.Lock()
	info.SigningIn = ts.signingIn
	changed := !sameTailscale(info, ts.last)
	ts.last = info
	ts.mu.Unlock()
	if changed {
		a.emitTailscaleState(info)
	}
	return info
}

func sameTailscale(x, y models.TailscaleInfo) bool {
	return x.State == y.State && x.Version == y.Version && x.CanInstall == y.CanInstall &&
		x.SigningIn == y.SigningIn && slices.Equal(x.Connected, y.Connected)
}

// tailscaleDone is whether there is nothing left to watch for: Tailscale is
// running, no sign-in waits, and every chapter with a Tailscale address is
// reached.
func (a *App) tailscaleDone(info models.TailscaleInfo) bool {
	if info.State != services.TailscaleRunning || info.SigningIn {
		return false
	}
	for _, c := range a.manifest.Chapters {
		if _, ok := services.TailscaleAddress(c); ok && !slices.Contains(info.Connected, c.ID) {
			return false
		}
	}
	return true
}

// watchTailscale looks at the state every few seconds, one watch at a time,
// until there is nothing left to watch for (tailscaleDone) or
// tailscaleWatchFor has passed.
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
			if a.tailscaleDone(a.lookAtTailscale(ctx)) {
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
// Tailscale address. Under a Tailscale preview a switch is not changed, since
// what the row shows is not this machine's Tailscale: the save is refused and
// the preview ended, as refuseUnderPreview does for a chapter.
func (a *App) checkTailscaleSave(before models.AppSettings, settings *models.AppSettings) error {
	if !slices.Equal(before.TailscaleServers, settings.TailscaleServers) && a.underTailscalePreview() {
		a.previews.ClearLauncher()
		return errors.New("Tailscale was showing a preview, so nothing was changed. The preview is cleared: try again")
	}
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
