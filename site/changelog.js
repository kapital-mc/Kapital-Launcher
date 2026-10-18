// The changelog popup's body, made from the repository's releases feed. The
// feed rather than the REST API, as resolveLatest in vite.config.js reads it:
// it has no API rate limit, so an unauthenticated build on a shared CI runner
// does not fail on one. It holds only the newest 10 releases, which is as far
// back as the popup goes. Plain functions over strings with no dependencies:
// parseChangelog reads GitHub's own generated markup, renderChangelog escapes
// everything it writes.
const TAG = /^v\d+\.\d+\.\d+(-(alpha|beta)\.\d+)?$/

// What a player is owed: the app's features, its fixes and anything else that
// changed it. Documentation and Maintenance are counted in the release notes
// (.github/scripts/release-notes.py), not listed, and so are dropped here.
const KEPT = { features: 'Features', fixes: 'Fixes', 'other changes': 'Other changes' }

const MONTHS = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
]

const ENTITIES = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'" }

// One pass, so "&amp;lt;" decodes to "&lt;" and no further.
const decode = (s) =>
  s.replace(/&(#x[0-9a-f]+|#\d+|amp|lt|gt|quot|apos);/gi, (match, name) => {
    if (name[0] !== '#') return ENTITIES[name.toLowerCase()]
    const code = name[1].toLowerCase() === 'x' ? parseInt(name.slice(2), 16) : parseInt(name.slice(1), 10)
    return code > 0 && code <= 0x10ffff ? String.fromCodePoint(code) : match
  })

const text = (html) => decode(html.replace(/<[^>]*>/g, '')).replace(/\s+/g, ' ').trim()

// GitHub's generated notes end an item with "by @user in #123", or with the
// pull request's URL in place of "#123". The reader gets the sentence only.
const ATTRIBUTION = /\s+by @[\w-]+(?:\[bot\])?(?:\s+in\s+(?:https?:\/\/\S+|#\d+))?$/

function pullNumber(li, base) {
  for (const [, href] of li.matchAll(/href="([^"]*)"/g)) {
    const url = decode(href)
    const found = url.startsWith(`${base}/pull/`) && /\/pull\/(\d+)$/.exec(url)
    if (found) return Number(found[1])
  }
  return null
}

function parseItems(html, base) {
  return [...html.matchAll(/<li\b[^>]*>([\s\S]*?)<\/li>/gi)]
    .map(([, li]) => ({ text: text(li).replace(ATTRIBUTION, ''), pr: pullNumber(li, base) }))
    .filter((item) => item.text !== '')
}

// What follows the "What's changed" heading, up to the next h2 or the "Full
// Changelog" line. GitHub's generated notes capitalise the C, the repository's
// own script does not, and GitHub may wrap a heading in a div with an anchor.
function changes(html) {
  const headings = [...html.matchAll(/<h2\b[^>]*>([\s\S]*?)<\/h2>/gi)]
  const start = headings.find((h) => text(h[1]).replace('’', "'").toLowerCase() === "what's changed")
  if (!start) return ''
  const from = start.index + start[0].length
  const next = headings.find((h) => h.index > start.index)
  let body = html.slice(from, next ? next.index : undefined)
  const end = body.search(/<p\b[^>]*>\s*<strong>\s*Full Changelog/i)
  if (end !== -1) body = body.slice(0, end)
  return body
}

function parseSections(html, base) {
  const body = changes(html)
  const heads = [...body.matchAll(/<h3\b[^>]*>([\s\S]*?)<\/h3>/gi)]
  const sections = []
  heads.forEach((head, i) => {
    const title = KEPT[text(head[1]).toLowerCase()]
    if (!title) return
    const items = parseItems(body.slice(head.index + head[0].length, heads[i + 1]?.index), base)
    if (items.length) sections.push({ title, items })
  })
  return sections
}

export function parseChangelog(xml, base) {
  base = base.replace(/\/+$/, '')
  const releases = []
  for (const [entry] of xml.matchAll(/<entry\b[\s\S]*?<\/entry>/g)) {
    const link = /releases\/tag\/([^"<\s]+)"/.exec(entry)
    const title = /<title[^>]*>([^<]*)<\/title>/.exec(entry)
    const tag = link ? decodeURIComponent(link[1]) : text(title?.[1] ?? '')
    const date = /<updated>\s*(\d{4}-\d{2}-\d{2})/.exec(entry)
    if (!TAG.test(tag) || !date) continue
    const content = /<content\b[^>]*>([\s\S]*?)<\/content>/.exec(entry)
    releases.push({ tag, date: date[1], sections: content ? parseSections(decode(content[1]), base) : [] })
  }
  return releases
}

const ESCAPES = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }
const escape = (s) => String(s).replace(/[&<>"']/g, (c) => ESCAPES[c])

const prose = 'text-fg-soft m-0 text-sm leading-normal'
const inline = 'text-accent underline-offset-2 hover:underline'

function renderSection({ title, items }, base) {
  const lis = items.map(({ text: label, pr }) => {
    const href = escape(`${base}/pull/${pr}`)
    const link = pr === null ? '' : ` <a class="text-fg-muted underline-offset-2 hover:underline" href="${href}">#${pr}</a>`
    return `<li>${escape(label)}${link}</li>`
  })
  return `<h4 class="text-fg-muted m-0 text-xs font-normal tracking-label uppercase">${escape(title)}</h4>
<ul class="text-fg-soft m-0 flex list-disc flex-col gap-1.5 pl-5 text-sm leading-normal">${lis.join('')}</ul>`
}

function renderRelease({ tag, date, sections }, base) {
  const [year, month, day] = date.split('-').map(Number)
  const shown = `${day} ${MONTHS[month - 1]} ${year}`
  const body = sections.length
    ? sections.map((s) => renderSection(s, base)).join('\n')
    : `<p class="${prose}">Maintenance only, nothing that changes the app.</p>`
  return `<section class="flex flex-col gap-3">
<div class="flex items-baseline justify-between gap-4">
<h3 class="font-display text-fg m-0 text-lg font-semibold"><a class="text-fg no-underline underline-offset-2 hover:underline" href="${escape(`${base}/releases/tag/${tag}`)}">${escape(tag)}</a></h3>
<time class="text-fg-muted font-mono text-xs" datetime="${escape(date)}">${escape(shown)}</time>
</div>
${body}
</section>`
}

export function renderChangelog(releases, base) {
  if (new URL(base).protocol !== 'https:') {
    throw new Error(`changelog: base must be an https URL, got ${JSON.stringify(base)}`)
  }
  base = base.replace(/\/+$/, '')
  if (releases.length === 0) {
    return `<p class="${prose}">The release notes are on <a class="${inline}" href="${escape(`${base}/releases`)}">GitHub</a>.</p>`
  }
  return `<div class="flex flex-col gap-8">\n${releases.map((r) => renderRelease(r, base)).join('\n')}\n</div>`
}
