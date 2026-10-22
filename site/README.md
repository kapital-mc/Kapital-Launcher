# Download site

One static page: the launcher's name, the Download button for the visitor's
system with the version and the other system under it, and the launcher window
with the three chapters as a strip of panels in front of a backdrop of the
open chapter's picture. The footer holds Changelog, GitHub and Code signing
policy. It is separate from the Kapitel Kapital wiki and runs two small
scripts, bundled into one file: one picks the operating system, one closes
the popups with motion. Neither touches the network or storage.
The decision record is `docs/adr/0009-download-site.md`.

## Where the look comes from

`src/style.css` imports the app's generated `frontend/src/styles/tokens.css`
and its shared `frontend/src/styles/base.css` (fonts, base rules, scrims),
then the site's own: `src/fonts.css` (Silkscreen, the page's pixel face,
`type.family.pixel` in the tokens, served from `src/fonts/` with its licence),
`src/showcase.css` (the chapter strip and the backdrop) and `src/popup.css`
(the popups' motion). `index.html` uses the Kapital Launcher logo (the
author's one-line artwork, 560px wide) as its `h1`, and the chapters'
pictures, titles and icons from `frontend/src/assets/`. Buttons and labels are
in Silkscreen, with bold only on the popups' headings; the Download buttons are
the logo's rose (`logo-gradient-top`) with square corners. A value the page
needs and the tokens do not have is a token to add in `design/tokens.json`,
the same rule as the app.

The page is dark whatever the visitor's system preference (`data-theme="dark"`
on the root): the artwork and the window are drawn for the dark palette.

## The window

Under the Download button is the launcher window: the app's header bar
(`shell/HeaderBar.tsx` and `HeaderTools.tsx`, as Windows draws them: the
compact tools on the left, minimise, maximise and close on the right), a
picture with nothing in it to press, and under it the three chapters as a
strip of panels. The open chapter's panel is wide, its picture at full
strength and its title large; the others are dimmed
(`effect.showcase.dim`). Width, picture and title ease between the two on
the slow duration.

The panels are the labels of a radio group (`name="chapter"`), so choosing a
chapter needs no script. Everything that follows the open chapter reads that
radio with `:has()` on the root, in `src/showcase.css`: the accent, the
backdrop, the download popup's banner, and the two cards beside the window,
the open chapter's pack (version, mods, loader) on the left and its state
(released or in development) on the right, both from `data/launcher.json`.
Frangfurd is open when the page loads.

### The backdrop

Behind the window, from its top edge down and as wide as the page, is the
open chapter's picture cut into a grid of 24 by 12 squares, under a second
grid of dark squares of the same size. `mosaic.js` writes both grids into
`index.html` when the site is built (`<!--mosaic:art-->` and
`<!--mosaic:pixels-->`), as empty spans with two data attributes: a delay
bucket (`data-d`) and the chapters each dark square belongs to (`data-m`).
The pattern is seeded, so every build writes the same page, and it is dark
along the top, open in the middle and ragged towards the bottom and the sides.
On a switch each square changes picture and darkness at its own delay, so the
new chapter replaces the old one square by square over the slow duration.

Each square paints only its own slice of the picture, sized as
`object-fit: cover` over the whole backdrop through container units, so the
slices join and nothing larger than a square is painted. The picture is
tinted toward the accent (`effect.showcase.tint`), lightly. It is not blurred:
a blur reads across the squares' edges, which then show seams mid-swap. The pictures' aspect ratios are in
`src/showcase.css` beside their URLs. Reduced motion swaps every square at
once.

Narrower than 1024px the two cards are left out; narrower than 768px the
panels stack.

A change to the header bar the window copies, or to the manifest's chapter
facts, is a change to this page in the same pull request.

## The links

