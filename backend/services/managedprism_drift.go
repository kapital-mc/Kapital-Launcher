package services

import (
	"context"
	"log/slog"
	"sync"

	"kapital/backend/models"
)

// selfUpdate is the last self-update Reconcile looked at: the version Prism
// reported and what checking its signature gave, so a version is verified and
// logged once, not on every look.
type selfUpdate struct {
	mu      sync.Mutex
	version string
	err     error
}

// Reconcile corrects rel for a managed Prism that updated itself (#221).
// Prism's own updater (Sparkle on macOS, which stays on: its settings are per
// user and shared with any other Prism) can replace the bundle in place, after
// which managed.json names the version the launcher installed, not the one on
// disk. detected is what detection read from the managed executable. When it
// is a different Prism version, rel says that one is installed, an update is
// offered only for a newer release, and the bundle is checked again as an
// install checks it, since the launcher did not place what now runs. A copy
// that no longer verifies is offered the latest release as a repair, which
// installs into its own folder as any update does. The first look at each
// version is logged; managed.json is left as the install wrote it, as the
// program folder's name.
func (m *ManagedPrism) Reconcile(ctx context.Context, rel *models.PrismRelease, detected string) {
	if rel.Installed == "" || detected == rel.Installed || !prismTag.MatchString(detected) {
		return
	}
	recorded := rel.Installed
	m.drift.mu.Lock()
	defer m.drift.mu.Unlock()
	if m.drift.version != detected {
		m.drift.version = detected
		m.drift.err = m.verify(ctx, m.appDir(recorded))
		slog.Info("managed prism updated outside the launcher", "recorded", recorded, "running", detected, "verified", m.drift.err == nil)
		if m.drift.err != nil {
			slog.Warn("managed prism no longer verifies", "version", detected, "error", m.drift.err)
		}
	}
	rel.Installed = detected
	rel.UpdateAvailable = newerVersion(rel.Version, detected) || m.drift.err != nil
}
