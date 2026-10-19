# ADR-0014: Tailscale as a chapter's private address

**Status:** accepted, 2026-10-18. Built.

## Context

Frangfurd's server is now also reachable through Tailscale, at
`ts.frangfurd.click`: an A record in Cloudflare DNS for the server's Tailscale
address, 100.64.209.106, set to DNS only (Cloudflare's proxy cannot carry
Minecraft, and an address in 100.64.0.0/10 cannot be proxied anyway). The name
resolves for everyone but answers only on a machine that is in the server's
tailnet. The author asked (2026-10-18) for the launcher to get and set up
Tailscale with no Tailscale window, for a switch per chapter that adds the
Tailscale address under Address and to the in-game server list, and for
Get Prism to move into a popup with a downloads section in the sidebar.

What Tailscale allows, read in its documentation on 2026-10-18:

- **Funnel cannot carry Minecraft.** It serves TLS on ports 443, 8443 and
  10000 under a `ts.net` name only, so every player needs Tailscale itself.
- **No hidden install on macOS.** The Standalone app installs from a signed
  `.pkg`, then asks the player to allow a system extension and a VPN
  configuration. Those are macOS's own prompts and no app can answer them.
- **The macOS app is its own CLI.** The app's executable acts as `tailscale`
  when `TAILSCALE_BE_CLI=1` is set.
- **The CLI reports and signs in.** `tailscale status --json` has a
  `BackendState` (`Running`, `NeedsLogin`, `NeedsMachineAuth`, `Starting`,
  `Stopped`, `NoState`). `tailscale login` prints a `login.tailscale.com`
  address and waits for the browser.
- **The package server is public.** `pkgs.tailscale.com/stable/?mode=json`
  names the current MSIs and macOS packages, with a `.sha256` beside each.
- **Who may join is the tailnet owner's choice.** A player is invited to the
  tailnet, or the server is shared with them; sharing does not work for a
  tagged machine. The launcher has no part in that.

Embedding Tailscale in the launcher (`tsnet`) was ruled out: the game would
then have to join a local relay the launcher runs, and the in-game server list
would name `localhost` rather than the server.

## Decision

- **A manifest address may be a Tailscale one.** `ServerAddress.tailscale`,
  at most one per server and never the first, so the default stays public
  (ADR-4 amendment, 2026-10-18).
- **A switch per chapter, off by default.** `AppSettings.tailscaleServers`
  holds the chapters where the player turned it on, between Address and Join on
  the settings page. Off, the Tailscale choice under Address is shown disabled,
  a saved Tailscale choice is dropped, and Play falls back to the first
  address. On, it is a choice like the others.
- **The in-game server list follows the switch.** Go edits the instance's
  `servers.dat` (uncompressed NBT) on a switch and again before every Play:
  one entry named `<Chapter> (Tailscale)` is added, or every entry with the
  Tailscale address is removed. The player's other entries are kept byte for
  byte. A missing list is seeded from the pack's
  `config/defaultoptions/servers.dat` so the first run still has the pack's
  servers. A failure is logged and refuses nothing.
- **Getting Tailscale is the official installer, on approval.** Get Tailscale,
  or the switch turned on without Tailscale, opens the same popup as Get Prism
  with the version and size. On Download, Go reads the package index again,
  takes this OS's package (the universal `.pkg`, or the `.msi` of this
  architecture), holds its name to a plain file name carrying the version,
  downloads it from `pkgs.tailscale.com` only (no redirect off
  that host), checks its size and the published SHA-256, and on macOS also
  that `pkgutil --check-signature` names Tailscale's Developer ID Installer,
  `Tailscale Inc. (W5364U7YZB)`. Then it runs it: `msiexec /i <file> /passive
  /norestart TS_NOLAUNCH=1` on Windows, `open -W <file>` on macOS, so the
  player sees the OS's own installer and its password or administrator prompt.
- **Signing in is the browser.** The launcher runs `tailscale login` and opens
  the first address it prints only when it is https on `login.tailscale.com`
  with no user info; the address is never logged. Open Tailscale starts the
  app when it is installed and not running.
