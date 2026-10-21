# ADR-0006: Self-update and code signing

**Status:** proposed. Roadmap, milestone 7. The in-app update is built (amendment, 2026-10-21).

## Context

Unsigned Windows builds trigger SmartScreen; unsigned macOS builds hit
Gatekeeper. A code-signing certificate costs money every year, and this is a
personal tool for a handful of people who know where it came from.

## Proposal

- **Manual updates first.** A GitHub Release per version, artefacts attested
  with `actions/attest` and a `checksums.txt`, the way Konnekt's release
  workflow does, so provenance is verifiable without a certificate.
- **No certificate** for now. The SmartScreen and Gatekeeper prompts are
  acceptable for personal use, and the release notes say how to get past
  them and how to verify the attestation.
- An in-app update check (GitHub Releases API, compare against `Version`) is
  a later step, and an in-place updater later still, if ever; Konnekt's
  `selfupdate` path is the reference when it comes.

## Amendment, 2026-10-04: installers, and signing reopened

The first beta goes to players through the download site, which is the
"Reopen when" below. Decided with the author:

- **Installers** (issue 177): an NSIS setup on Windows, for the one user, and
  a disk image on macOS. Both are attested like the files before them.
- **Windows signing through SignPath Foundation** (issue 179), free for
  open-source projects. Its terms want an OSI licence on the repository and a
  release already published in the form to be signed, so the first beta ships
  unsigned and the application follows it. The publisher will read SignPath
  Foundation. No certificate clears SmartScreen on day one; reputation builds
  with downloads.
- **No Developer ID for the beta.** Gatekeeper is only satisfied by Developer
  ID and notarization, which is the Apple Developer Program at 99 USD a year.
  Players use Privacy & Security, Open Anyway, as the release notes say.

## Reopen when

Friends beyond the author install it, or a build is distributed anywhere other
than the repository's own releases.

## Amendment, 2026-10-21: the launcher updates itself

The author asked for an update that needs no more than a restart. Decided with
the author: Update downloads in the background, the sidebar's downloads show
the bar, and once it is ready the header's update button turns to Restart. A
game that is running keeps running.

- **Our own code, no library.** minio/selfupdate replaces one file, so not a
  macOS bundle; Sparkle and WinSparkle mean an Objective-C framework or a DLL
  inside a Wails build. The job is a download, two checks and two renames.
- **A signed checksum file.** The release workflow signs `checksums.txt` with
  ed25519 (`scripts/update-sign`) and publishes `checksums.txt.sig`. The public
  key is built in (`services.UpdatePublicKeys`); the private key is the
  `release` environment's secret `UPDATE_SIGNING_KEY` alone. A build the app
  downloads itself carries no quarantine flag (the app has no
  `LSFileQuarantineEnabled`) and no Mark of the Web, so neither Gatekeeper's
  Open Anyway nor SmartScreen sees an update: the signature is the gate.
- **macOS swaps the whole bundle.** A new release asset,
  `Kapital-Launcher-<tag>-macos-universal.zip`, made with `ditto -c -k
  --keepParent`, is unpacked by `ditto` into a hidden folder beside the app,
  checked with `codesign --verify --deep --strict` for the launcher's
  identifier, and renamed into place, the running bundle renamed aside first.
  A copy macOS translocated, or one in a folder the player cannot write,
  offers the site as before. App Management (macOS 13 on) is described by
  Apple for notarized apps and same-team updates; the macOS workflow swaps a
  bundle in `/Applications` to show it does not stop ours.
- **Windows swaps the exe.** The bare exe each release already carries is
  downloaded beside the running one, which Windows lets be renamed but not
  overwritten; the new one is renamed into place and the setup's
  `DisplayVersion` updated, for the entry whose icon is this exe.
- **Restart** happens after the window has closed (`main` after `wails.Run`):
  `open -n` on macOS, the exe on Windows. It waits while a game loads or
  closes, as its window may be hidden behind the loading card, and while a
  download runs. The new copy removes the old one at its start. Nothing in
  the data folder moves: settings, the managed Prism, the sync copy (which
  follows the new version before the next Play) and Tailscale.
- **The first release with this is installed by hand**; the next is the first
  one a player gets by restarting. A build without a key offers the site only.
