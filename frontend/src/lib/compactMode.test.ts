import { describe, expect, it } from 'vitest'
import { holdsCompact } from './compactMode'

const floor = { width: 880, height: 700, hysteresis: 40 }

describe('holdsCompact', () => {
  it('holds compact below the floor on either side', () => {
    expect(holdsCompact({ width: 879, height: 800 }, false, floor)).toBe(true)
    expect(holdsCompact({ width: 1280, height: 699 }, false, floor)).toBe(true)
    expect(holdsCompact({ width: 1280, height: 800 }, false, floor)).toBe(false)
  })

  it('lets go only once the window is the hysteresis past both sides', () => {
    expect(holdsCompact({ width: 900, height: 800 }, true, floor)).toBe(true)
    expect(holdsCompact({ width: 1280, height: 720 }, true, floor)).toBe(true)
    expect(holdsCompact({ width: 920, height: 740 }, true, floor)).toBe(false)
    // Not held before, the same size stays full: the line does not flip the layout.
    expect(holdsCompact({ width: 900, height: 720 }, false, floor)).toBe(false)
  })
})
