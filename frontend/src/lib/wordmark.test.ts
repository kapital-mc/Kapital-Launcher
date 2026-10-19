import { describe, expect, it } from 'vitest'
import { WORDMARK, parseWordmark } from './wordmark'

describe('WORDMARK', () => {
  it('holds the outline and a face for each of the fifteen letters', () => {
    expect(WORDMARK.viewBox).toBe('0 0 800 219')
    expect(WORDMARK.outline).toHaveLength(1)
    expect(WORDMARK.faces).toHaveLength(15)
    for (const d of [...WORDMARK.outline, ...WORDMARK.faces]) expect(d).toMatch(/^M[\d .MZ]+Z$/)
  })

  it('reads only path data, whatever else the file holds', () => {
    const mark = parseWordmark(
      '<svg viewBox="0 0 10 10"><script>alert(1)</script><path data-part="face" onclick="x" d="M0 0 1 1Z"/></svg>',
    )
    expect(mark).toEqual({ viewBox: '0 0 10 10', outline: [], faces: ['M0 0 1 1Z'] })
  })
})
