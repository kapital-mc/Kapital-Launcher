import { readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'

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
async function resolveLatest(base) {
  const feed = await fetch(`${base}/releases.atom`, { signal: AbortSignal.timeout(15000) })
  if (!feed.ok) throw new Error(`site: ${base}/releases.atom answered ${feed.status}`)
  const xml = await feed.text()
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

async function readLinks() {
  const links = JSON.parse(readFileSync(new URL('./links.json', import.meta.url), 'utf8'))
  const all = { github: links.github }
  for (const [key, url] of Object.entries(all)) {
    if (url === null) continue
    if (typeof url !== 'string' || new URL(url).protocol !== 'https:') {
      throw new Error(`site: ${key} link must be an https URL or null, got ${JSON.stringify(url)}`)
    }
  }

  let tag = process.env.SITE_RELEASE ?? links.release
  if (tag === 'null') tag = null
  if (tag === 'latest') {
    if (all.github === null) throw new Error('site: release "latest" needs the github link')
    tag = await resolveLatest(all.github.replace(/\/+$/, ''))
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
    release = {
      tag,
      ...installerUrls(base, tag),
      NOTES_URL: `${base}/releases/tag/${tag}`,
    }
  }
  return { ...all, release }
}

const escapeAttr = (s) => s.replaceAll('&', '&amp;').replaceAll('"', '&quot;')

// The popup and its button sit between release markers in index.html: kept,
// with their placeholders filled, when a release is set, dropped when not.
const BLOCK = /[ \t]*<!--release:start-->[\s\S]*?<!--release:end-->[ \t]*\r?\n?/g

function fillRelease(html, release) {
  if (release === null) return html.replace(BLOCK, '')
  const out = html
    .replaceAll('<!--release:start-->', '')
    .replaceAll('<!--release:end-->', '')
    .replace(/%(WINDOWS_URL|MACOS_URL|NOTES_URL)%/g, (_, key) => escapeAttr(release[key]))
    .replaceAll('%TAG%', escapeAttr(release.tag))
  if (/%[A-Z_]+_URL%/.test(out)) throw new Error('site: an unfilled release placeholder is left')
  return out
}

function fillLinks(html, links) {
  const seen = new Set()
  const out = fillRelease(html, links.release)
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

async function links() {
  const resolved = await readLinks()
  return {
    name: 'kapital:links',
    transformIndexHtml: { order: 'pre', handler: (html) => fillLinks(html, resolved) },
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
  plugins: [tailwindcss(), await links(), stylesheetFirst()],
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
