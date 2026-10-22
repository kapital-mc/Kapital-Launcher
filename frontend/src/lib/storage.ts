import type { StorageItem } from '../types'
import { formatBytes } from './bytes'
import { publishedOn } from './terrain'

/** The kinds Go deletes on request (services.ClearStorage). */
const DELETABLE: readonly string[] = [
  'terrain',
  'terrainDownload',
  'worldMap',
  'saves',
  'screenshots',
  'logs',
]

export const canDelete = (item: StorageItem): boolean => DELETABLE.includes(item.kind)

/** The kinds whose confirmed delete asks once more in a popup: worlds, terrain, maps and pictures. */
const ASKS_TWICE: readonly string[] = [
  'terrain',
  'terrainDownload',
  'worldMap',
  'saves',
  'screenshots',
]

export const asksTwice = (item: StorageItem): boolean => ASKS_TWICE.includes(item.kind)

/** A world asks every time: "Don't ask me again" does not cover it. */
export const alwaysAsks = (item: StorageItem): boolean => item.kind === 'saves'

/** A row's key: its kind, and the server folder for terrain or the world folder for a save. */
export const storageKey = (item: StorageItem): string => `${item.kind}/${item.name}`

/** The groups the section shows, in order: a mod's files together, the worlds, then one kind each. */
export type StorageGroupId = 'terrain' | 'map' | 'saves' | 'screenshots' | 'logs' | 'mods' | 'other'

export interface StorageGroup {
  id: StorageGroupId
  title: string
  bytes: number
  items: StorageItem[]
}

const GROUP_OF: Record<string, StorageGroupId> = {
  terrain: 'terrain',
  terrainDownload: 'terrain',
  worldMap: 'map',
  minimap: 'map',
  saves: 'saves',
  screenshots: 'screenshots',
  logs: 'logs',
  mods: 'mods',
}

const GROUPS: readonly { id: StorageGroupId; title: string }[] = [
  { id: 'terrain', title: 'Distant Horizons' },
  { id: 'map', title: "Xaero's maps" },
  { id: 'saves', title: 'Saves' },
  { id: 'screenshots', title: 'Screenshots' },
  { id: 'logs', title: 'Logs and crash reports' },
  { id: 'mods', title: 'Mods' },
  { id: 'other', title: 'Everything else' },
]

/**
 * The colour of each group, in the split bar and on its rows. Written out whole so Tailwind
 * finds every class.
 */
export const GROUP_COLOUR: Record<StorageGroupId, string> = {
  terrain: 'bg-storage-terrain',
  map: 'bg-storage-map',
  saves: 'bg-storage-saves',
  screenshots: 'bg-storage-screenshots',
  logs: 'bg-storage-logs',
  mods: 'bg-storage-mods',
  other: 'bg-storage-other',
}

/** Go's rows gathered into groups, in Go's order within each; a group with nothing is left out. */
export function storageGroups(items: readonly StorageItem[]): StorageGroup[] {
  return GROUPS.map(({ id, title }) => {
    const mine = items.filter((item) => (GROUP_OF[item.kind] ?? 'other') === id)
    return { id, title, items: mine, bytes: mine.reduce((n, item) => n + item.bytes, 0) }
  }).filter((group) => group.items.length > 0)
}

/**
 * Whether a group is drawn as a heading over its own rows, folded at first: Distant Horizons,
 * Xaero's maps and the saves.
 */
export const hasRows = (group: StorageGroup): boolean =>
  group.id === 'terrain' || group.id === 'map' || group.id === 'saves'

/**
 * The server a terrain folder is for. Distant Horizons names it `<server id>_<key>` when the server
 * sends one (`1790374988_frangfurd`), else after the server's name in the list.
 */
export function serverName(item: StorageItem): string {
  const label = item.label || item.name
  return /^\d+_(.+)$/.exec(label)?.[1] ?? label
}

const TITLES: Record<string, string> = {
  terrainDownload: 'Unfinished download',
  worldMap: 'World map',
  minimap: 'Minimap and waypoints',
}

/** A row's title under its group's heading. */
export function storageTitle(item: StorageItem): string {
  if (item.kind === 'terrain') return serverName(item)
  if (item.kind === 'saves') return item.name
  return TITLES[item.kind] ?? item.kind
}

const count = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`

/** The small line under a row's title: its size, and a count or the last write where it helps. */
export function storageDetail(item: StorageItem, locale?: string): string {
  const size = formatBytes(item.bytes)
  switch (item.kind) {
    case 'terrain':
    case 'terrainDownload':
    case 'worldMap':
    case 'minimap':
      return item.lastModified
        ? `${size} · last written ${publishedOn(item.lastModified, locale)}`
        : size
    case 'saves':
      return item.lastModified
        ? `${size} · last played ${publishedOn(item.lastModified, locale)}`
        : size
    case 'screenshots':
      return `${size} · ${count(item.files, 'picture', 'pictures')}`
    case 'logs':
      return `${size} · ${count(item.files, 'file', 'files')}`
  }
  return size
}
