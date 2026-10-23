import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { LIVE, MAX_RESPONSE, body, request, responseText, srvTarget, summarize, varint } from './status.js'

const manifest = JSON.parse(readFileSync(new URL('../data/launcher.json', import.meta.url), 'utf8'))

const response = (json) => {
  const text = [...new TextEncoder().encode(json)]
  const payload = [...varint(0), ...varint(text.length), ...text]
  return Uint8Array.from([...varint(payload.length), ...payload])
}

test('the live address is the manifest’s', () => {
  const chapter = manifest.chapters.find((c) => c.id === LIVE.chapter)
  const address = chapter.server.addresses.find((a) => a.label === LIVE.label)
  assert.equal(address?.address, LIVE.address)
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
  const b = body({ online: true, players: 1, max: 2 })
  assert.deepEqual(b.luxemburg, { online: false })
  assert.deepEqual(b.lichdenstein, { online: false })
  assert.deepEqual(b.frangfurd, { online: true, players: 1, max: 2 })
  assert.deepEqual(body(null).frangfurd, { online: false })
})
