package services

import (
	"fmt"
	"regexp"
	"slices"

	"kapital/backend/models"
)

// tailscaleInvite is the one shape a Tailscale share link may have: the whole
// string, so nothing can come before or after it.
var tailscaleInvite = regexp.MustCompile(`^https://login\.tailscale\.com/admin/invite/[A-Za-z0-9]{8,64}$`)

// validateTailscaleAddress holds a server's Tailscale address to its rules: at
// most one, and never the first. The first is the default every player gets,
// and a player without Tailscale cannot reach a tailnet address. A share link
// belongs to the Tailscale address alone and has Tailscale's own shape.
func validateTailscaleAddress(where string, s *models.Server) error {
	n := 0
	for i, a := range s.Addresses {
		if a.Invite != "" && (!a.Tailscale || !tailscaleInvite.MatchString(a.Invite)) {
			return fmt.Errorf("%s: server: %q: an invite is a Tailscale share link on the Tailscale address", where, a.Label)
		}
		if !a.Tailscale {
			continue
		}
		if i == 0 {
			return fmt.Errorf("%s: server: the first address is the default and cannot be the Tailscale one", where)
		}
		n++
	}
	if n > 1 {
		return fmt.Errorf("%s: server: more than one Tailscale address", where)
	}
	return nil
}

// TailscaleAddress is the chapter's Tailscale address, when its server has one.
func TailscaleAddress(chapter models.Chapter) (models.ServerAddress, bool) {
	if chapter.Server == nil {
		return models.ServerAddress{}, false
	}
	for _, a := range chapter.Server.Addresses {
		if a.Tailscale {
			return a, true
		}
	}
	return models.ServerAddress{}, false
}

// UsesTailscale reports whether the player turned the chapter's Tailscale
// switch on. Off for every chapter until they do.
func UsesTailscale(settings models.AppSettings, chapterID string) bool {
	return slices.Contains(settings.TailscaleServers, chapterID)
}

// ValidateTailscaleServers refuses a save whose Tailscale switches name a
// chapter the manifest does not have, or one whose server has no Tailscale
// address. What is saved is a chapter id, never an address.
func ValidateTailscaleServers(chapters []models.Chapter, ids []string) error {
	for _, id := range ids {
		found := false
		for _, c := range chapters {
			_, has := TailscaleAddress(c)
			found = found || (c.ID == id && has)
		}
		if !found {
			return fmt.Errorf("settings: Tailscale switch for %q: no such chapter with a Tailscale address", id)
		}
	}
	return nil
}

// PruneTailscaleServers drops the chapters the manifest no longer lists with a
// Tailscale address, so a stale id on file is never written back (and so
// refused) by a save of something else.
func PruneTailscaleServers(chapters []models.Chapter, ids []string) []string {
	var kept []string
	for _, id := range ids {
		if ValidateTailscaleServers(chapters, []string{id}) == nil {
			kept = append(kept, id)
		}
	}
	return kept
}

// DropTailscaleChoices removes a saved choice of the Tailscale address for a
// chapter whose switch is off: turning the switch off moves the chapter back
// to its default address, and turning it on again offers the choice anew.
func DropTailscaleChoices(chapters []models.Chapter, settings models.AppSettings) map[string]string {
	var kept map[string]string
	for id, label := range settings.ServerChoices {
		if !UsesTailscale(settings, id) && isTailscaleLabel(chapters, id, label) {
			continue
		}
		if kept == nil {
			kept = map[string]string{}
		}
		kept[id] = label
	}
	return kept
}

func isTailscaleLabel(chapters []models.Chapter, id, label string) bool {
	for _, c := range chapters {
		if a, ok := TailscaleAddress(c); ok && c.ID == id && a.Label == label {
			return true
		}
	}
	return false
}
