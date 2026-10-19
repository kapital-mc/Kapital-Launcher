import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { parseChangelog, renderChangelog } from './changelog.js'

const BASE = 'https://github.com/sandrogekeler/Kapital-Launcher'
const feed = readFileSync(new URL('./test/fixtures/releases.atom', import.meta.url), 'utf8')

const escapeXml = (s) => s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')

// One feed entry around a release body, escaped the way GitHub escapes it.
const entry = (html, tag = 'v1.0.0-beta.9', updated = '2026-11-02T08:00:00Z') => `<entry>
  <updated>${updated}</updated>
  <link rel="alternate" type="text/html" href="${BASE}/releases/tag/${tag}"/>
  <title>${tag}</title>
  <content type="html">${escapeXml(html)}</content>
</entry>`

const pull = (n) => `<a href="${BASE}/pull/${n}">#${n}</a>`

test('the real feed gives four releases, newest first', () => {
  const releases = parseChangelog(feed, BASE)
  assert.deepEqual(
    releases.map((r) => [r.tag, r.date]),
    [
      ['v1.0.0-beta.4', '2026-10-18'],
      ['v1.0.0-beta.3', '2026-10-15'],
      ['v1.0.0-beta.2', '2026-10-12'],
      ['v1.0.0-beta.1', '2026-10-04'],
    ],
  )
})

test('beta.4 lists Features then Fixes, with their pull requests', () => {
  const [beta4] = parseChangelog(feed, BASE)
  assert.deepEqual(beta4.sections, [
    {
      title: 'Features',
      items: [
        { text: "Download the server's Distant Horizons terrain from the settings page", pr: 272 },
        { text: 'Open each chapter column in a popup with its full text', pr: 274 },
      ],
    },
    {
      title: 'Fixes',
      items: [
        { text: "Show Frangfurd's pack as 1.0.6 when its source cannot be read", pr: 271 },
        { text: 'Fix the memory slider sliver and say when Prism has no account by the profile name', pr: 275 },
      ],
    },
  ])
})

test('Maintenance and Documentation are dropped', () => {
  const releases = parseChangelog(feed, BASE)
  for (const release of releases) {
    assert.ok(release.sections.every((s) => ['Features', 'Fixes', 'Other changes'].includes(s.title)))
  }
  const beta3 = releases.find((r) => r.tag === 'v1.0.0-beta.3')
  assert.deepEqual(
    beta3.sections.map((s) => s.title),
    ['Features', 'Fixes'],
  )
  const prs = beta3.sections.flatMap((s) => s.items.map((i) => i.pr))
  assert.ok(!prs.includes(240) && !prs.includes(265), 'a Maintenance pull request leaked in')
  assert.ok(!JSON.stringify(releases).includes('Bump '))
})

test('no attribution survives in any item', () => {
  for (const release of parseChangelog(feed, BASE)) {
    for (const { items } of release.sections) {
      for (const item of items) assert.doesNotMatch(item.text, /@|\bby\s*$| in #/)
    }
  }
})

test('entities are decoded once and the attribution is stripped', () => {
  const html = `<h2>What's Changed</h2>
<h3>Fixes</h3>
<ul>
<li>Fix &lt;b&gt; &amp;amp; &quot;quotes&quot; &#39;x&#39; &#x27;y&#x27; &#8211; <code>go</code> by <a href="https://github.com/me">@me</a> in ${pull(7)}</li>
<li>Bump a from 1 to 2 by <a href="https://github.com/dependabot">@dependabot</a>[bot] in ${pull(8)}</li>
<li>Raw link by @me in ${BASE}/pull/9</li>
<li>No pull request here by <a href="https://github.com/me">@me</a> in <a href="https://example.com/pull/10">#10</a></li>
</ul>`
  const [release] = parseChangelog(entry(html), BASE)
  assert.deepEqual(release.sections[0].items, [
    { text: 'Fix <b> &amp; "quotes" \'x\' \'y\' – go', pr: 7 },
    { text: 'Bump a from 1 to 2', pr: 8 },
    { text: 'Raw link', pr: null },
    { text: 'No pull request here', pr: null },
  ])
})

test('a lowercase heading, a wrapped heading and a following h2 are all handled', () => {
  const html = `<p>intro</p>
<h2>Windows 11</h2><ul><li>not a change by @me in ${pull(1)}</li></ul>
<div class="markdown-heading"><h2 class="heading-element">What's changed</h2><a id="user-content-x" class="anchor" aria-label="Permalink" href="#x"></a></div>
<h3 class="x">Other changes</h3>
<ul><li class="y">Tidy the thing by @me in ${pull(2)}</li></ul>
<p class="z"><strong>Full Changelog</strong>: <a href="${BASE}/compare/a...b"><tt>a...b</tt></a></p>
<h2>New Contributors</h2><ul><li>@x made their first contribution in ${pull(3)}</li></ul>`
  const [release] = parseChangelog(entry(html), BASE)
  assert.deepEqual(release.sections, [{ title: 'Other changes', items: [{ text: 'Tidy the thing', pr: 2 }] }])
})

test('a release with nothing left keeps an empty list, and a foreign tag is skipped', () => {
  const only = `<h2>What's Changed</h2><h3>Maintenance</h3><ul><li>Bump x by @me in ${pull(5)}</li></ul>`
  const releases = parseChangelog(entry(only) + entry(only, 'nightly') + entry('<p>no heading</p>', 'v2.0.0'), BASE)
  assert.deepEqual(releases, [
    { tag: 'v1.0.0-beta.9', date: '2026-11-02', sections: [] },
    { tag: 'v2.0.0', date: '2026-11-02', sections: [] },
  ])
})

test('renderChangelog writes a release with links, a date and its sections', () => {
  const html = renderChangelog(parseChangelog(feed, BASE).slice(0, 1), BASE)
  assert.match(html, new RegExp(`<a [^>]*href="${BASE}/releases/tag/v1.0.0-beta.4">v1.0.0-beta.4</a>`))
  assert.match(html, /<time [^>]*datetime="2026-10-18">18 October 2026<\/time>/)
  assert.match(html, new RegExp(`href="${BASE}/pull/272">#272</a>`))
  assert.ok(html.indexOf('>Features<') < html.indexOf('>Fixes<'))
  assert.doesNotMatch(html, /style=|\u2014/)
})

test('renderChangelog escapes every value', () => {
  const evil = [
    {
      tag: 'v1.0.0-beta.1',
      date: '2026-10-04',
      sections: [{ title: 'Fixes', items: [{ text: '<script>alert("x")</script> & \'y\'', pr: 3 }] }],
    },
  ]
  const html = renderChangelog(evil, BASE)
  assert.ok(!html.includes('<script>'))
  assert.ok(html.includes('&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt; &amp; &#39;y&#39;'))
})

test('renderChangelog refuses a base that is not https', () => {
  assert.throws(() => renderChangelog([], 'http://github.com/a/b'), /https/)
  assert.throws(() => renderChangelog([], 'not a url'))
})

test('renderChangelog says so when a release changed nothing for the player', () => {
  const html = renderChangelog([{ tag: 'v1.0.0-beta.9', date: '2026-11-02', sections: [] }], BASE)
  assert.match(html, /Maintenance only, nothing that changes the app\./)
  assert.match(html, /2 November 2026/)
})

test('renderChangelog falls back to a link to the releases when there are none', () => {
  const html = renderChangelog([], `${BASE}/`)
  assert.equal(html.match(/<p /g).length, 1)
  assert.match(html, new RegExp(`href="${BASE}/releases">GitHub</a>`))
  assert.match(html, /The release notes are on/)
})
