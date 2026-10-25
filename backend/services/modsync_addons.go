package services

import (
	"slices"

	"kapital/backend/models"
)

// syncAddon is one of the chapter's add-ons in a sync run (ADR-2, twenty-second
// amendment): the manifest's entry, the player's choice, and the jars its record
// says it put in the mods folder, if it was ever synced.
type syncAddon struct {
	addon   models.PackAddon
	enabled bool
	jars    []string
}

// addonsFor resolves the chapter's add-ons for this run. Only ids the manifest
// lists count: a choice for an add-on the manifest dropped is nothing.
func (s *syncRun) addonsFor(env syncEnv, addons []models.PackAddon, enabled []string) []syncAddon {
	runs := make([]syncAddon, 0, len(addons))
	for _, a := range addons {
		jars, err := AddonJars(env.mcDir, a.ID)
		if err != nil {
			s.say("the record of add-on %s cannot be read (%s)", q(a.ID), errKind(err))
		}
		runs = append(runs, syncAddon{addon: a, enabled: slices.Contains(enabled, a.ID), jars: jars})
	}
	return runs
}

// prepareAddons puts the jars of every add-on that is on back to ".jar" before
// any installer runs, so its own run finds nothing missing and downloads nothing
// that did not change. The player's disabled list is put away again afterwards
// by putAway, as for any mod.
func (s *syncRun) prepareAddons(env syncEnv, addons []syncAddon) {
	for _, a := range addons {
		if !a.enabled || len(a.jars) == 0 {
			continue
		}
		if _, err := SetAddonJars(env.mcDir, a.jars, true, nil); err != nil {
			s.say("add-on %s: not every jar could be put back (%s)", q(a.addon.ID), errKind(err))
		}
	}
}

// installAddons syncs each add-on that is on, after the chapter's own pack and
// only when that one succeeded: a pack that could not sync leaves no ground to
// add to. It returns the code the sync exits with, the chapter's own first, then
// the first add-on's that failed.
func (s *syncRun) installAddons(env syncEnv, addons []syncAddon, code int) int {
	for _, a := range addons {
		if !a.enabled {
			continue
		}
		if code != 0 {
			s.say("add-on %s waits for the pack sync to succeed", q(a.addon.ID))
			continue
		}
		s.say("running the sync of add-on %s", q(a.addon.ID))
		args := packwizAddonSyncArgs(env.mcDir, a.addon.Packwiz, addonRecordName(a.addon.ID))
		c, err := s.run(env.java, args, env.mcDir, s.d.Stdout, s.d.Stderr)
		switch {
		case err != nil:
			s.say("the sync of add-on %s could not be started (%s)", q(a.addon.ID), errKind(err))
			code = syncExitNoInstaller
		case c != 0:
			s.say("the sync of add-on %s exited with code %d", q(a.addon.ID), c)
			code = c
		}
	}
	return code
}

// putAwayAddons renames the jars of every add-on that is off to
// ".jar.disabled", whatever happened before: off is the state an add-on must be
// in unless the player asked for it.
func (s *syncRun) putAwayAddons(env syncEnv, addons []syncAddon) {
	for _, a := range addons {
		if a.enabled || len(a.jars) == 0 {
			continue
		}
		n, err := SetAddonJars(env.mcDir, a.jars, false, nil)
		if err != nil {
			s.say("add-on %s: not every jar could be put away (%s)", q(a.addon.ID), errKind(err))
		}
		if n > 0 {
			s.say("add-on %s is off, %d jars put away", q(a.addon.ID), n)
		}
	}
}
