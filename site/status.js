// The server status the download page shows, worked out on Cloudflare by
// functions/status.js: the Java Edition Server List Ping, as the launcher's
// backend/services/ping.go does it, kept here without Cloudflare's socket so
// `pnpm test` can cover it.
//
// Only Frangfurd is asked, at the Germany address of data/launcher.json (its
// playit.gg tunnel); Luxemburg and Lichdenstein are offline and are never
// asked. The handshake asks with protocol 0, one byte, which the Frangfurd
// server answers (ping.go has the history). Nothing the server says is
// passed on but whether it answered and its player counts, as numbers.

// The chapter asked and the address asked, as data/launcher.json writes it.
export const LIVE = { chapter: 'frangfurd', label: 'Germany', address: 'rails-enjoyed.tun.ply.gg' }
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

// The body /status answers with: every chapter, the live one as asked.
export function body(live) {
  return {
    luxemburg: { online: false },
    lichdenstein: { online: false },
    [LIVE.chapter]: live ?? { online: false },
  }
}
