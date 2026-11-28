# kapital.click: one domain for everything Kapital

Bought 2026-11-28 at Namecheap. Today it still uses Namecheap's nameservers, so
nothing below works until step 0 is done.

## Naming convention

Two kinds of name, so a player never has to guess which one to type:

| Shape | What it is | Example |
|---|---|---|
| `<chapter>.kapital.click` | What a player types into Minecraft | `frangfurd.kapital.click` |
| `<chapter>-<role>.kapital.click` | Everything else a chapter has | `frangfurd-dh.kapital.click` |
| `<site>.kapital.click` | Things shared by every chapter | `wiki.kapital.click` |

Roles, kept to a short fixed list:

| Role | Meaning |
|---|---|
| `dh` | Distant Horizons terrain snapshot (R2) |
| `ts` | Tailscale address of the chapter's server (DNS only) |
| `dl` | Downloads: worlds and server packages (R2), one bucket per chapter |
| `map` | Web map, later |

Change from the first draft: `lichdenstein-s`, `lichdenstein-w` and `luxemburg-w`
become `lichdenstein-dl` and `luxemburg-dl`, with the file named in the path:

```text
https://luxemburg-dl.kapital.click/world-1.5.0.zip
https://lichdenstein-dl.kapital.click/world-2026-10-22.zip
https://lichdenstein-dl.kapital.click/server-1.0.0.zip
```

This is one bucket and one allowlisted host per chapter instead of one per file,
and adding a Frangfurd world later needs no new DNS. `-s` would also read like
the server's join address, which `lichdenstein.kapital.click` is reserved for.

## Who does what

**You** (Cloudflare and Namecheap dashboards, accounts and keys):

| Phase | Step |
|---|---|
| 0 | Add `kapital.click` to Cloudflare, copy its two nameservers |
| 0 | Enable DNSSEC, paste the DS record into Namecheap, check auto-renew |
| 1 | Connect `frangfurd-dh.kapital.click` to the bucket `frangfurddistanthorizons` |
| 1 | Add the A record `frangfurd-ts` (DNS only) |
| 1 | Change Konnekt's snapshot URL, if it has one |
| 2 | Connect `luxemburg-dl` and `lichdenstein-dl` custom domains to their buckets |
| 2 | Create an R2 API token and set up rclone with it (install: `winget install Rclone.Rclone`) |
| 2 | Put the Groq key into the server after a restore (it never goes into the zip) |
| 3 | Add the Pages custom domains `launcher.` and `wiki.`, the Worker domain `packs.` |
| 4 | Redirect rule, Cloudflare Tunnel, Email Routing, and a join-address route if you want one |
| all | Review and merge the pull requests |

**Claude** (code, files on this machine, read-only checks):

| Phase | Step |
|---|---|
| 0 | Point Namecheap at Cloudflare's nameservers, once you send them and say go |
| 0 | Check the zone is live (`nslookup`) |
| 1 | Launcher PR: new hosts on the allowlist, manifest URLs, `servers.dat` cleanup of `ts.frangfurd.click`, ADR-14 amendment |
| 1 | Check `frangfurd-dh` and `frangfurd-ts` answer |
| 2 | Create the buckets `luxemburg-dl` and `lichdenstein-dl`, on your go |
| 2 | Build the world zips and the server package (secrets and player lists left out), write their SHA-256 |
| 2 | Run the rclone upload once you have set it up |
| 2 | Launcher PR: world download; site PR: server download and the Konnekt popup; ADR |
| 3 | PRs for the new launcher, wiki and packs addresses (Kapital-Launcher, wiki repo) |
| 4 | DNS and launcher changes for a join address, once a route is chosen |
| all | File the issues, check each address after you switch it |

## Step 0: move DNS to Cloudflare (once)

1. Cloudflare dashboard: Add a domain, `kapital.click`, Free plan. Skip importing
   records, there are none yet.
2. Copy the two nameservers Cloudflare shows.
3. Namecheap: Domain List, `kapital.click`, Manage, Advanced DNS, DNSSEC: turn it
   off first, or the domain stops resolving during the move.
4. Namecheap: Domain, Nameservers: Custom DNS, paste both.
5. Wait for Cloudflare to mark the zone Active (minutes to a few hours).
6. Cloudflare: DNS, Settings, Enable DNSSEC. It shows a DS record: Key Tag,
   Algorithm `13`, Digest Type `2`, Digest. Namecheap: Advanced DNS, DNSSEC, Add
   new DS record with those four values.
7. Namecheap: check that auto-renew is on for `kapital.click`.

R2 custom domains, Pages custom domains and Worker custom domains all need the
zone on Cloudflare, and all are free.

## Phase 1: Frangfurd (easy)

Both records are live (2026-11-28); the launcher change is issue 385.

### frangfurd-dh.kapital.click

Cloudflare:

1. R2, bucket `frangfurddistanthorizons`, Settings, Custom Domains, Add:
   `frangfurd-dh.kapital.click`.
