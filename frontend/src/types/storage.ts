// What the storage section's bound methods carry (app_storage.go).

/**
 * A kind of storage in an instance. Terrain, the unfinished terrain download, the world map, the
 * screenshots and the logs can be deleted; the minimap, mods and the rest are only shown. Every
 * kind's folder can be opened.
 */
export type StorageKind =
  'terrain' | 'terrainDownload' | 'worldMap' | 'minimap' | 'screenshots' | 'logs' | 'mods' | 'other'

/** One kind of storage in an instance; for terrain, one server's Distant Horizons folder. */
export interface StorageItem {
  kind: StorageKind | string
  /** The server's folder for terrain, what ClearChapterStorage takes back; '' otherwise. */
  name: string
  /** The server's name as Distant Horizons wrote it, decoded, for terrain; '' otherwise. */
  label: string
  bytes: number
  files: number
  /** The newest file's time, RFC 3339, or ''. */
  lastModified: string
}

/** What a chapter's instance holds on disk, by kind, from GetChapterStorage. */
export interface ChapterStorage {
  chapterId: string
  totalBytes: number
  /** Free space on the instance's disk, -1 when unknown. */
  freeBytes: number
  /** Terrain largest first, then the other kinds in a fixed order; empty kinds left out. */
  items: StorageItem[]
  /** The walk stopped at its ceiling, so the sizes are low. */
  partial: boolean
  /** The tracker or the game log says the game runs; the log half is a guess, shown as a hint. */
  running: boolean
}
