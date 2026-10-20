import { describe, expect, it } from 'vitest'
import { memoryTone } from './chapterSettings'

describe('memoryTone', () => {
  const gb = (n: number) => n * 1024
  it('is fine within a gigabyte of the recommendation and trouble four away', () => {
    expect(memoryTone(gb(9), gb(9), 0)).toBe(0)
    expect(memoryTone(gb(8), gb(9), 0)).toBe(0)
    expect(memoryTone(gb(10), gb(9), 0)).toBe(0)
    expect(memoryTone(gb(6.5), gb(9), 0)).toBe(1)
    expect(memoryTone(gb(11.5), gb(9), 0)).toBe(1)
    expect(memoryTone(gb(5), gb(9), 0)).toBe(2)
    expect(memoryTone(gb(16), gb(9), 0)).toBe(2)
  })
  it('wants six gigabytes left on the machine, and three is trouble', () => {
    expect(memoryTone(gb(10), 0, gb(16))).toBe(0)
    expect(memoryTone(gb(11.5), 0, gb(16))).toBe(1)
    expect(memoryTone(gb(13), 0, gb(16))).toBe(2)
    expect(memoryTone(gb(9), gb(9), gb(12))).toBe(2)
  })
  it('says nothing with nothing to measure against', () => {
    expect(memoryTone(gb(4), 0, 0)).toBe(0)
  })
})
