// GET /api/status: whether Frangfurd's servers and the services around them
// are up, for the home page's server card (src/cards.js) and the status page
// (src/status-page.js). A Cloudflare Pages Function, so the pages ask only
// their own origin and the visitor's browser never meets the servers. The
// parsing is site/status.js's; this file adds Cloudflare's socket, DNS over
// HTTPS for the tunnels' SRV records, a time limit per check and a 30 s
// cache, so each service is asked at most once per 30 s per Cloudflare
// location, however many visit.
// aislop-ignore-next-line ai-slop/hallucinated-import -- the Workers runtime's own module, not a package
import { connect } from 'cloudflare:sockets'
import {
  DEFAULT_PORT,
  MAP,
  MAX_RESPONSE,
  PACK,
  SERVERS,
  TERRAIN,
  answered,
  body,
  httpRequest,
  httpStatus,
  packVersion,
  request,
  responseText,
  snapshotDate,
  srvTarget,
  summarize,
} from '../../status.js'

const TIMEOUT = 4000
const MAX_AGE = 30

async function target(host) {
  try {
    // aislop-ignore-next-line ai-slop/hardcoded-url -- fixed on purpose: Cloudflare's own resolver, the SRV lookup ping.go does
    const url = `https://cloudflare-dns.com/dns-query?name=_minecraft._tcp.${host}&type=SRV`
    const answer = await fetch(url, { headers: { accept: 'application/dns-json' } })
    if (answer.ok) return srvTarget(await answer.json()) ?? { host, port: DEFAULT_PORT }
  } catch {
    // No SRV record to be had: the host on the default port, as the game does.
  }
  return { host, port: DEFAULT_PORT }
}

// Writes `out` to a socket and reads until `parse` returns something.
async function exchange(hostname, port, out, parse) {
  const socket = connect({ hostname, port })
  try {
    const writer = socket.writable.getWriter()
    await writer.write(out)
    writer.releaseLock()
    const reader = socket.readable.getReader()
    let bytes = new Uint8Array(0)
    for (;;) {
      const { value, done } = await reader.read()
      if (done) throw new Error('closed before the response')
      const next = new Uint8Array(bytes.length + value.length)
      next.set(bytes)
      next.set(value, bytes.length)
      bytes = next
      if (bytes.length > MAX_RESPONSE + 16) throw new Error('response too large')
      const result = parse(bytes)
      if (result !== null) return result
    }
  } finally {
    socket.close().catch(() => {})
  }
}

async function ping(address) {
  const { host, port } = await target(address)
  const text = await exchange(host, port, request(address, port), responseText)
  return summarize(text)
}

// The map is plain HTTP on its own port, reached over a socket: fetch() keeps
// to port 80 under an older compatibility date.
async function map() {
  const url = new URL(MAP)
  const status = await exchange(url.hostname, Number(url.port) || 80, httpRequest(MAP), httpStatus)
  return { online: answered(status) }
}

async function terrain() {
  const response = await fetch(TERRAIN, { headers: { accept: 'application/json' } })
  if (!response.ok) return { online: false }
  return { online: true, date: snapshotDate(await response.json().catch(() => null)) }
}

async function pack() {
  const response = await fetch(PACK)
  if (!response.ok) return { online: false }
  return { online: true, version: packVersion(await response.text()) }
}

function withTimeout(promise) {
  let timer
  const limit = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error('timed out')), TIMEOUT)
  })
  return Promise.race([promise, limit]).finally(() => clearTimeout(timer))
}

const settle = (promise) => withTimeout(promise).catch(() => null)

export async function onRequestGet({ request: incoming, waitUntil }) {
  const key = new Request(new URL('/api/status', incoming.url).toString())
  const cache = typeof caches === 'undefined' ? null : caches.default
  const hit = await cache?.match(key)
  if (hit) return hit

  const [servers, mapState, terrainState, packState] = await Promise.all([
    Promise.all(SERVERS.map((s) => settle(ping(s.address)))),
    settle(map()),
    settle(terrain()),
    settle(pack()),
  ])
  const answer = body({
    servers: Object.fromEntries(SERVERS.map((s, i) => [s.id, servers[i]])),
    map: mapState,
    terrain: terrainState,
    pack: packState,
  })
  const response = new Response(JSON.stringify(answer), {
    headers: {
      'content-type': 'application/json; charset=utf-8',
      'cache-control': `public, max-age=${MAX_AGE}`,
      'x-content-type-options': 'nosniff',
    },
  })
  if (cache) waitUntil(cache.put(key, response.clone()))
  return response
}
