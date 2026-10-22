import { readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import { parseChangelog, renderChangelog } from './changelog.js'
import { backdropSquares, bannerSquares, renderBackdropArt, renderBackdropPixels, renderBanner } from './mosaic.js'

// The page's links, filled in at build time so the built HTML carries no inline
// script. GitHub and `release` come from links.json and stay null until there
// is something public to point at: a null GitHub renders nothing, a null
// release renders the "Download · coming soon" fallback and leaves out the
// download popup. A release is a tag, and the two installer URLs and the
// release notes URL are derived from it and the GitHub link, so nothing is
// written twice. `release: "latest"` resolves the tag when the site is built
// (resolveLatest below), and SITE_RELEASE in the environment overrides
// links.json for a build without network access.
const TAG = /^v\d+\.\d+\.\d+(-(alpha|beta)\.\d+)?$/

const installerUrls = (base, tag) => ({
  WINDOWS_URL: `${base}/releases/download/${tag}/Kapital-Launcher-${tag}-windows-amd64-setup.exe`,
  MACOS_URL: `${base}/releases/download/${tag}/Kapital-Launcher-${tag}-macos-universal.dmg`,
})

// The newest release that has both installers, prereleases included, which
// GitHub's releases/latest/download/ would skip. Read from the repository's
// public releases feed rather than the REST API: the feed has no API rate
// limit, so an unauthenticated build on a shared CI runner does not fail on
// one. The feed lists newest first. A release whose installers are not up yet
// (a release still uploading) is passed over for the one before it. Any network
// failure fails the build: Pages then keeps the deployment it has.
async function fetchFeed(base) {
  const feed = await fetch(`${base}/releases.atom`, { signal: AbortSignal.timeout(15000) })
  if (!feed.ok) throw new Error(`site: ${base}/releases.atom answered ${feed.status}`)
  return feed.text()
}

async function resolveLatest(base, xml) {
  const tags = [...xml.matchAll(/\/releases\/tag\/([^"<\s]+)"/g)].map((m) => decodeURIComponent(m[1]))
  for (const tag of tags) {
    if (!TAG.test(tag)) continue
    const found = await Promise.all(
      Object.values(installerUrls(base, tag)).map((url) =>
        fetch(url, { method: 'HEAD', redirect: 'manual', signal: AbortSignal.timeout(15000) }).then(
          (r) => r.status === 302 || r.ok,
        ),
      ),
    )
    if (found.every(Boolean)) return tag
  }
  throw new Error(`site: no release on ${base} has both installers`)
}

// The Prism instance zips a release carries (#251), one per chapter with a
// hosted pack in data/launcher.json, named as PrismZipName names them in
// backend/services/prisminstancezip.go. Each fills %PRISM_<ID>_URL%.
const ZIP_NAME = /^[A-Za-z0-9]{1,32}$/

function prismZips(base, tag) {
  const manifest = JSON.parse(readFileSync(new URL('../data/launcher.json', import.meta.url), 'utf8'))
  const zips = {}
  for (const chapter of manifest.chapters) {
    if (!chapter.pack.packwiz) continue
    if (!ZIP_NAME.test(chapter.name)) throw new Error(`site: no zip name for chapter ${JSON.stringify(chapter.name)}`)
    zips[`PRISM_${chapter.id.toUpperCase()}_URL`] =
      `${base}/releases/download/${tag}/Kapitel-Kapital-${chapter.name}.zip`
  }
  return zips
}

// Whether the release has every zip. A release made before the zips existed, or
// a build that cannot reach GitHub, has none to offer, and the page leaves the
// Prism section out rather than link to nothing.
async function hasAll(urls) {
  try {
    const found = await Promise.all(
      urls.map((url) =>
        fetch(url, { method: 'HEAD', redirect: 'manual', signal: AbortSignal.timeout(15000) }).then(
          (r) => r.status === 302 || r.ok,
        ),
      ),
    )
    return found.length > 0 && found.every(Boolean)
  } catch {
    return false
  }
}

// The changelog popup's body, from the same feed: each release's own notes,
// so the page follows a release the way its Download button does. A build
// that resolves "latest" has the feed already and fails without it; one with
// a pinned tag tries it and, if GitHub cannot be reached, links to the
// releases instead. SITE_RELEASE means a build without network access, so it
// does not try.
function changelog(base, xml) {
  if (base === null) return { html: null, releases: [] }
  const releases = xml === null ? [] : parseChangelog(xml, base)
  if (xml !== null && releases.length === 0) console.warn('site: the releases feed gave no changelog entries')
  return { html: renderChangelog(releases, base), releases }
}

async function readLinks() {
  const links = JSON.parse(readFileSync(new URL('./links.json', import.meta.url), 'utf8'))
  const all = { github: links.github }
  for (const [key, url] of Object.entries(all)) {
    if (url === null) continue
    if (typeof url !== 'string' || new URL(url).protocol !== 'https:') {
      throw new Error(`site: ${key} link must be an https URL or null, got ${JSON.stringify(url)}`)
    }
  }
  const github = all.github === null ? null : all.github.replace(/\/+$/, '')

  let tag = process.env.SITE_RELEASE ?? links.release
  if (tag === 'null') tag = null
  let xml = null
  if (tag === 'latest') {
    if (github === null) throw new Error('site: release "latest" needs the github link')
    xml = await fetchFeed(github)
    tag = await resolveLatest(github, xml)
  } else if (github !== null && process.env.SITE_RELEASE === undefined) {
    xml = await fetchFeed(github).catch(() => null)
  }
  let release = null
  if (tag !== null) {
    if (typeof tag !== 'string' || !TAG.test(tag)) {
      throw new Error(
        `site: release must be a tag like v0.1.0-beta.1, "latest" or null, got ${JSON.stringify(tag)}`,
      )
    }
    if (all.github === null) throw new Error('site: a release needs the github link')
    const base = all.github.replace(/\/+$/, '')
    const zips = prismZips(base, tag)
    release = {
      tag,
      ...installerUrls(base, tag),
      NOTES_URL: `${base}/releases/tag/${tag}`,
      prism: (await hasAll(Object.values(zips))) ? zips : null,
    }
  }
  const notes = changelog(github, xml)
  return { ...all, release, changelog: notes.html, releases: notes.releases }
}

const escapeAttr = (s) => s.replaceAll('&', '&amp;').replaceAll('"', '&quot;')

// The popup and its button sit between release markers in index.html: kept,
// with their placeholders filled, when a release is set, dropped when not.
const BLOCK = /[ \t]*<!--release:start-->[\s\S]*?<!--release:end-->[ \t]*\r?\n?/g

// The Prism section of the popup sits between its own markers, kept when the
// release has the zips.
const PRISM_BLOCK = /[ \t]*<!--prism:start-->[\s\S]*?<!--prism:end-->[ \t]*\r?\n?/g

function fillPrism(html, zips) {
  if (zips === null) return html.replace(PRISM_BLOCK, '')
  return html
    .replaceAll('<!--prism:start-->', '')
    .replaceAll('<!--prism:end-->', '')
    .replace(/%(PRISM_[A-Z]+_URL)%/g, (match, key) => (key in zips ? escapeAttr(zips[key]) : match))
}

function fillRelease(html, release) {
  if (release === null) return html.replace(BLOCK, '')
  const out = fillPrism(html, release.prism)
    .replaceAll('<!--release:start-->', '')
    .replaceAll('<!--release:end-->', '')
    .replace(/%(WINDOWS_URL|MACOS_URL|NOTES_URL)%/g, (_, key) => escapeAttr(release[key]))
    .replaceAll('%TAG%', escapeAttr(release.tag))
  if (/%[A-Z_]+_URL%/.test(out)) throw new Error('site: an unfilled release placeholder is left')
  return out
}

// The changelog popup and its button sit between changelog markers, dropped
// with no GitHub link to read releases from; its body replaces <!--changelog-->.
const CHANGELOG_BLOCK = /[ \t]*<!--changelog:start-->[\s\S]*?<!--changelog:end-->[ \t]*\r?\n?/g

function fillChangelog(html, body) {
  if (body === null) return html.replace(CHANGELOG_BLOCK, '')
  if (!html.includes('<!--changelog-->')) throw new Error('site: index.html has no <!--changelog--> placeholder')
  return html
    .replaceAll('<!--changelog:start-->', '')
    .replaceAll('<!--changelog:end-->', '')
    .replace('<!--changelog-->', () => body)
}

function fillLinks(html, links) {
  const seen = new Set()
  const out = fillRelease(fillChangelog(html, links.changelog), links.release)
    .replace(/<a\s+data-link="(\w+)"([^>]*)>([\s\S]*?)<\/a\s*>/g, (_, key, attrs, body) => {
      if (key === 'release' || !(key in links)) {
        throw new Error(`site: index.html links "${key}", links.json has no such key`)
      }
      seen.add(key)
      const url = links[key]
      return url ? `<a href="${escapeAttr(url)}"${attrs}>${body}</a>` : ''
    })
    .replace(/<span\s+data-link-fallback="(\w+)"([^>]*)>([\s\S]*?)<\/span\s*>/g, (_, key, attrs, body) =>
      links[key] ? '' : `<span${attrs}>${body}</span>`,
    )
  for (const key of ['github']) {
    if (!seen.has(key)) throw new Error(`site: index.html has no data-link="${key}"`)
  }
  return out
}

// version.json is the release the page offers, for the launcher's update notice
// (backend/services/appupdate.go, ADR-9 amendment): the same tag as the Download
// button, so the launcher never announces a version the page does not offer
// yet. A build with no release writes none, and the launcher shows no notice.
function versionFile(release) {
  if (release === null) return ''
  return `${JSON.stringify({ version: release.tag })}\n`
}

// changelog.json is the same releases as the Changelog popup, as plain text,
// for the full changelog under the launcher's What's new
// (backend/services/releasenotes.go). A build that read no feed writes none,
// and the launcher links to this page instead.
function changelogFile(releases) {
  if (releases.length === 0) return ''
  const plain = releases.map(({ tag, date, sections }) => ({
    tag,
    date,
    sections: sections.map(({ title, items }) => ({ title, items: items.map((item) => item.text) })),
  }))
  return `${JSON.stringify({ releases: plain })}\n`
}

async function links() {
  const resolved = await readLinks()
  const version = versionFile(resolved.release)
  const notes = changelogFile(resolved.releases)
  return {
    name: 'kapital:links',
    transformIndexHtml: { order: 'pre', handler: (html) => fillLinks(html, resolved) },
    generateBundle() {
      if (version) this.emitFile({ type: 'asset', fileName: 'version.json', source: version })
      if (notes) this.emitFile({ type: 'asset', fileName: 'changelog.json', source: notes })
    },
  }
}

// The squares of the chapter backdrop and the download popup's banner
// (mosaic.js), written in where index.html marks them. Each marker is
// expected exactly once.
const MOSAIC = {
  '<!--mosaic:art-->': () => renderBackdropArt(backdropSquares()),
  '<!--mosaic:pixels-->': () => renderBackdropPixels(backdropSquares()),
  '<!--mosaic:banner-->': () => renderBanner(bannerSquares()),
}

function mosaic() {
  return {
    name: 'kapital:mosaic',
    transformIndexHtml: {
      order: 'pre',
      handler(html) {
        let out = html
        for (const [marker, render] of Object.entries(MOSAIC)) {
          const count = out.split(marker).length - 1
          // The banner is in the download popup, which a build with no release leaves out.
          if (count === 0 && marker === '<!--mosaic:banner-->') continue
          if (count !== 1) throw new Error(`site: index.html has ${count} ${marker}, expected one`)
          out = out.replace(marker, render)
        }
        return out
      },
    },
  }
}

// Vite writes the page's script as a module before the stylesheet. A module
// never stops the parser, so the whole body is parsed while the stylesheet is
// still loading. Chromium holds the first paint for a stylesheet in the head
// anyway, but Safari and Firefox can paint what is parsed before it arrives
// (Firefox whenever something asks for layout early, an extension included),
// and that frame is the page unstyled: raw text and full-width icons.
// A classic script after the stylesheet stops the parser until the stylesheet
// has loaded, so there is no body to paint until it can be painted styled. The
// script is detect.js, which waits for DOMContentLoaded and so can run in the
// head; an inline script would do as well but the CSP forbids one.
const SCRIPT = /[ \t]*<script type="module" crossorigin src="([^"]+)"><\/script>\r?\n?/g
const SHEET = /<link rel="stylesheet" crossorigin href="[^"]+">/g

