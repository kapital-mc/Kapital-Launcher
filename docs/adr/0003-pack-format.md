# ADR-0003: Pack format and distribution

**Status:** accepted in principle, 2026-09-28; nothing built yet (Roadmap,
milestone 4).

## Context

The handoff (§3, ADR-3) proposed packwiz as the source of truth, with
`packwiz-installer` in Prism's pre-launch command for day-to-day sync (option B)
and a `.mrpack` import for a fresh install (option A). Verified 2026-09-28:
packwiz is MIT, exports Modrinth and CurseForge packs, and its installer is
built for auto-updating MultiMC-family instances. Prism's custom commands
substitute `$INST_JAVA`, `$INST_DIR`, `$INST_MC_DIR`, `$INST_ID` and
`$INST_NAME`, and warn that `$INST_JAVA_ARGS` breaks on arguments with spaces.

## Decision

As proposed, with two things stated more precisely than the handoff did:

- **The pre-launch command is a template in Go** (`preLaunchCommand` in
  `backend/services/packinstance.go`):
  `"$INST_JAVA" -jar "$INST_MC_DIR/packwiz-installer-bootstrap.jar" --bootstrap-no-update --bootstrap-main-jar "$INST_MC_DIR/packwiz-installer.jar" <pack.toml url>`.
  The manifest contributes only the URL, already validated as https on an
  allowlisted host and free of anything Prism's command line reads (`$`,
  quotes, spaces). No manifest field can ever carry a command.
- **Hashes are required.** `.mrpack` files carry per-file hashes and packwiz's
  `index.toml` does; a file whose hash is missing or does not match is
  refused, not skipped.
- Mods are referenced by their Modrinth or CurseForge download URL and never
  re-hosted. Prefer Modrinth where a mod is on both, and expect some
  CurseForge mods to opt out of third-party distribution. The one exception
  is the project's own jars (amendment below).
- Hosting is a static host that serves `pack.toml`, `index.toml`, the mods'
  metadata and every file the pack carries itself (configs, scripts, its own
  resource pack); GitHub Releases or Cloudflare Pages. For Frangfurd that is
  426 files besides the metadata, about 107 MB, all of it public once hosted
  (#35). The manifest schema already has `pack.packwiz` and `pack.mrpack`
  slots.

## Authoring: a repository, not an admin panel

The question came up whether packs should be managed from an admin panel
that adds and removes mods and tests locally, or fetched by the launcher
from a hosted `.mrpack` and manifest. The second, and the reasons are what
packwiz already is:

- **The pack source is a git repository** of packwiz TOML, one folder per
  chapter. Adding a mod is `packwiz modrinth add <slug>` (or `curseforge
  add`), removing is `packwiz remove`, and `packwiz refresh` rewrites
  `index.toml` with hashes. Every change is a commit with a diff, which is
  the audit trail an admin panel would have to build.
- **Local testing is `packwiz serve`**, packwiz's built-in HTTP server for
  exactly this: point an instance's pre-launch installer at
  `http://localhost:8080/pack.toml` and launch. The launcher's developer
  setting for that is `packOverrides` in `settings.json` (#41): Install then
  writes the instance against the local address, loopback only. The instance
  keeps syncing from it, and the state line says "Dev pack", until the player
  switches it back to the published pack in the chapter's settings (ADR-2,
  seventh amendment), which rewrites that one key.
- **Publishing is a push.** A static host serves `pack.toml` and
  `index.toml`; `packwiz modrinth export` produces the `.mrpack` a fresh
  install imports. The launcher's manifest carries the two URLs and nothing
  else changes.

An admin panel would be a second application with its own hosting, auth and
database, to do what the CLI and a git remote do for one person and a few
friends. It is the wrong size. If the pack authors ever stop being people
who use git, the panel is a front-end over the same repository, not a
replacement for it.

## Open

- How Prism's import handles an instance whose name already exists. Answered
  from Prism 11.1.1's source on 2026-09-30 (the comment on #22 cites the
  lines): `-I` always opens the New Instance dialog and waits for OK, the
  folder name comes from that dialog's name field, and a collision gets a
  `(1)` suffix, never an overwrite. So an import cannot promise the
  `kapital-<id>` folder `--launch` needs. Decided in #22: the launcher writes
  that folder itself (ADR-2, amendment).
- Whether `packwiz-installer-bootstrap.jar` is fetched by the launcher (then
  hashed) or by the pack import. Decided in #22: the launcher fetches both
  jars from their GitHub releases, pinned by size and SHA-256
  (`backend/services/packwizjars.go`), and copies them into the instance. The
  bootstrap runs with `--bootstrap-no-update`, because it otherwise asks
  `api.github.com` for a newer installer on every Play.

## Fresh install, as built (#22)

