import { describe, expect, it } from 'vitest'
import type { Contraption } from '../types'
import { contraptionDetail, contraptionKey, materialName } from './contraptions'

const bus: Contraption = {
  kind: 'toolgun',
  file: 'Bus.excraft',
  name: 'Bus',
  bytes: 10616,
  modified: Date.UTC(2026, 9, 27, 12),
  blocks: 609,
  blockTypes: 31,
  subLevels: 1,
  materials: [],
  items: [],
  unread: false,
}

describe('contraptionDetail', () => {
  it('states blocks, sub-levels, size on disk and the date', () => {
    expect(contraptionDetail(bus, 'en-GB')).toBe('609 blocks · 1 sub-level · 10 KB · 27 Oct 2026')
  })
  it('gives a schematic its box and leaves out what the file does not say', () => {
    const s = { ...bus, kind: 'schematic', blocks: -1, subLevels: 0, size: [52, 14, 38] }
    expect(contraptionDetail(s, 'en-GB')).toBe('52 × 14 × 38 · 10 KB · 27 Oct 2026')
  })
})

describe('materialName', () => {
  it("names Minecraft's blocks bare and another mod's with its namespace", () => {
    expect(materialName('minecraft:oak_planks')).toEqual({ name: 'Oak planks', mod: null })
    expect(materialName('create:copycat_slab')).toEqual({ name: 'Copycat slab', mod: 'create' })
    expect(materialName('stone')).toEqual({ name: 'Stone', mod: null })
  })
})

it('keys a row by kind and file', () => {
  expect(contraptionKey(bus)).toBe('toolgun/Bus.excraft')
})