| Button | Source | When `null` |
|---|---|---|
| Download | `links.json` `release`: `"latest"` or a tag | Shows "Download · coming soon", no popup |
| GitHub | `links.json` `github` | Left out |
| Prism zips, in the popup | The release, one per chapter with a `packwiz` URL in `data/launcher.json` (#251) | Left out, also when the release has not all of them |

`vite.config.js` fills them in at build time and refuses anything that is not
https. `release` is a tag such as `v1.0.0-beta.1`, checked against
`^v\d+\.\d+\.\d+(-(alpha|beta)\.\d+)?$`; from it and the `github` link the
build derives the two installers and the release notes:

- `<github>/releases/download/<tag>/Kapital-Launcher-<tag>-windows-amd64-setup.exe`
- `<github>/releases/download/<tag>/Kapital-Launcher-<tag>-macos-universal.dmg`
- `<github>/releases/tag/<tag>`

`release` is `"latest"`: the build picks the newest release, prereleases
included, whose two installers are both up. GitHub's own
`releases/latest/download/` would skip prereleases, which every release so far
is. It reads the repository's public releases feed (`<github>/releases.atom`),
not the REST API, so an unauthenticated build on a shared CI runner does not run
into the API's rate limit, and checks each installer link before choosing a
release. A network failure fails the build, and Cloudflare Pages then keeps the
deployment it has.

So the site follows a release on its next build: the next push to `main`, or
the Pages deploy hook the release workflow calls once a release is published
(the repository secret `SITE_DEPLOY_HOOK`, see `.github/workflows/build.yml`).

A tag in `release` pins the page to that release instead, and `null` shows
"coming soon". For a build without network access, `SITE_RELEASE` in the
environment overrides `links.json`, for example `SITE_RELEASE=v1.0.0-beta.2
pnpm build` or `SITE_RELEASE=null pnpm build`.

```json
{
  "release": "latest",
  "github": "https://github.com/sandrogekeler/Kapital-Launcher"
}
```

## The download popup

With a release set, each Download button opens a native popover
(`popovertarget`), no script needed: a banner of the open chapter's picture
through a fixed pattern of dark squares (`<!--mosaic:banner-->`) with the
version and a close button, a heading, a Windows or macOS switch, the one
download for the chosen system, a link to the release notes for how to open
it, and the Prism instance zips folded into a dropdown that slides open (a
checkbox and a grid row easing from `0fr` to `1fr`, so the popup grows with it
and no script is needed). The switch is a fieldset of two radios and CSS
`:has()`, so it works without script. Windows is checked by default.

There are two Download buttons and two links under them, one per system, and
`data-os` on the root shows one of each: the button for the visitor's system,
the link for the other. Without the script Windows is shown.

`src/detect.js` is bundled with `src/popup.js` into the page's one script
file, emitted by Vite under `assets/`. The build turns its tag into a classic script placed after the
stylesheet (`stylesheetFirst` in `vite.config.js`), which stops the parser
until the stylesheet has loaded. Without that, the body was parsed while the
stylesheet was still on its way, and Safari and Firefox could paint it
unstyled for a frame on every load. The script waits for `DOMContentLoaded`
before it looks for the radio, and the build refuses a script that imports,
exports or reads `import.meta`, since a classic script cannot. It reads the visitor's system
(`navigator.userAgentData?.platform`, then `navigator.platform`, then the user
agent), sets `data-os` on the root and checks that radio, and checks the radio
a button names (`data-os-pick`) when it is pressed. It does nothing else: no
network, no storage. iPhone, iPad, Android and Linux are neither, and keep
Windows. With `release` set to `null`, the
build leaves the popup out of the page.

Its heading is in the pixel face; the tag sits in the banner, filled in at
build time. It opens with the app's reveal (a short rise out of
a slight blur) while the page dims and softens behind it, and closes the same
way in reverse over the slow duration on the glide curve, so the fade is seen.
The other system's panel fades in when the switch moves: `src/popup.css`, on
the motion tokens. The changelog and code signing popups move the same way.
Both halves go through `src/popup.js`. Only Chromium keeps a closing popover
in the top layer while it fades, and other browsers drop it at once, so the
script makes each popover manual, plays the closing animation on the Close
button, Escape or a click outside, and hides the popup when it ends. In
Safari the first frame of an open popup, with the page blurred behind it, can
take longer than the whole opening fade, so the script also holds the popup
clear until that frame is drawn and only then lets the fade run. Without the
script the popups still open and close on CSS alone. A browser without
`@starting-style` shows and hides them plainly, and reduced motion turns the
motion off.

