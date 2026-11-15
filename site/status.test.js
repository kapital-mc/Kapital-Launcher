import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import {
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
  varint,
} from './status.js'

const manifest = JSON.parse(readFileSync(new URL('../data/launcher.json', import.meta.url), 'utf8'))

const response = (json) => {
  const text = [...new TextEncoder().encode(json)]
  const payload = [...varint(0), ...varint(text.length), ...text]
  return Uint8Array.from([...varint(payload.length), ...payload])
}

test('the addresses asked are the manifest’s', () => {
  for (const server of SERVERS) {
    const chapter = manifest.chapters.find((c) => c.id === server.chapter)
    const address = chapter.server.addresses.find((a) => a.label === server.label)
    assert.equal(address?.address, server.address, server.label)
  }
  const frangfurd = manifest.chapters.find((c) => c.id === 'frangfurd')
  assert.equal(frangfurd.map, MAP)
  assert.equal(frangfurd.pack.packwiz, PACK)
  assert.equal(frangfurd.pack.dhSnapshot, TERRAIN)
})

test('varints are the protocol’s', () => {
  assert.deepEqual(varint(0), [0x00])
  assert.deepEqual(varint(127), [0x7f])
  assert.deepEqual(varint(128), [0x80, 0x01])
  assert.deepEqual(varint(25565), [0xdd, 0xc7, 0x01])
})

test('the request is a status handshake with protocol 0, then a status request', () => {
  const bytes = [...request('a.b', 25565)]
  // length, id 0, protocol 0, name, port, next state 1; then length 1, id 0
  assert.deepEqual(bytes, [9, 0x00, 0x00, 3, 97, 46, 98, 0x63, 0xdd, 1, 1, 0x00])
})

test('a response is read once all of it has arrived', () => {
  const bytes = response('{"players":{"online":3,"max":20}}')
  assert.equal(responseText(bytes.subarray(0, 5)), null)
  assert.deepEqual(summarize(responseText(bytes)), { online: true, players: 3, max: 20 })
})

test('a response too large or of another packet is refused', () => {
  assert.throws(() => responseText(Uint8Array.from(varint(MAX_RESPONSE + 1))))
  assert.throws(() => responseText(Uint8Array.from([2, 0x01, 0x00])))
})

test('only the counts are passed on, and only as numbers', () => {
  assert.deepEqual(summarize('{"players":{"online":"3","max":-1},"description":"hi"}'), {
    online: true,
    players: null,
    max: null,
  })
})

test('an SRV answer names a plain host and a port', () => {
  const dns = (data) => ({ Answer: [{ type: 33, data }] })
  assert.deepEqual(srvTarget(dns('1 1 56932 rails-enjoyed.tun.ply.gg.')), { host: 'rails-enjoyed.tun.ply.gg', port: 56932 })
  assert.equal(srvTarget(dns('1 1 0 a.b')), null)
  assert.equal(srvTarget(dns('1 1 25565 127.0.0.1:80/x')), null)
  assert.equal(srvTarget({}), null)
})

test('Luxemburg and Lichdenstein are always offline', () => {
  const b = body({ servers: { germany: { online: true, players: 1, max: 2 } } })
  assert.deepEqual(b.luxemburg, { online: false })
  assert.deepEqual(b.lichdenstein, { online: false })
})

test('Frangfurd is online when one of its addresses answered, with its counts', () => {
  const up = { online: true, players: 1, max: 2 }
  assert.deepEqual(body({ servers: { germany: up } }).frangfurd, up)
  assert.deepEqual(body({ servers: { global: null, germany: up } }).servers.global, { online: false })
  assert.deepEqual(body().frangfurd, { online: false })
  assert.deepEqual(body().map, { online: false })
  assert.deepEqual(Object.keys(body().servers), SERVERS.map((s) => s.id))
  assert.equal(body({ checked: new Date(0) }).checked, '1970-01-01T00:00:00.000Z')
})

test('the map is asked with a plain GET for its root', () => {
  const text = new TextDecoder().decode(httpRequest('http://a.b:1111'))
  assert.match(text, /^GET \/ HTTP\/1\.1\r\nHost: a\.b:1111\r\n/)
  assert.ok(text.endsWith('\r\n\r\n'))
})

test('an HTTP status is read from the first line only', () => {
  const bytes = (text) => new TextEncoder().encode(text)
  assert.equal(httpStatus(bytes('HTTP/1.1 200 OK\r\nContent-Type: text/html')), 200)
  assert.equal(httpStatus(bytes('HTTP/1.1 20')), null)
  assert.throws(() => httpStatus(bytes('SSH-2.0-OpenSSH\r\n')))
  assert.throws(() => httpStatus(bytes('x'.repeat(80))))
  assert.ok(answered(200) && answered(302))
  assert.ok(!answered(404) && !answered(502) && !answered(null))
})

test('the pack version is read from pack.toml and checked', () => {
  assert.equal(packVersion('name = "Frangfurd"\nversion = "1.0.16"\n'), '1.0.16')
  assert.equal(packVersion('version = "<script>"'), null)
  assert.equal(packVersion('[versions]\nminecraft = "1.21.1"'), null)
  assert.equal(packVersion(null), null)
})

test('the terrain snapshot gives its day', () => {
  assert.equal(snapshotDate({ createdAt: '2026-10-25T03:05:15Z' }), '2026-10-25')
  assert.equal(snapshotDate({ createdAt: 'yesterday' }), null)
  assert.equal(snapshotDate({ createdAt: '2026-99-99T00:00:00Z' }), null)
  assert.equal(snapshotDate(null), null)
})
