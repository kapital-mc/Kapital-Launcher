import { describe, expect, it } from 'vitest'
import type { StorageItem } from '../types'
import {
  canDelete,
  confirmText,
  serverName,
  storageDetail,
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
    expect(storageTitle(old)).toBe('Distant Horizons terrain: Frangfurd (Germany)')
    expect(serverName(item({ kind: 'terrain', name: 'raw' }))).toBe('raw')
    expect(storageKey(old)).toBe('terrain/Frangfurd+%28Germany%29')
  })

  it('offers a delete only for what Go deletes', () => {
    const kinds = ['terrain', 'terrainDownload', 'worldMap', 'screenshots', 'logs', 'mods', 'other']
    expect(kinds.filter((kind) => canDelete(item({ kind })))).toEqual([
      'terrain',
      'terrainDownload',
      'worldMap',
      'logs',
    ])
  })

  it('says each row’s size and what to know about it', () => {
    expect(storageDetail(item({ kind: 'screenshots', files: 1 }))).toBe('3.0 MB · 1 picture')
    expect(storageDetail(item({ kind: 'logs' }))).toBe('3.0 MB · 2 files')
    expect(storageDetail(item({ kind: 'mods' }))).toBe(
      "3.0 MB · the pack's mods, put back on every Play",
    )
    expect(
      storageDetail(item({ kind: 'terrain', lastModified: '2026-10-20T19:40:00Z' }), 'en-GB'),
    ).toBe('3.0 MB · last written 20 Oct 2026')
    expect(storageDetail(item({ kind: 'terrain' }))).toBe('3.0 MB')
    expect(storageDetail(item({ kind: 'saves' }))).toBe('3.0 MB')
    expect(storageTitle(item({ kind: 'saves' }))).toBe('saves')
  })

  it('asks before a delete in words that say what is lost', () => {
    expect(confirmText(item({ kind: 'terrain', name: '1_frangfurd' }))).toBe(
      'Delete 3.0 MB of terrain for frangfurd? It comes back from the server as you play.',
    )
    expect(confirmText(item({ kind: 'worldMap' }))).toBe(
      'Delete the world map (3.0 MB)? What you explored is lost.',
    )
    expect(confirmText(item({ kind: 'logs' }))).toBe(
      'Delete the logs and crash reports (2 files, 3.0 MB)?',
    )
    expect(confirmText(item({ kind: 'terrainDownload' }))).toBe(
      'Delete the unfinished download (3.0 MB)? The next one starts over.',
    )
  })
})