The CSP in `public/_headers` allows it with `script-src 'self'` and nothing
more, so an inline script would be blocked.

## The changelog

"Changelog", in the footer, opens a popup with each release's changes,
newest first. `changelog.js` makes it from the repository's releases feed,
the same `releases.atom` the build reads to pick the newest release: for each
release, what follows its notes' "What's changed" heading, sorted into
Features, Fixes and Other changes. Documentation and Maintenance are left out,
as `.github/scripts/release-notes.py` counts rather than lists them: they
change nothing a player runs. Each line is the pull request's title with a
link to it. Every value is escaped, and the links are built from the GitHub
link in `links.json`.

So the changelog follows a release the way the Download button does: the
release workflow's deploy hook rebuilds the site once a release is published,
and every push to `main` rebuilds it too. The feed holds the newest ten
releases, which is as far back as the popup goes. A build with a pinned tag
tries the feed and, if GitHub cannot be reached, links to the releases page
instead; `SITE_RELEASE` builds do not try. `pnpm test` runs the parser against
a saved copy of the real feed in `test/fixtures/releases.atom`.

The same releases go into `changelog.json` beside `version.json`, as plain
text without the links, for the full changelog under the launcher's What's new
(`backend/services/releasenotes.go`). A build that read no feed writes none.

## The code signing policy

Beside it, "Code signing policy" opens a popup the same way.
It holds the section SignPath Foundation
asks a project to publish on its download page (signpath.org/terms, issue
179): the attribution sentence, the two team roles, the privacy sentence, and
how to remove the app. The heading and those sentences are SignPath's wording
and stay as they are. The same text is in the repository's `README.md` and in
`.github/release-body.md`, which heads every release; change the three
together.

## Commands

```bash
pnpm install
pnpm build      # dist/
pnpm preview    # build, then serve dist/ locally
pnpm test       # the changelog parser, against test/fixtures/, and mosaic.js
```

There is no `vite` dev server script on purpose: the page's fonts and images
live in `frontend/`, outside the dev server's root, and only the build
resolves them. A build takes well under a second. `pnpm preview` does not
apply `public/_headers`; only Cloudflare does.

## Hosting

Cloudflare Pages, connected to this repository from the dashboard. There is no
Wrangler file: the settings below live in the Pages project.
`public/_headers` sets the CSP and caching and is copied into `dist/`.

In the Cloudflare dashboard under Workers & Pages, create a Pages project by
connecting this repository, then set:

| Setting | Value |
|---|---|
| Framework preset | None |
| Production branch | `main` |
| Root directory | `site` |
| Build command | `pnpm build` |
| Build output directory | `dist`, which is relative to the root directory. Not `site`: that is the unbuilt source, whose links are empty and whose fonts and images point into `frontend/`. |
| Environment variable | `PNPM_VERSION` = `12`, the version CI uses. The build image's default, 10.11.1, installs the site but predates `minimumReleaseAge` in `pnpm-workspace.yaml`, so the release-age policy would not apply. |
| Build watch paths | Include `site/*`, `frontend/src/styles/*`, `frontend/src/assets/*`, `data/launcher.json`. Paths are from the repository root. |

Pages installs the dependencies itself before the build command, from the root
directory's `pnpm-lock.yaml`. Every other branch gets a preview deployment.

These settings come from Cloudflare's Pages documentation as read on
2026-09-28 and have not been run on Cloudflare yet. The build was checked
locally, and the install with pnpm 12 and with 10.11.1.
