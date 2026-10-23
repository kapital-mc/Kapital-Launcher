// GET /status: whether the chapters' servers are up, for the page's server
// card (src/cards.js). A Cloudflare Pages Function, so the page asks only its
// own origin and the visitor's browser never meets the server. The ping is
// site/status.js's; this file adds Cloudflare's socket, DNS over HTTPS for
// the tunnel's SRV record, a time limit and a 30 s cache, so the server is
// asked at most once per 30 s per Cloudflare location, however many visit.
// aislop-ignore-next-line ai-slop/hallucinated-import -- the Workers runtime's own module, not a package
import { connect } from 'cloudflare:sockets'
import { DEFAULT_PORT, LIVE, MAX_RESPONSE, body, request, responseText, srvTarget, summarize } from '../status.js'

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

async function ping(address) {
  const { host, port } = await target(address)
  const socket = connect({ hostname: host, port })
  try {
    const writer = socket.writable.getWriter()
    await writer.write(request(address, port))
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
      const text = responseText(bytes)
      if (text !== null) return summarize(text)
    }
  } finally {
    socket.close().catch(() => {})
  }
}

function withTimeout(promise) {
  let timer
  const limit = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error('timed out')), TIMEOUT)
  })
  return Promise.race([promise, limit]).finally(() => clearTimeout(timer))
}

export async function onRequestGet({ request: incoming, waitUntil }) {
  const key = new Request(new URL('/status', incoming.url).toString())
  const cache = typeof caches === 'undefined' ? null : caches.default
  const hit = await cache?.match(key)
  if (hit) return hit

  const live = await withTimeout(ping(LIVE.address)).catch(() => null)
  const response = new Response(JSON.stringify(body(live)), {
    headers: {
      'content-type': 'application/json; charset=utf-8',
      'cache-control': `public, max-age=${MAX_AGE}`,
      'x-content-type-options': 'nosniff',
    },
  })
  if (cache) waitUntil(cache.put(key, response.clone()))
  return response
}
