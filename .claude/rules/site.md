---
paths:
  - "site/**"
---

# The download site

`site/README.md` covers the page, its links and the Cloudflare setup, and
`docs/adr/0009-download-site.md` records why it is built this way. The rules:

- **Four small scripts, bundled into one, with no storage and one request,
  to the site's own `/status`.**
  `src/detect.js` picks the visitor's system (data-os on the root, for the
  Download button and the link under it, and the download popup's radio);
  `src/popup.js` starts the popups' opening fade after their first frame and
  plays the closing before it hides them; `src/cycle.js` opens the next
  chapter every `duration.cycle`; `src/cards.js` scrambles the floating
  cards' text into the new chapter's and fills in the server card from
  `/status`, the one Pages Function (`functions/status.js`, whose ping is
  `status.js`, tested). It pings only the manifest's Frangfurd Germany
  address and passes on nothing but whether it answered and two counts. Vite bundles them into one file, and
  the build places it after the stylesheet as a classic script, so the parser
  waits for the stylesheet and no browser paints the page unstyled
  (`stylesheetFirst` in `vite.config.js`); keep it free of imports, exports and
  `import.meta`, which the build refuses, and declare nothing at the top level.
  The CSP in `public/_headers` is `default-src 'none'` with scripts, styles,
  images and fonts from the site itself. An inline script or style (a `style`
  attribute included), a data: URI or a remote asset means changing that
  policy on purpose, in the same change. Choosing a chapter needs no script:
  the panels are a radio group's labels and `src/showcase.css` reads the
  radio with `:has()`; the cycle only checks the next radio. Its icons are
  lucide's paths inlined as SVG markup, which the policy allows.
- **The app's tokens and base styles, not the site's.** `src/style.css`
  imports `frontend/src/styles/tokens.css` and `frontend/src/styles/base.css`
  (faces, base rules, scrims), then `src/fonts.css` (Silkscreen, the pixel
  face only the site draws, served from `src/fonts/` with its OFL licence),
  `src/showcase.css` (the chapter strip and the backdrop, the one thing the
  page draws that the app does not) and `src/popup.css` (the popups' opening
  and closing). A token change moves the app and
  the site together. The frontend style rules apply here: token utilities
  only, no literal colour, no pixel text size, no em dash. The suite's
  invariants cover `site/index.html` and `site/src`.
- **The window's header bar is a copy of the app's.** Its markup carries the
  classes of `shell/HeaderBar.tsx` and `HeaderTools.tsx` as Windows draws
  them; the strip under it is the page's own. The panels and the two cards
  beside the window carry the chapter facts of `data/launcher.json`. A change
  to the header bar or to those facts is a change to the site's copy, in the
  same pull request.
- **The backdrop's squares are written by the build.** `mosaic.js` fills
  `<!--mosaic:art-->`, `<!--mosaic:pixels-->` and `<!--mosaic:banner-->` with
  empty spans; `src/showcase.css` places, paints and times them. The grid
  sizes are in both files: change them together. `pnpm test` covers it.
- **Links are data.** `links.json` holds `release` (a tag, or `null`) and
  GitHub; the page has no Wiki button since 2026-10-06. Never write a URL for the
  buttons or the popup's downloads into `index.html`; the build derives the
  installer and release notes URLs from the tag and fills them, and rejects a
  tag that is not `vX.Y.Z` or `vX.Y.Z-alpha.N`/`-beta.N` and anything but
  https.
- **The changelog is the releases' own notes.** `changelog.js` reads the
  releases feed the build already fetches and renders Features, Fixes and
  Other changes into the Changelog popup, escaping every value. Never write a
  release's changes into `index.html` by hand; `pnpm test` covers the parser
  against a saved feed in `test/fixtures/`.
