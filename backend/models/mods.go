package models

// ModToggle is a mod a chapter's manifest offers as a quick switch (issue 156):
// a name to show and the stable start of the mod's jar file name. A jar's name
// carries the mod's version, so the manifest names the part before it
// ("DistantHorizons-") and every jar of the instance that starts with it is the
// mod, whichever version the pack ships now. It names a jar and never runs one:
// the launcher only matches it against the file names in the instance's mods
// folder.
type ModToggle struct {
	Name      string `json:"name"`
	JarPrefix string `json:"jarPrefix"`
	// Requires is the jarPrefix of another toggle of the chapter that this mod
	// needs to load (Colorwheel needs Iris): with that one off, this one is off
	// too and cannot be turned on. Empty for a mod that needs none of them.
	Requires string `json:"requires,omitempty"`
}

// PackAddon is a second packwiz pack a chapter's manifest offers on top of its
// own: Camera Tools, Flashback and what it needs. It is off for every player
// until they turn it on, and only then does the pre-launch sync download it into
// the chapter's instance, with a record of its own beside the pack's. It names a
// pack and never runs one: the URL is held to the pack URL's rules.
type PackAddon struct {
	// ID is the add-on's stable name, in the settings and in the file name of
	// its record ("kapital-addon-camera.json"): lower case letters, digits and
	// dashes.
	ID string `json:"id"`
	// Name is what the switch is called: "Camera Tools".
	Name string `json:"name"`
	// Description is the line under the switch: what it adds and what it costs.
	Description string `json:"description"`
	// Packwiz is the https URL of the add-on's pack.toml.
	Packwiz string `json:"packwiz"`
}

// AddonState is a manifest add-on as the settings page shows it.
type AddonState struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Enabled is the player's choice, saved in the launcher's settings.
	Enabled bool `json:"enabled"`
	// Jars are the base names of the add-on's jars in the mods folder, enabled
	// or put away; none before its first sync.
	Jars []string `json:"jars"`
}

// ModFile is one jar in an instance's mods folder as the settings page lists it.
type ModFile struct {
	// Name is the jar's base name, "Mod-1.2.3.jar", whether the file on disk is
	// that or "Mod-1.2.3.jar.disabled": it is what SetModsDisabled takes back.
	Name string `json:"name"`
	// Disabled is whether the file on disk is the ".jar.disabled" one, Prism's
	// own convention for a mod that is switched off.
	Disabled bool `json:"disabled"`
	// Size is the file's size in bytes.
	Size int64 `json:"size"`
}

// ModToggleState is a manifest toggle resolved against the mods folder.
type ModToggleState struct {
	Name      string `json:"name"`
	JarPrefix string `json:"jarPrefix"`
	Requires  string `json:"requires,omitempty"`
	// Blocked is true when a toggle this one requires, directly or through
	// another, is off: the switch cannot be turned on until that one is.
	Blocked bool `json:"blocked"`
	// Jars are the base names of the jars the toggle matches now; none before
	// the pack's first sync.
	Jars []string `json:"jars"`
	// Disabled is true when the toggle matches at least one jar and every one of
	// them is disabled.
	Disabled bool `json:"disabled"`
}

// ChapterMods is the settings page's view of a chapter's mods: what is in the
// instance's mods folder and which of it is switched off, with the manifest's
// quick toggles resolved against it.
type ChapterMods struct {
	ChapterID string `json:"chapterId"`
	// Mods is every jar, sorted by name; empty before the pack's first sync.
	Mods    []ModFile        `json:"mods"`
	Toggles []ModToggleState `json:"toggles"`
	// Addons are the manifest's add-ons with the player's choice. Their jars are
	// not in Mods: the add-on's switch is the only one they have.
	Addons []AddonState `json:"addons"`
	// Running is whether the game's tracker or its log says the instance is
	// running, as ChapterSettingsInfo.Running: a snapshot, a hint in the view.
	Running bool `json:"running"`
}