2. Leave `dh.frangfurd.click` connected. A bucket can have several custom
   domains, and launchers already installed still use the old one.
3. Check: `https://frangfurd-dh.kapital.click/dh/latest.json` answers.

Repos:

- Kapital-Launcher: add `frangfurd-dh.kapital.click` to `AllowedManifestHosts`
  (`backend/services/manifest.go`), keep `dh.frangfurd.click` beside it, and point
  `dhSnapshot` in `data/launcher.json` at the new host. The site's status page
  reads the manifest, so it follows by itself.
- Konnekt: if its publish settings hold a public URL for the snapshot, change it.

### frangfurd-ts.kapital.click

Cloudflare: DNS, Add record:

| Type | Name | Content | Proxy |
|---|---|---|---|
| A | `frangfurd-ts` | `100.64.209.106` | DNS only (grey cloud) |

Check: `nslookup frangfurd-ts.kapital.click` returns `100.64.209.106`.

Kapital-Launcher, in one release:

- The `servers.dat` sync (ADR-14) matches the Tailscale entry by address. It has
  to also remove entries with the former address `ts.frangfurd.click`, or players
  keep a stale entry next to the new one.
- Then switch the Tailscale address in `data/launcher.json` and amend ADR-14.

## Phase 2: world and server downloads

1. R2: create buckets `luxemburg-dl` and `lichdenstein-dl`, connect
   `luxemburg-dl.kapital.click` and `lichdenstein-dl.kapital.click`.
2. Upload with rclone (multipart). The dashboard and `wrangler` refuse files this
   large.
3. Launcher: a `world` field in the manifest (URL, SHA-256, size), both hosts on
   the allowlist, a download that resumes and checks the hash before unpacking
   into the instance's `saves/`. Lichdenstein's world is a Paper world and gets
   merged into the singleplayer layout on unpack.
4. Site: Lichdenstein server download in the download popup, with a Konnekt
   popup when it starts.
5. Server package built from `kapital-mc/lichdenstein-server` plus the jars in
   `server-plugins.json`, never zipped from the live folder (it holds the Groq
   key, AdvancedLogin's password hashes, CoreProtect's database and player
   lists).

Storage: about 27 GB, $0.27 to $0.42 a month, downloads free.

## Phase 3: the sites

`*.pages.dev` and `*.workers.dev` keep answering after a custom domain is added,
so every launcher already installed keeps working. Nothing needs a redirect.

### launcher.kapital.click

1. Pages project `kapital-launcher`, Custom domains, Add:
   `launcher.kapital.click`.
2. Kapital-Launcher: `DownloadPage` in `backend/services/appupdate.go` (the update
   feed `version.json` follows from it), README and docs links.

### wiki.kapital.click

1. Pages project `kapitel-kapital`, Custom domains, Add: `wiki.kapital.click`.
2. Kapital-Launcher: `wiki.baseUrl` in `data/launcher.json`, the host on
   `AllowedManifestHosts` (keep `kapitel-kapital.pages.dev` for the old builds).
   The lore export and screenshots are fetched from `baseUrl`, so they follow.
3. Wiki repo: its own site URL (canonical links, sitemap, any absolute links).

### packs.kapital.click

1. Worker `kapital-packs`, Settings, Domains and Routes, Add custom domain:
   `packs.kapital.click`.
2. Kapital-Launcher: the four `packwiz` URLs in `data/launcher.json`, the host on
   the allowlist, `PACK` in `site/status.js`.

This also takes `alessandrogekeler` out of every pack URL players see.

## Phase 4: later

| Name | What | How |
|---|---|---|
| `kapital.click`, `www` | Redirect to `launcher.kapital.click` | Cloudflare Redirect Rule |
| `frangfurd.kapital.click` (and the other chapters) | Join address players type, reserved | Not a playit alias: the playit addresses stay as they are. Needs a route of its own first (see below). |
| `frangfurd-map.kapital.click` | The web map over https | Cloudflare Tunnel (`cloudflared`) on the server machine, replacing `http://spiral-reminders.tun.ply.gg:1111` |
| `security@`, `contact@kapital.click` | Mail for SECURITY.md and players | Cloudflare Email Routing, forwarding to your inbox, free |
| `status` | Server status page | Already part of the launcher site as `/status`; no own subdomain needed |

### A join address without playit

`<chapter>.kapital.click` stays unused until one of these is chosen:

| Route | Cost | Catch |
|---|---|---|
| Port forward on the home router, A record to the home IP | Free | Shows the home IP to every player; a changing IP needs a DDNS updater |
| A small VPS as a TCP relay (or running the server there) | A few euros a month | One more machine to keep up |
| Cloudflare Spectrum (TCP proxy) | Paid plan only | Check what the plan covers before relying on it |

The launcher's ping already follows SRV records, so any of these can use a port
other than 25565.

Keep `frangfurd.click` until it lapses on 2028-10-13: `dh.` and `ts.` are in
launchers already installed.
