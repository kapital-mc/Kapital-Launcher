# ADR-0009: The download site

**Status:** accepted, 2026-09-28. Built, not yet deployed.

## Context

The launcher needs a public page with three ways out: download it, read its
source, read the wiki. The author asked for it separate from the Kapitel
Kapital wiki, kept simple, and hosted on Cloudflare Pages.

Three facts read on 2026-09-28 shaped the answer:

- Cloudflare's own documentation tells new projects to use Workers static
  assets instead of Pages: "If you are starting a new project, use Workers
  instead of Pages. Pages continues to work, but new features and
  optimizations are focused on Workers"
  (developers.cloudflare.com/workers/best-practices/workers-best-practices).
  New Pages projects can still be created, and the wiki is one.
- The repository is private and has no release, so a GitHub link shows a
  visitor a 404 and a Download link has nothing to point at.
- Everything the page looks like already exists here: the generated
  `tokens.css`, the fonts, the wordmark, a screenshot and the `Button`
  variants.

## Decision

- **`site/` in this repository**, its own pnpm project beside `frontend/`.
  It imports `frontend/src/styles/tokens.css`, the fonts and the images by
  relative path rather than copying them, so the page and the app cannot
  drift apart. Since 2026-09-29 (#16) it also imports the shared
  `frontend/src/styles/base.css` (faces, base rules, scrims) instead of
  keeping its own copy, and adds no style of its own. The page's heading
  is the Kapital Launcher logo, the same artwork as the app's header
  (ADR-10), in place of the Kapitel Kapital wordmark. Since 2026-10-01 the
  page shows the launcher window under the buttons, built from the app's
  components' classes and cycling through the three chapters on CSS
  animations in `src/showcase.css`, the site's one stylesheet of its own; a
  dwell duration and the accent mix percentages joined the tokens for it, and
  the window's size is emitted to CSS.
- **Vite and Tailwind v4, no framework, no script** (one small script since
  2026-10-04, see the amendment). One HTML page styled with
  the same token utilities as the app. The built output is HTML, one CSS file,
  fonts and two images.
- **Cloudflare Pages**, connected from the dashboard: root directory `site`,
  build command `pnpm build`, output directory `dist`. The author chose it
  over Workers static assets, having weighed Cloudflare's guidance above: it
  is what the wiki runs on, and a static page needs nothing Workers adds.
  There is no Wrangler file and no Wrangler dependency; the project's
  settings live in the dashboard and are written down in `site/README.md`.
- **The links are data.** `site/links.json` held Download and GitHub as an
  https URL or `null`; the wiki comes from `data/launcher.json`'s
  `wiki.baseUrl`. A build-time Vite plugin fills them in: a null Download
  shows "coming soon", a null GitHub is left out, and a non-https URL fails
  the build. Both start null.
- **A strict CSP in `site/public/_headers`**: `default-src 'none'` with
  styles, images and fonts from the site itself. Possible only because the
  page had no script and inlines no asset (see the amendment).

## Consequences

- The site depends on files inside `frontend/`. Moving the token output, the
  fonts or the screenshots breaks the site build, which CI runs in its own
  job for that reason.
- The window on the page is a copy of the app's screens, not a render of
  them. A component or a chapter fact that changes in the app has to change
  in `site/index.html` too; nothing checks that they agree.
- Deploys are Pages' own builds. CI builds the site but does not deploy it.
  Every other branch gets a Pages preview deployment.
- Pages is the platform Cloudflare no longer develops. If it is ever retired
  or the site needs something only Workers has, the move is a `wrangler.jsonc`
  with `assets.directory` set to `./dist`
  (developers.cloudflare.com/workers/static-assets/migration-guides/migrate-from-pages);
  the page itself does not change.
- Until a custom domain is added the site lives on `<project>.pages.dev`.
- Download and GitHub go live by editing one file, when milestone 7's
  release workflow exists and the repository is public.

## Amendment, 2026-10-04 (issue 178)

The Download button now opens a popup with the build for the visitor's
operating system and a switch to the other, so the page runs one script:
`src/detect.js`, built by Vite into a file under `assets/`. It reads
`navigator.userAgentData?.platform` (or `navigator.platform`, or the user
agent), and checks the Windows or macOS radio. It makes no request and stores
nothing. The popup itself is the browser's popover and the switch is two
radios read by CSS `:has()`, so without the script, or on another system,
Windows is preselected and the switch still works. The reason for a script at
all is that CSS cannot read the operating system.

The CSP gains `script-src 'self'` and nothing else: no inline script, no
`unsafe-*`, no remote host. The rest of the policy is unchanged. The links
file now names a release tag (`release`, or `null`) in place of a Download
URL; the build derives the two installer URLs and the release notes URL from
the tag and the GitHub link, because `releases/latest/download/` skips
prereleases. A `null` release keeps "Download · coming soon" and leaves the
popup out of the page.

## Amendment, 2026-10-06

The Wiki button is gone, at the author's request: the page offers Download and
GitHub only. The build no longer reads `data/launcher.json` for a link, and a
`data-link="wiki"` in `index.html` would now fail it as an unknown key. The
launcher itself still links the wiki from each chapter's "From the wiki" panel.

## Amendment, 2026-10-12

The page flashed unstyled for a frame on every load: raw text and the inlined
icons at full width. Vite wrote the script as a module before the stylesheet,
and a module never stops the parser, so the whole body was parsed while the
stylesheet was still loading. Chromium holds the first paint for a stylesheet
in the head; Safari and Firefox can paint what is parsed before it arrives.
The build now turns the script's tag into a classic script after the
stylesheet, which the parser waits on, so there is no body to paint until it
can be painted styled. An empty inline script would do the same, but the CSP
forbids inline script, and this keeps the page at one script and the policy
unchanged. `src/detect.js` waits for `DOMContentLoaded`, since it now runs
before the popup is parsed.

## Amendment, 2026-10-12: the newest release, resolved at build time

The tag in `links.json` was bumped by hand for every release, so the page kept
offering `v1.0.0-beta.1` after `v1.0.0-beta.2` was out. `release` may now be
`"latest"`, and is: the build resolves it to the newest release, prereleases
included, whose two installers are both up.

- **From the releases feed, not the API.** The build reads
  `<github>/releases.atom`, which lists releases newest first and has no REST
  API rate limit. CI builds the site without a token on shared runners, where
  the API's unauthenticated limit of 60 requests an hour is easily used up.
  Each candidate's installer links are checked with a `HEAD` request, so a
  release still uploading is passed over for the one before it.
- **Resolved when built, not when visited.** The page stays static, with no
  request to GitHub from a visitor's browser and no change to the CSP. The
  cost is that the page follows a release on its next build, not at once: the
  next push to `main`, or the Pages deploy hook the release workflow
  (`build.yml`, "Rebuild the download site") calls once a release is
  published. The hook URL is the repository secret `SITE_DEPLOY_HOOK`;
  without it, or when the call fails, the release still succeeds and the site
  follows on its next build.
- **Failures.** A network failure or a feed with no complete release fails the
  build, and Pages keeps the deployment it has. `SITE_RELEASE` in the
  environment overrides `links.json` for a build without network access. A tag
  in `links.json` still pins the page to that release, and `null` still shows
  "coming soon".

## Amendment, 2026-10-14: the launcher asks the site for the newest version

A player on an older build had no way to learn of a newer one but to visit the
page. The author asked for a popup that opens the site, from the next release
on, and chose the site over GitHub as the source and "ask again next start"
over skipping a version.

- **The site publishes the version.** Its build writes `version.json`,
  `{"version": "<tag>"}`, with the same tag the Download button offers (the
  newest release with both installers, the amendment above), and none when it
  offers no release. `public/_headers` serves it `no-cache`. The launcher so
  never announces a version the page does not offer yet, and it contacts no
  host it did not already know.
- **The launcher reads it once a start** (`services.CheckUpdate`,
  `GetUpdate`): one GET of the fixed address, 8 s, no redirect, a 200 and
  4 KiB at most, the value held to the release tag pattern, and compared with
  `main.Version` by semantic versioning precedence (a release above its
  prereleases, alpha below beta). A dev build asks nothing. Not reaching the
  site shows nothing, with the reason in the log. Nothing about the player or
  the build is sent (SECURITY_CHECKLIST S4.9).
- **The popup only points the way.** "A new version is out" names both
  versions and opens the site in the browser; the player installs as they did
  the first time. Not now and Escape put it away until the next start. The
  launcher downloads and installs nothing itself, so ADR-6's signing questions
  do not reach this.
- **Seen before a release carries it** through the developer preview
  "Kapital Launcher update available" (#124), which makes the answer up and
  fetches nothing.
- **From the next release on.** Builds up to `v1.0.0-beta.2` have no check;
  their players learn of the update the old way.


## Amendment, 2026-10-18: a plain hero, clickable chapters and a changelog

The author asked for three changes to the page.

- **The hero is the name, the version and the buttons.** The description
  under the logo, the line on supported systems (the download popup names
  them) and the line on Prism Launcher are gone. The release tag the page
  offers sits between the logo and the buttons, filled in at build time like
  the popup's.
- **A quieter download popup.** The notes under each download button are
  gone, the Prism instance zips sit in a closed dropdown that slides open and
  grows the popup with it (a checkbox and a grid row eased from `0fr` to
  `1fr`; a `details` element cannot animate its height in Safari yet), and the
  code signing notice is one line linking the release notes. Every popup
  now closes visibly: the closing half runs over the slow duration on the
  glide curve, where it had the opening's fast-out curve and was over in its
  first frames. CSS alone plays it only in Chromium, the one engine that
  keeps a closing popover in the top layer (the `overlay` transition), so a
  third small script, `src/popup.js`, makes the popovers manual and hides
  each one when its closing animation ends. Escape and a click outside still
  close it, as the browser's light dismiss did. On 2026-10-20 the author
  found no opening motion in Safari either: there the first frame of an open
  popup can outlast the 200 ms fade, so the fade was over before it was
  drawn. The script now holds the popup clear for two frames before the
  opening transition runs. The author also saw a clipped glow flash when hovering the
  Download button: the glow behind the window was a box under a large blur
  filter, which Safari repaints in pieces when something beside it changes.
  It is now a radial gradient, with no filter.
- **The window's chapter rows are buttons.** A click shows that chapter with
  the app's own motion and restarts the cycle, so the chosen chapter stays a
  full dwell before the next slides in. This needs a script: the CSS cycle
  cannot be restarted from a chosen point without one. So the page now runs a
  second small script, `src/showcase.js`, bundled with `src/detect.js` into
  the same one classic script file. It touches neither the network nor
  storage, and only sets attributes; the motion stays in `src/showcase.css`
  on the tokens. Without it the CSS cycle runs as before. The rows are now
  exposed to assistive technology as buttons; the rest of the window stays
  hidden from it. The CSP is unchanged: the script is still `'self'`.
- **A changelog popup.** The build already reads the releases feed to pick
  the newest release; `changelog.js` now also renders each release's
  Features, Fixes and Other changes from it, escaped, into a popup. It
  follows a release on the same rebuild as the Download button. The code
  signing policy moved from a `details` element into a popup of its own, and
  the two open from a pair of quiet buttons above the footer. SignPath's
  wording is unchanged.

## Amendment, 2026-10-19: What's new after an update

The author asked for a page on the first start after an update that shows a
release's main changes by name rather than its list of pull requests, with
small changes grouped ("Bug fixes", "UI improvements") and the full changelog
under a dropdown.

- **The highlights are written by hand and built in.** `data/whatsnew.json`
  holds each release's highlights, newest first in the popup: a title with a
  line, or with the small changes it groups, or both. It is written before the
  release is cut, since the build embeds it; a release without an entry shows
  nothing. `ParseWhatsNew` refuses the file whole at startup on a fault, so a
  mistake fails CI rather than a player's start.
- **Once per update.** Go records the version seen (`seenVersion`, written by
  Go alone) when the popup is closed, and shows every release since it up to
  this build. A fresh install has nothing new and records its version
  silently. A launcher from before this change left no version: its player
  sees this build's entry only. A dev build shows nothing.
- **The full changelog comes from the site.** Its build writes
  `changelog.json` beside `version.json`, the Changelog popup's releases as
  plain text, served `no-cache`. The launcher reads it only when the dropdown
  is opened, from the fixed address, under the update check's limits with
  256 KiB in place of 4 KiB, and keeps the releases What's new covers. Not
  reaching it says so and links to the site (SECURITY_CHECKLIST S4.12).
- **Seen before a release carries it** through the developer preview "What's
  new after an update", which shows the newest entry, makes the changelog up,
  fetches nothing and records nothing.

## Amendment, 2026-10-20: the update offer under its button

The author asked for the update offer to open under the header's update
button rather than over the window, with a Download button to the site and a
smaller way out. The offer is now a panel that drops out of the button
(`UpdateNotice`, which also draws the button): "Update available", both
versions, one line, Download and Not now. It opens by itself once a start as
before; Not now, Escape or a press anywhere else folds it back into the
button until the button opens it again. What it does is unchanged: it opens
`services.DownloadPage` and nothing else. Not now, Cancel and Close across
the launcher's popups and prompts are the same small borderless button
(`Button`'s `quiet`), so the action beside them leads. What's new keeps its X.

## Amendment, 2026-10-22: the pixel window

The author redesigned the page with Claude over a set of mockups, after
essential.gg and other Minecraft sites: no sales line, a page that makes the
download easy and looks good, at most two screens tall. What changed:

- **The window is a strip of chapters, not the app's layout.** The header bar
  is still the app's (`HeaderBar`, tools on the left, Windows' buttons on the
  right), but under it the three chapters are panels side by side; the open
  one is wide with its picture at full strength, the others dimmed. The cycle
  is gone. The panels are the labels of a radio group, so choosing a chapter
  needs no script, and `src/showcase.js` is removed: the page runs two
  scripts, `detect.js` and `popup.js`.
- **A backdrop of squares.** Behind the window the open chapter's picture
  shows through a grid of dark squares and changes square by square on a
  switch. The squares are spans `mosaic.js` writes at build time, since the
  CSP allows no `style` attribute; each paints only its own slice of the
  picture, so the page never paints a full-size copy per square (an early
  mockup that did so glitched the whole viewport). The picture is lightly
  blurred and tinted toward the accent: `effect.showcase` joined the tokens,
  with the dim of a closed panel.
- **A pixel face.** Buttons, labels and the popups' headings are in
  Silkscreen (OFL), the face of the first mockup the author liked, served
  from the site like the others; it is `type.family.pixel` in the tokens. The
  Download buttons are the logo's rose with square corners.
- **One Download button for the visitor's system**, with the version and the
  other system under it; both open the download popup with that system
  picked. The popup has a banner of the open chapter's picture. Changelog,
  GitHub and the code signing policy moved to the footer, and the logo is the
  only thing above the button.
- **Cards from the manifest, not live data.** Beside the window are the open
  chapter's pack (version, mods, loader) and its state. A live player count
  was in the mockups, but the page does not touch the network, so it shows
  whether the chapter is released instead.
- **No operating system logos.** Apple's trademark guidelines do not allow
  the Apple logo on a website without a licence, and Microsoft's logos only
  under its brand guidelines, so the systems are named in words.