function stylesheetFirst() {
  return {
    name: 'kapital:stylesheet-first',
    apply: 'build',
    generateBundle(_, bundle) {
      for (const chunk of Object.values(bundle)) {
        if (chunk.type !== 'chunk') continue
        if (chunk.imports.length || chunk.dynamicImports.length || chunk.exports.length) {
          throw new Error(`site: ${chunk.fileName} imports or exports, so it cannot run as a classic script`)
        }
        if (chunk.code.includes('import.meta')) {
          throw new Error(`site: ${chunk.fileName} reads import.meta, so it cannot run as a classic script`)
        }
      }
    },
    transformIndexHtml: {
      order: 'post',
      handler(html) {
        const scripts = [...html.matchAll(SCRIPT)]
        const sheets = html.match(SHEET) ?? []
        if (scripts.length !== 1 || sheets.length !== 1) {
          throw new Error(
            `site: expected one module script and one stylesheet in the built page, found ${scripts.length} and ${sheets.length}`,
          )
        }
        const [tag, src] = scripts[0]
        return html.replace(tag, '').replace(sheets[0], `${sheets[0]}\n    <script src="${src}"></script>`)
      },
    },
  }
}

export default defineConfig(async () => ({
  plugins: [tailwindcss(), await links(), mosaic(), stylesheetFirst()],
  build: {
    // The script runs as a classic script (stylesheetFirst above), and the page
    // has no module to preload, so the module preload polyfill has no work.
    modulePreload: { polyfill: false },
    // Every asset is a file, never a data: URI, so the CSP in public/_headers
    // can stay at img-src 'self' and font-src 'self'. The script is a file too,
    // never inline, so script-src 'self' is all it needs.
    assetsInlineLimit: 0,
  },
}))
