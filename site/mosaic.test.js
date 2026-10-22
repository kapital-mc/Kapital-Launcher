import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  BANNER_COLS,
  BANNER_ROWS,
  COLS,
  DELAYS,
  ROWS,
  backdropSquares,
  bannerSquares,
  renderBackdropArt,
  renderBackdropPixels,
  renderBanner,
} from './mosaic.js'

test('the backdrop has one square per cell of the grid', () => {
  const squares = backdropSquares()
  assert.equal(squares.length, COLS * ROWS)
  assert.equal(renderBackdropArt(squares).match(/<span /g).length, COLS * ROWS)
  assert.equal(renderBackdropPixels(squares).match(/<span /g).length, COLS * ROWS)
})

test('every build writes the same backdrop', () => {
  assert.deepEqual(backdropSquares(), backdropSquares())
  assert.deepEqual(bannerSquares(), bannerSquares())
})

test('every delay is a bucket the stylesheet times', () => {
  for (const { d } of backdropSquares()) assert.ok(Number.isInteger(d) && d >= 0 && d < DELAYS)
})

test('each chapter has a different dark pattern, dark along the top and open in the middle', () => {
  const squares = backdropSquares()
  const pattern = (n) => squares.map((s) => s.m.includes(n)).join()
  assert.notEqual(pattern('01'), pattern('02'))
  assert.notEqual(pattern('02'), pattern('03'))
  const row = (y) => squares.slice(y * COLS, (y + 1) * COLS)
  assert.ok(row(0).every((s) => s.m.length === 3))
  const middle = row(ROWS / 2).slice(COLS / 4, (COLS * 3) / 4)
  assert.ok(middle.filter((s) => s.m.length > 0).length < middle.length / 2)
})

test('the banner has one square per cell and is dark on the left', () => {
  const squares = bannerSquares()
  assert.equal(squares.length, BANNER_COLS * BANNER_ROWS)
  assert.equal(renderBanner(squares).match(/<span/g).length, BANNER_COLS * BANNER_ROWS)
  assert.ok(squares[0])
})
