import type { StorageItem } from '../types'
import { formatBytes } from './bytes'
import { publishedOn } from './terrain'

/** The kinds Go deletes on request (services.ClearStorage). */
const DELETABLE: readonly string[] = ['terrain', 'terrainDownload', 'worldMap', 'logs']

export const canDelete = (item: StorageItem): boolean => DELETABLE.includes(item.kind)

/** A row's key: its kind, and the server folder for terrain. */
export const storageKey = (item: StorageItem): string => `${item.kind}/${item.name}`

/**
 * The server a terrain folder is for. Distant Horizons names it `<server id>_<key>` when the server
 * sends one (`1790374988_frangfurd`), else after the server's name in the list.
 */
export function serverName(item: StorageItem): string {
  const label = item.label || item.name
  return /^\d+_(.+)$/.exec(label)?.[1] ?? label
}

const TITLES: Record<string, string> = {
  terrainDownload: 'Unfinished terrain download',
  worldMap: 'World map',
  screenshots: 'Screenshots',
  logs: 'Logs and crash reports',
  mods: 'Mods',
  other: 'Everything else',
}

export function storageTitle(item: StorageItem): string {
  if (item.kind === 'terrain') return `Distant Horizons terrain: ${serverName(item)}`
  return TITLES[item.kind] ?? item.kind
}

const count = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`

/** The small line under a row's title: its size and what to know about it. */
export function storageDetail(item: StorageItem, locale?: string): string {
  const size = formatBytes(item.bytes)
  switch (item.kind) {
    case 'terrain':
      return item.lastModified
        ? `${size} · last written ${publishedOn(item.lastModified, locale)}`
        : size
    case 'terrainDownload':
      return `${size} · kept so the download can continue`
    case 'worldMap':
      return `${size} · the map you explored`
    case 'screenshots':
      return `${size} · ${count(item.files, 'picture', 'pictures')}`
    case 'logs':
      return `${size} · ${count(item.files, 'file', 'files')}`
    case 'mods':
      return `${size} · the pack's mods, put back on every Play`
    case 'other':
      return `${size} · settings, caches and saves`
  }
  return size
}

/** What the confirmation says before a delete. */
export function confirmText(item: StorageItem): string {
  const size = formatBytes(item.bytes)
  switch (item.kind) {
    case 'terrain':
      return `Delete ${size} of terrain for ${serverName(item)}? It comes back from the server as you play.`
    case 'terrainDownload':
      return `Delete the unfinished download (${size})? The next one starts over.`
    case 'worldMap':
      return `Delete the world map (${size})? What you explored is lost.`
    case 'logs':
      return `Delete the logs and crash reports (${count(item.files, 'file', 'files')}, ${size})?`
  }
  return `Delete ${size}?`
}
