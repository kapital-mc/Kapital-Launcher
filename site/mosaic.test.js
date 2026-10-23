import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  BANNER_COLS,
  BANNER_DELAYS,
  BANNER_ROWS,
  COLS,
  DELAYS,
  COVERED,
  ROWS,
  SUBJECTS,
  backdropSquares,
  bannerSquares,
  renderBackdropArt,
  renderBackdropPixels,
  renderBanner,
  renderBannerArt,
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

test('the banner has one square per cell in both of its grids', () => {
  const squares = bannerSquares()
  assert.equal(squares.length, BANNER_COLS * BANNER_ROWS)
  assert.equal(renderBannerArt(squares).match(/<span /g).length, BANNER_COLS * BANNER_ROWS)
  assert.equal(renderBanner(squares).match(/<span /g).length, BANNER_COLS * BANNER_ROWS)
})

test("each chapter's banner keeps its subject clear and the tag and close button dark", () => {
  const squares = bannerSquares()
  const at = (x, y) => squares[y * BANNER_COLS + x]
  SUBJECTS.forEach(({ columns, squares: clear }, i) => {
    const chapter = `0${i + 1}`
    for (let y = 0; y < BANNER_ROWS; y++) {
      for (let x = columns[0]; x <= columns[1]; x++) {
        if (COVERED.some(([cx, cy]) => cx === x && cy === y)) continue
        assert.ok(!at(x, y).m.includes(chapter), `${chapter} dark at ${x},${y}`)
      }
    }
    for (const [x, y] of clear) assert.ok(!at(x, y).m.includes(chapter))
    for (const [x, y] of COVERED) assert.ok(at(x, y).m.includes(chapter))
  })
  const pattern = (n) => squares.map((s) => s.m.includes(n)).join()
  assert.notEqual(pattern('01'), pattern('02'))
  assert.notEqual(pattern('02'), pattern('03'))
})

test('the banner changes in a wave from right to left', () => {
  const squares = bannerSquares()
  for (const { d } of squares) assert.ok(Number.isInteger(d) && d >= 0 && d < BANNER_DELAYS)
  const column = (x) => {
    const ds = squares.filter((_, i) => i % BANNER_COLS === x).map((s) => s.d)
    return ds.reduce((a, b) => a + b, 0) / ds.length
  }
  for (let x = 1; x < BANNER_COLS; x++) assert.ok(column(x) <= column(x - 1), `column ${x}`)
})
