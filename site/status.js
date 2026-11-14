// The server status the download site shows, worked out on Cloudflare by
// functions/api/status.js: the Java Edition Server List Ping, as the
// launcher's backend/services/ping.go does it, and the checks of the services
// around Frangfurd. Kept here without Cloudflare's socket so `pnpm test` can
// cover it.
//
// Only Frangfurd is asked: its two playit.gg addresses, its BlueMap, its
// Distant Horizons terrain download and its pack, each as data/launcher.json
// writes it (status.test.js holds them to it). Luxemburg and Lichdenstein are
// offline and are never asked. The handshake asks with protocol 0, one byte,
// which the Frangfurd server answers (ping.go has the history). Nothing a
// service says is passed on but whether it answered, the player counts as
// numbers, the pack's version and the terrain snapshot's date, each checked.

// The Tailscale address is left out: it is a private address that Cloudflare
// cannot reach.
export const SERVERS = [
  { id: 'global', chapter: 'frangfurd', label: 'Global', address: 'female-specified.gl.joinmc.link' },
  { id: 'germany', chapter: 'frangfurd', label: 'Germany', address: 'rails-enjoyed.tun.ply.gg' },
]
// The manifest's own values, as for the addresses above.
// aislop-ignore-next-line ai-slop/hardcoded-url -- the manifest's map, held to it by status.test.js
export const MAP = 'http://spiral-reminders.tun.ply.gg:1111'
export const TERRAIN = 'https://dh.frangfurd.click/dh/latest.json'
// aislop-ignore-next-line ai-slop/hardcoded-url -- the manifest's pack, held to it by status.test.js
export const PACK = 'https://kapital-packs.alessandrogekeler.workers.dev/frangfurd/pack.toml'

export const DEFAULT_PORT = 25565
// The longest status response read, as in ping.go: a favicon is about 10 KB.
export const MAX_RESPONSE = 1 << 20

export function varint(value) {
  const out = []
  let v = value >>> 0
  do {
    let byte = v & 0x7f
    v >>>= 7
    if (v !== 0) byte |= 0x80
    out.push(byte)
  } while (v !== 0)
  return out
}

function packet(id, body) {
  const payload = [...varint(id), ...body]
  return [...varint(payload.length), ...payload]
}

// Handshake (next state 1, status) then an empty Status Request, as one write.
export function request(host, port) {
  const name = [...new TextEncoder().encode(host)]
  const handshake = packet(0x00, [...varint(0), ...varint(name.length), ...name, (port >> 8) & 0xff, port & 0xff, 1])
  return Uint8Array.from([...handshake, ...packet(0x00, [])])
}

// Reads a varint at offset: [value, next offset], or null while incomplete.
function readVarint(bytes, offset) {
  let value = 0
  for (let i = 0; i < 5; i++) {
    if (offset + i >= bytes.length) return null
    const byte = bytes[offset + i]
    value |= (byte & 0x7f) << (7 * i)
    if ((byte & 0x80) === 0) return [value >>> 0, offset + i + 1]
  }
  throw new Error('varint too long')
}

// The Status Response's JSON text once all of it has arrived, else null.
export function responseText(bytes) {
  const length = readVarint(bytes, 0)
  if (!length) return null
  const [size, start] = length
  if (size > MAX_RESPONSE) throw new Error('response too large')
  if (bytes.length < start + size) return null
  const id = readVarint(bytes, start)
  if (!id || id[0] !== 0x00) throw new Error('unexpected packet')
  const text = readVarint(bytes, id[1])
  if (!text || text[1] + text[0] > start + size) throw new Error('malformed response')
  return new TextDecoder().decode(bytes.subarray(text[1], text[1] + text[0]))
}

// What the page is told about an answer: that it came, and the counts.
export function summarize(text) {
  const json = JSON.parse(text)
  const count = (n) => (Number.isInteger(n) && n >= 0 && n < 1e6 ? n : null)
  return { online: true, players: count(json?.players?.online), max: count(json?.players?.max) }
}

// The target and port of a _minecraft._tcp SRV answer from DNS over HTTPS
// (application/dns-json), or null. As in ping.go, a target that is not a
// plain host name is not used.
export function srvTarget(dns) {
  const answer = dns?.Answer?.find((record) => record.type === 33)
  const parts = typeof answer?.data === 'string' ? answer.data.trim().split(/\s+/) : []
  if (parts.length !== 4) return null
  const port = Number(parts[2])
  const host = parts[3].replace(/\.$/, '')
  if (!Number.isInteger(port) || port < 1 || port > 65535) return null
  if (!/^[a-z0-9-]+(\.[a-z0-9-]+)+$/i.test(host)) return null
  return { host, port }
}

// The request line of an HTTP/1.1 GET for a URL's root, for a service the
// function reaches over a plain socket (the map, on its own port).
export function httpRequest(url) {
  const { host } = new URL(url)
  return new TextEncoder().encode(
    `GET / HTTP/1.1\r\nHost: ${host}\r\nUser-Agent: kapital-launcher-site\r\nConnection: close\r\n\r\n`,
  )
}

// The status code of an HTTP response once its first line has arrived, else
// null. Anything that is not an HTTP status line is refused.
export function httpStatus(bytes) {
  const head = new TextDecoder().decode(bytes.subarray(0, 64))
  const end = head.indexOf('\r\n')
  if (end === -1) {
    if (bytes.length >= 64) throw new Error('not an HTTP response')
    return null
  }
  const match = /^HTTP\/1\.[01] (\d{3})/.exec(head.slice(0, end))
  if (!match) throw new Error('not an HTTP response')
  return Number(match[1])
}

// Whether an HTTP status says the service is there.
export const answered = (status) => Number.isInteger(status) && status >= 200 && status < 400

// The version a packwiz pack.toml names, or null.
export function packVersion(toml) {
  const match = /^version\s*=\s*"([^"\n]*)"/m.exec(typeof toml === 'string' ? toml : '')
  return match && /^[0-9A-Za-z.+-]{1,32}$/.test(match[1]) ? match[1] : null
}

// The day the terrain snapshot was written, from its latest.json, or null.
export function snapshotDate(json) {
  const created = json?.createdAt
  if (typeof created !== 'string' || !/^\d{4}-\d{2}-\d{2}T[\d:.]+Z$/.test(created)) return null
  const date = new Date(created)
  return Number.isNaN(date.getTime()) ? null : created.slice(0, 10)
}

// The body /api/status answers with. Each chapter is online when one of its
// addresses answered, with that address's counts, for the home page's server
// card; the rest is for the status page.
export function body({ servers = {}, map = null, terrain = null, pack = null, checked = new Date() } = {}) {
  const off = { online: false }
  const each = Object.fromEntries(SERVERS.map((s) => [s.id, servers[s.id] ?? off]))
  const live = Object.values(each).find((s) => s.online) ?? off
  return {
    checked: checked.toISOString(),
    luxemburg: off,
    lichdenstein: off,
    frangfurd: live,
    servers: each,
    map: map ?? off,
    terrain: terrain ?? off,
    pack: pack ?? off,
  }
}
