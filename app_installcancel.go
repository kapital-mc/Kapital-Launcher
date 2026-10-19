package main

import (
	"context"
	"errors"
	"sync"
)

// installCancel is Cancel on a Prism or Tailscale install, from the sidebar's
// downloads section: it ends the install only while its download runs. Once
// the download is whole the steps after it (unpacking, the signature check,
// Tailscale's installer) run to their end, so a cancel never leaves an install
// half done. The zero value is ready to use.
type installCancel struct {
	mu sync.Mutex
	// cancel ends the running install's context; nil when none runs.
	cancel context.CancelFunc
	// downloading is whether the running install's latest step is its download.
	downloading bool
	// cancelled is whether the player cancelled the running install.
	cancelled bool
}

// errInstallCancelled is what an install the player cancelled answers.
var errInstallCancelled = errors.New("the download was cancelled")

// begin starts tracking an install and returns its context and the end to call
// when it returns, which says whether the player cancelled it. A second
// install while one runs is not tracked: the installer refuses it anyway.
func (c *installCancel) begin(parent context.Context) (context.Context, func() bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return parent, func() bool { return false }
	}
	ctx, cancel := context.WithCancel(parent)
	c.cancel, c.downloading, c.cancelled = cancel, false, false
	return ctx, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		cancel()
		c.cancel = nil
		return c.cancelled
	}
}

// step records the install's latest step and says whether to pass it on: the
// failure a cancel causes is not news, as the cancel step follows it.
func (c *installCancel) step(phase string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.downloading = phase == "downloading"
	return !c.cancelled || phase != "failed"
}

// stop cancels the running install if it is still downloading, reporting
// whether it did.
func (c *installCancel) stop() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel == nil || !c.downloading {
		return false
	}
	c.cancelled = true
	c.cancel()
	return true
}
