import { describe, expect, it } from 'vitest'
import type { StorageItem } from '../types'
import {
  canDelete,
  confirmText,
  hasRows,
  serverName,
  storageDetail,
  storageGroups,
  storageKey,
  storageTitle,
} from './storage'

const item = (over: Partial<StorageItem>): StorageItem => ({
  kind: 'other',
  name: '',
  label: '',
  bytes: 1024 * 1024 * 3,
  files: 2,
  lastModified: '',
  ...over,
})

describe('storage rows', () => {
  it('names a terrain folder by its server', () => {
    const current = item({
      kind: 'terrain',
      name: '1790374988_frangfurd',
      label: '1790374988_frangfurd',
    })
    const old = item({
      kind: 'terrain',
      name: 'Frangfurd+%28Germany%29',
      label: 'Frangfurd (Germany)',
    })
    expect(serverName(current)).toBe('frangfurd')
    expect(serverName(old)).toBe('Frangfurd (Germany)')
    expect(storageTitle(old)).toBe('Frangfurd (Germany)')
    expect(serverName(item({ kind: 'terrain', name: 'raw' }))).toBe('raw')
    expect(storageKey(old)).toBe('terrain/Frangfurd+%28Germany%29')
  })

  it('offers a delete only for what Go deletes', () => {
    const kinds = [
      'terrain',
      'terrainDownload',
      'worldMap',
      'minimap',
      'screenshots',
      'logs',
      'mods',
      'other',
    ]
    expect(kinds.filter((kind) => canDelete(item({ kind })))).toEqual([
      'terrain',
      'terrainDownload',
      'worldMap',
      'screenshots',
      'logs',
    ])
  })

  it('gathers a mod’s rows into one group, in a fixed order, and leaves out empty groups', () => {
    const groups = storageGroups([
      item({ kind: 'terrain', name: 'a', bytes: 5 }),
      item({ kind: 'terrain', name: 'b', bytes: 3 }),
      item({ kind: 'terrainDownload', bytes: 2 }),
      item({ kind: 'minimap', bytes: 1 }),
      item({ kind: 'mods', bytes: 7 }),
      item({ kind: 'saves', bytes: 4 }),
    ])
    expect(groups.map((g) => [g.id, g.bytes, g.items.length])).toEqual([
      ['terrain', 10, 3],
      ['map', 1, 1],
      ['mods', 7, 1],
      ['other', 4, 1],
    ])
    expect(groups.map(hasRows)).toEqual([true, true, false, false])
    expect(groups[0]!.title).toBe('Distant Horizons')
    expect(groups[1]!.title).toBe("Xaero's maps")
  })

  it('says each row’s size and what to know about it', () => {
    expect(storageDetail(item({ kind: 'screenshots', files: 1 }))).toBe('3.0 MB · 1 picture')
    expect(storageDetail(item({ kind: 'logs' }))).toBe('3.0 MB · 2 files')
    expect(storageDetail(item({ kind: 'minimap' }))).toBe('3.0 MB · your waypoints')
    expect(storageDetail(item({ kind: 'mods' }))).toBe(
      "3.0 MB · the pack's mods, put back on every Play",
    )
    expect(
      storageDetail(item({ kind: 'terrain', lastModified: '2026-10-20T19:40:00Z' }), 'en-GB'),
    ).toBe('3.0 MB · last written 20 Oct 2026')
    expect(storageDetail(item({ kind: 'terrain' }))).toBe('3.0 MB')
    expect(storageDetail(item({ kind: 'saves' }))).toBe('3.0 MB')
    expect(storageTitle(item({ kind: 'saves' }))).toBe('saves')
    expect(storageTitle(item({ kind: 'minimap' }))).toBe('Minimap and waypoints')
  })

  it('asks before a delete in words that say what is lost', () => {
    expect(confirmText(item({ kind: 'terrain', name: '1_frangfurd' }))).toBe(
      'Delete 3.0 MB of terrain? It comes back from the server as you play.',
    )
    expect(confirmText(item({ kind: 'worldMap' }))).toBe(
      'Delete 3.0 MB? What you explored is lost.',
    )
    expect(confirmText(item({ kind: 'screenshots', files: 40 }))).toBe(
      'Delete 40 pictures (3.0 MB)? They do not go to the bin.',
    )
    expect(confirmText(item({ kind: 'logs' }))).toBe('Delete 2 files (3.0 MB)?')
    expect(confirmText(item({ kind: 'terrainDownload' }))).toBe(
      'Delete 3.0 MB? The next download starts over.',
    )
  })
})
