package models

// The places a preview applies to: one chapter's screens, Prism's, or the
// launcher's own (its update, the player's Tailscale).
const (
	PreviewScopeChapter  = "chapter"
	PreviewScopePrism    = "prism"
	PreviewScopeLauncher = "launcher"
)

// PreviewSituation is one situation the Developer section can fake (#124): a
// screen the player only sees when something goes wrong. The list is fixed in
// Go (services.PreviewSituations), and the frontend renders it as it comes.
type PreviewSituation struct {
	// ID is the name StartPreview takes.
	ID string `json:"id"`
	// Label is the button's text.
	Label string `json:"label"`
	// Scope is PreviewScopeChapter, PreviewScopePrism or PreviewScopeLauncher.
	Scope string `json:"scope"`
	// Card is whether the situation opens the loading card, which needs the
	// loading splash to be on.
	Card bool `json:"card"`
	// PlaysInstall is whether the view then runs the install of its scope
	// (Prism's, or the launcher's one: Tailscale), whose progress and failure
	// are the situation.
	PlaysInstall bool `json:"playsInstall"`
	// Popup is whether the view opens its scope's install popup (getting
	// Prism, or Tailscale), on its offer or over the install it plays.
	Popup bool `json:"popup,omitempty"`
	// Page is the chapter page the screen is on, "" for the main screen: the
	// Tailscale situations are "settings", the settings page of the first
	// chapter whose server has a Tailscale address, where its row is.
	Page string `json:"page,omitempty"`
}

// PreviewStart is what StartPreview says happened.
type PreviewStart struct {
	// CardSkipped is whether the situation wanted the loading card and it was
	// not opened: the loading splash is off, or the card could not open.
	CardSkipped bool `json:"cardSkipped"`
}
