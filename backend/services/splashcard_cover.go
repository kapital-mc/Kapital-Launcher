package services

import (
	"errors"
	"log/slog"
	"sync"

	"kapital/backend/splashhost"
)

// The macOS stand-in for the game window hold (#258, ADR-0012's macOS
// amendment). macOS lets no app hide or move another's window without the
// Accessibility permission, so instead of the game's window the launcher moves
// its own: from the moment the tracker finds the game's Java, the card is
// raised above the game's windows over a cover on every screen
// (splashhost.Host.Cover), and at the handover the cover goes, the game's
// process is activated by its pid and the card closes. To the tracker it is a
// WindowHolder like Windows' hold, so the handover, a run that ends first and
// the logging are the same code.

// errNoCardToCover is a game found with no card up for its chapter: the player
// left it, or it ended. The game then shows as it is made.
var errNoCardToCover = errors.New("no loading card is up to cover the game")

// GameHold is the hold the tracker uses for a run with this card, in place of
// its own (TrackRequest.Hold), or nil to keep the tracker's: Windows holds the
// game's window itself. On macOS it raises the card over its cover.
func (c *SplashCard) GameHold(chapterID string) func(pid int) (WindowHolder, error) {
	if c.cfg.GOOS != "darwin" {
		return nil
	}
	return func(pid int) (WindowHolder, error) { return c.cover(chapterID, pid) }
}

// cover raises the chapter's card, if it is up and has not ended, and returns
// the holder that lowers it again. The host only dispatches, but it is called
// with mu released all the same, as every host call is.
func (c *SplashCard) cover(chapterID string, pid int) (WindowHolder, error) {
	c.mu.Lock()
	run := c.run
	if run == nil || run.chapter.ID != chapterID || !c.live(run) || run.ended {
		c.mu.Unlock()
		return nil, errNoCardToCover
	}
	host := run.host
	c.mu.Unlock()
	host.Cover(true)
	slog.Info("loading card covers the game", "chapter", chapterID)
	return &cardCover{host: host, pid: pid, activate: c.cfg.Activate}, nil
}

// cardCover is one run's cover, as the tracker's WindowHolder.
type cardCover struct {
	host     splashhost.Host
	pid      int
	activate func(pid int) bool

	once sync.Once
	rep  WindowReport
}

// Release lowers the card and takes the cover away. With foreground it is the
// handover, and the game's process is asked to come to the front; the card is
// closed after it, by Handover. Without, the run ended first and the card stays
// at the normal level with its error, where Prism's console and the launcher
// can come in front of it. A card the player left, or closed, ignores the
// lowering. Safe twice: the second call returns the first one's report.
func (h *cardCover) Release(foreground bool) WindowReport {
	h.once.Do(func() {
		h.host.Cover(false)
		// Nothing of the game's window was hidden, so there are no hide times.
		h.rep = WindowReport{FirstHideMs: -1, MaxHideMs: -1}
		if foreground {
			h.rep.Foreground = h.activate(h.pid)
		}
	})
	return h.rep
}
