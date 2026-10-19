# Download site

One static page: the launcher's name, the version it offers, the Download and
GitHub buttons, and under them the launcher window itself, cycling
through its three chapters, whose rows a visitor can click. Above the footer,
Changelog and Code signing policy each open a popup. It is separate from the
Kapitel Kapital wiki and runs two small scripts, bundled into one file: one
picks the operating system in the download popup, the other makes the
window's rows clickable. Neither touches the network or storage.
The decision record is `docs/adr/0009-download-site.md`.

## Where the look comes from

Nothing visual is defined here. `src/style.css` imports the app's generated
`frontend/src/styles/tokens.css` and its shared `frontend/src/styles/base.css`
(fonts, base rules, scrims), and `index.html` uses the
Kapital Launcher logo (the author's one-line artwork, the file the app's header bar draws, 560px wide)
and a screenshot from `frontend/src/assets/`. The logo is the page's `h1`,
its alt text the heading. The buttons carry the
classes of the app's `Button` variants: `play` for Download, `ghost` for the
others. A value the page needs and the tokens do not have is a token to add in
`design/tokens.json`, the same rule as the app.

The page is dark whatever the visitor's system preference (`data-theme="dark"`
on the root): the artwork and the window are drawn for the dark palette.

## The window

Below the buttons is the launcher window at its real size
(`layout.window` in the tokens), drawn with the classes of the app's own
components (`frontend/src/components`): the header bar, the chapter nav with
its gliding highlight, the account card, and the chapter card with its hero,
action bar and panels, filled from `data/launcher.json`. The icons are
lucide's paths inlined as SVG, the same glyphs the app renders. Apart from
the nav's chapter rows, which are buttons (below), the window is a picture:
nothing else in it is a control, the rest is hidden from assistive
technology, and a sentence before it names the three chapters.

It cycles through the chapters on CSS animations alone, in `src/showcase.css`,
with the app's own motion: the card slides the way the content moves and
blurs at the middle of its travel (`frontend/src/style.css`), the nav
highlight glides to the next row in the same duration and easing, the row's
art fades in behind it, and the glow behind the window, the Download
button and the popup's download buttons take the open chapter's colour, all
from one animation on the page's root, so the popup never restarts the cycle. A chapter dwells for
`motion.duration.dwell`, then the next slides in over `motion.duration.slow`;
the keyframe stops in `showcase.css` are fractions of that cycle and say how
to recompute them if either duration changes. With reduced motion the cards
cross-fade on the same clock and the highlight steps instead of gliding.

### Clicking a chapter

`src/showcase.js` turns the rows into a switch. Once it runs it sets
`data-showcase="live"` on the root, which stops the CSS cycle, and runs the
same cycle itself: every step (dwell plus slow, read from the tokens) the next
chapter slides in. A click on a row shows that chapter with the same slide,
highlight glide and colour change, in the direction the app uses (down the
list from below, up the list from above), and restarts the clock, so the
chosen chapter stays a full dwell before the cycle moves on. The script only
sets attributes: which chapter is open (`data-showcase-open`, `aria-current`,
`data-open`) and which card enters or leaves which way (`data-motion`). The
motion itself is the `[data-showcase="live"]` rules in `src/showcase.css`, on
the same tokens. Without the script the rows are inert and the CSS cycle runs
as before.

Narrower than the app's minimum window (1024px) the panels and the window's
disclaimer are left out and the window takes the height of its content;
narrower than 768px the sidebar and the hero's tools go too, and the card
alone is shown.

A change to one of the components the window copies, or to the manifest's
chapter facts, is a change to this page in the same pull request.

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

With a release set, Download is a button that opens a native popover
(`popovertarget`), no script needed: a heading, a Windows or macOS switch, the
one download for the chosen system, the Prism instance zips folded into a
`details` dropdown, a one-line note that the build is not code-signed yet with
a link to the release notes, and a close button. The switch is a fieldset of two radios and CSS `:has()`, so it works
without script. Windows is checked by default.

`src/detect.js` is bundled with `src/showcase.js` into the page's one script
file, emitted by Vite under `assets/`. The build turns its tag into a classic script placed after the
stylesheet (`stylesheetFirst` in `vite.config.js`), which stops the parser
until the stylesheet has loaded. Without that, the body was parsed while the
stylesheet was still on its way, and Safari and Firefox could paint it
unstyled for a frame on every load. The script waits for `DOMContentLoaded`
before it looks for the radio, and the build refuses a script that imports,
exports or reads `import.meta`, since a classic script cannot. It checks the radio for the visitor's system
(`navigator.userAgentData?.platform`, then `navigator.platform`, then the user
agent) and does nothing else: no network, no storage. iPhone, iPad, Android and
Linux are neither, and keep Windows checked. With `release` set to `null`, the
build leaves the popup out of the page.

Its heading is in the display face, like the wiki's; the tag sits above it,
filled in at build time. It opens and closes with the app's reveal (a short
rise out of a slight blur) while the page dims and softens behind it, and the
other system's panel fades in when the switch moves: `src/popup.css`, on the
motion tokens, with no script. A browser without `@starting-style` shows and
hides it plainly, and reduced motion turns the transitions off.

The CSP in `public/_headers` allows it with `script-src 'self'` and nothing
more, so an inline script would be blocked.

## The changelog

"Changelog", above the footer, opens a popup with each release's changes,
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

## The code signing policy

Beside it, "Code signing policy" opens a popup the same way, with no script.
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
pnpm test       # the changelog parser, against test/fixtures/
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