A fresh install no longer imports a `.mrpack` (option A). The launcher writes
the instance: `instance.cfg` with the pre-launch command, the chapter's JVM
preset (`pack.jvm`, a name the launcher maps to arguments) and memory
(`pack.memoryGb`, as `MaxMemAlloc`, with `MinMemAlloc` at 512 MB, Prism's own
default, so the heap starts small on any machine and grows as needed); `mmc-pack.json` with Minecraft and the loader read from the hosted
`pack.toml`, refused when they disagree with the manifest; and the two jars.
The first Play then installs the whole pack through the pre-launch command,
and every later Play syncs it. Checked in a container on 2026-09-30 by
parsing the generated `instance.cfg` with Qt's own `QSettings`, splitting the
command with `QProcess::splitCommand`, and running it against `packwiz serve`:
529 of 529 files, also from a path with a space in it.

## Amendment, 2026-10-12: the project's own jars (#207)

The mods from `sandrogekeler/frangfurd-mod` are on neither Modrinth nor
CurseForge, and will not be. Under "never re-hosted" they could never be part
of a pack, so they were copied by hand into the server and each client.

**Decision:** a pack may carry a jar this project builds and owns as a plain
file, listed in `index.toml` with its hash and served by the pack host like a
config file.

- **What it covers.** Jars whose name starts with a prefix on the importer's
  allowlist in `kapital-packs` (`FIRST_PARTY` in `tools/import_instance.py`,
  today `frangfurd_`). That repository's checks refuse any other stored jar.
  The jars come from a tagged frangfurd-mod release with a `checksums.txt`.
- **What it does not cover.** Any third-party mod, whatever its licence and
  however inconvenient its download. Those stay references, and a CurseForge
  mod with third-party downloads turned off is still swapped or removed.
- **Why it is allowed.** Re-hosting is refused because those files are someone
  else's to distribute. That reason does not apply to the project's own jars.

**Consequences:**

- The jars are public once served, although frangfurd-mod is a private
  repository: anyone with the `pack.toml` URL can download them (#35).
- Nothing changes in the sync. packwiz-installer already fetches plain index
  files from the pack host and checks their hash. A first-party jar's name is
  one `modJarName` accepts, so it is handled like any other mod.
- `AllowedManifestHosts` is unchanged: the jars come from the pack host the
  manifest already names.
- Every mod in frangfurd-mod is optional on the client (its ADR 0001), so
  Frangfurd's client mods, Frangfurd Menu and Frangfurd Tempo, are switches in
  `pack.toggles` (`frangfurd_menu-`, `frangfurd_tempo-`). Frangfurd Roles is a
  server mod and is not in the client pack.

## Amendment, 2026-10-13: a Prism instance zip for Prism alone (#251)

A player who uses Prism and not the launcher had no way in. The `.mrpack` that
`packwiz modrinth export` makes is not one: it is a snapshot nothing updates,
and it re-hosts, since packwiz puts every file whose download is not on
Modrinth's host list (`cdn.modrinth.com`, `github.com`,
`raw.githubusercontent.com`, `gitlab.com`) into the zip's overrides. For
Frangfurd that is a CurseForge jar. Read in packwiz's `modrinth/export.go` on
2026-10-13.

**Decision:** every release attaches one instance zip per chapter with a
hosted pack, `Kapitel-Kapital-<Name>.zip`, which Prism imports with Add
Instance, Import. It is packwiz's own documented MultiMC setup ("Installing
with packwiz-installer", which tells a pack author to export the instance and
host the zip), built by the same Go that writes a fresh instance:

- `services.PrismZip` takes `instanceFiles` (mmc-pack.json from the hosted
  `pack.toml`, refused when it disagrees with the manifest, and the two pinned
  jars) and adds `instance.cfg` and the jars' MIT licence. No mod is inside,
  so nothing is re-hosted.
- The pre-launch command is the first template, with packwiz-installer's
  window. Headless (`-g`) is for the launcher, whose splash covers the sync
  (#95); in Prism alone the window is the only progress a player sees.
- `scripts/prism-instances` builds them in the release workflow, and the
  download site's popup links them when the release it shows has all of them.

**What makes it hold up without a rebuild per pack release:** packwiz-installer
rewrites `mmc-pack.json` to the pack's Minecraft and loader versions before it
syncs (`LauncherUtils.handleMultiMC` in v0.5.14; seen on 2026-10-13 moving a
stale Fabric loader to the hosted Lichdenstein's 0.19.5), so a zip from an
older release still installs the current pack. Only a change to the command,
the JVM presets, the memory or the jar pins needs a new zip, and every release
makes one anyway.

**Consequences:**

- Prism names the instance from its import dialog, which suggests the zip's
  name, so the folder is not `kapital-<id>`, and the zip's name is chosen not to
  be (a case-insensitive disk would match it). The launcher does not count such
  an instance as installed; a player who later installs the launcher installs
  the chapter again.
- The zips are only as reachable as the release: they are attested and listed
  in `checksums.txt` with the installers.
- Checked on 2026-10-13 in a container with no Prism: the Lichdenstein zip's
  command, run from its `minecraft/` folder under a path with a space,
  installed the hosted pack (37 mods, 120 files). The import itself, by Prism's
  `InstanceImportTask` (a zip with an `instance.cfg` is a MultiMC instance,
  extracted from that file's folder, renamed from the dialog), is read from
  source and still to be seen on each OS.
