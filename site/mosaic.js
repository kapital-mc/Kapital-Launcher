// The squares of the chapter backdrop and of the download popup's banner,
// written into index.html when the site is built (vite.config.js), so the
// page carries no script for them and no inline style: every square is an
// empty span, and src/showcase.css places it, paints it and times it.
//
// The backdrop is two grids of the same size over the same area: the picture,
// one slice per square, and the dark squares on top. Each square carries a
// delay bucket (data-d), so a chapter switch replaces the picture square by
// square, and the chapters whose dark squares it is part of (data-m, the
// chapters' numbers). The pattern is seeded, so every build writes the same
// page.

// The grid, in squares. src/showcase.css repeats these numbers in its
// grid-template and its background-position: change both together.
export const COLS = 24
export const ROWS = 12
export const BANNER_COLS = 13
export const BANNER_ROWS = 4

// The number of delay buckets src/showcase.css spreads over the switch.
export const DELAYS = 8

// The chapters' numbers, as data/launcher.json writes them.
const CHAPTERS = ['01', '02', '03']

// A linear congruential generator: small, seeded and the same everywhere.
function random(seed) {
  let s = seed
  return () => {
    s = (s * 9301 + 49297) % 233280
    return s / 233280
  }
}

// How likely the square at (x, y) is dark: certain along the top, thinning to
// nothing a little above the middle and thickening again towards the bottom,
// and a little more likely towards the sides, so the picture shows through a
// ragged window rather than a rectangle.
function darkness(x, y) {
  const v = y < ROWS / 2 ? 1 - y / (ROWS * 0.42) : (y - ROWS * 0.55) / (ROWS * 0.45)
  const edge = Math.abs(x - (COLS - 1) / 2) / ((COLS - 1) / 2)
  return Math.max(v, 0) + edge * edge * 0.25
}

// Each square of the backdrop: its delay bucket and the chapters it is dark in.
export function backdropSquares() {
  const delay = random(7919)
  const dark = CHAPTERS.map((_, i) => random(104729 * (i + 1) + 3))
  const squares = []
  for (let y = 0; y < ROWS; y++) {
    for (let x = 0; x < COLS; x++) {
      const p = darkness(x, y)
      squares.push({
        d: Math.floor(delay() * DELAYS),
        m: CHAPTERS.filter((_, i) => dark[i]() < p),
      })
    }
  }
  return squares
}

// The banner's dark squares, one pattern: dark on the left, clearing towards
// the right, with the bottom row darker so the heading below meets the ground.
export function bannerSquares() {
  const rnd = random(13)
  const squares = []
  for (let y = 0; y < BANNER_ROWS; y++) {
    for (let x = 0; x < BANNER_COLS; x++) {
      const p = 1 - (x + 0.5) / (BANNER_COLS * 0.7) + (y === BANNER_ROWS - 1 ? 0.35 : 0)
      squares.push(rnd() < p)
    }
  }
  return squares
}

export function renderBackdropArt(squares) {
  return squares.map((s) => `<span data-d="${s.d}"></span>`).join('')
}

export function renderBackdropPixels(squares) {
  return squares
    .map((s) => (s.m.length ? `<span data-d="${s.d}" data-m="${s.m.join(' ')}"></span>` : `<span data-d="${s.d}"></span>`))
    .join('')
}

export function renderBanner(squares) {
  return squares.map((dark) => (dark ? '<span data-dark></span>' : '<span></span>')).join('')
}