- **State arrives as events.** `tailscale:state` carries a
  `models.TailscaleInfo` (state, version, whether this OS can install it, a
  sign-in waiting); `tailscale:install` a step of the install. Go looks again
  every 3 s for 15 minutes after an install, a sign-in or Open Tailscale, and
  emits only on a change. Nothing in the frontend polls.
- **Downloads have one place.** A section above the account tile lists every
  running download (Prism, Tailscale, a chapter's terrain) with its bar, so a
  popup or a settings card can be closed while it runs.

## Consequences

- Tailscale's windows remain: the OS installer, and on macOS the extension and
  VPN prompts. The popup says what to expect rather than hiding them.
- A player still needs access to the tailnet, which is the server owner's to
  give.
- Windows has no signer check beyond the digest from the same host; Windows
  Installer shows the MSI's verified publisher in its administrator prompt.
  Pinning Tailscale's Authenticode signer is in the security backlog.
- The launcher writes a second file into an instance, `servers.dat`, for this
  one entry (ADR-2's rule on what is written into Prism's data).
- To check on a real machine: `/passive` with `TS_NOLAUNCH`, the Windows CLI
  and tray app paths, the macOS CLI through `TAILSCALE_BE_CLI`, and catching
  the sign-in address. Each is marked `[verify]` in the code.

## Amendment, 2026-10-19: Install, Connect, then the switch

The author asked (2026-10-19) for the row to be the next step rather than a
switch with a line under it: **Install** while Tailscale is missing,
**Connect** while it does not reach the server, and the switch only once it
does. The hover line on the Tailscale choice under Address is gone.

- **Reaching the server is seen, not assumed.** `TailscaleInfo.connected`
  lists the chapters whose Tailscale address is one of this machine's peers:
  Go resolves the address's host and compares it with each peer's
  `TailscaleIPs` in `tailscale status --json`. A machine shared with the
  player shows up as a peer once they accept the link. Tailscale's sharing
  page (read 2026-10-19) says a shared machine "will typically" keep its
  address in the recipient's tailnet; when it does not, `ts.frangfurd.click`
  does not reach it either, and the row rightly stays on Connect.
- **Connect is one button for every step between.** `ConnectTailscale` opens
  Tailscale when it is not running, signs in when it needs to and then opens
  the share link, and opens the link when signed in. The link is the
  manifest's `invite` (ADR-4 amendment 2026-10-19). Go watches until every
  chapter with a Tailscale address is reached.
- **The share link is public.** It ships in the launcher, so it has to be a
  reusable link (Tailscale allows up to 1,000 uses) and anyone with the app
  can use it. Sharing quarantines the machine (it takes connections and
  starts none), and the author's tailnet policy can limit
  `autogroup:shared` to the Minecraft port; that is the author's to set.
  Changing the link needs a launcher release until the wiki publishes the
  manifest (ADR-4).
- **Previews.** The Developer section groups its previews as Launcher, Prism
  and Instance, as links. Launcher holds the launcher's update and four
  Tailscale situations (not installed, the install failing, not connected,
  connected), which open the first Tailscale chapter's settings page.

## Amendment, 2026-10-19: what the popup says, and Cancel

The author asked (2026-10-19) for the popup's second paragraph, which named
what macOS and Windows ask while installing, to say instead what Tailscale
does for the connection: it usually reaches the server directly, with a
lower ping than playit, and keeps working when playit has trouble. The OS's
own prompts are no longer announced in the popup; the installer step still
says the installer is open.

Later the same day the author asked for both paragraphs to be shorter: one
paragraph now says Tailscale connects over a private network, usually
direct, with a lower ping, and independent of playit.
Where the installer comes from is left to the facts line (`pkgs.tailscale.com`);
the check before it runs is unchanged.

Tailscale's download is cancelled from its row in the downloads section like
Prism's (ADR-11 amendment of the same day, `CancelTailscaleInstall`): only
while the installer downloads, never once it is being checked or has been
started. The Launcher previews gain the popup on its offer, and the install
preview now plays inside the popup.
