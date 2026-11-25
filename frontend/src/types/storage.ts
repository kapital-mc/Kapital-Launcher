// What the storage section's bound methods carry (app_storage.go).

/**
 * A kind of storage in an instance. Terrain, the unfinished terrain download, the world map, the
 * server map, the saves, the replays, the screenshots and the logs can be deleted; the minimap,
 * mods and the rest are only shown. Every kind's folder can be opened.
 */
export type StorageKind =
  | 'terrain'
  | 'terrainDownload'
  | 'worldMap'
  | 'minimap'
  | 'serverMap'
  | 'saves'
  | 'replays'
  | 'screenshots'
  | 'logs'
  | 'mods'
  | 'other'

/**
 * One kind of storage in an instance; for terrain, one server's Distant Horizons folder, for
 * saves, one world, and for replays, one recording.
 */
export interface StorageItem {
  kind: StorageKind | string
  /**
   * The server's folder for terrain, what ClearChapterStorage takes back, the world's folder for a
   * save, the file for a replay; '' otherwise.
   */
  name: string
  /** The server's name as Distant Horizons wrote it, decoded, for terrain; '' otherwise. */
  label: string
  bytes: number
  files: number
  /** The newest file's time, RFC 3339, or ''. */
  lastModified: string
  /** A save's icon.png as a data URL; absent for a world without one and every other kind. */
  icon?: string
}

/** What a chapter's instance holds on disk, by kind, from GetChapterStorage. */
export interface ChapterStorage {
  chapterId: string
  totalBytes: number
  /** Free space on the instance's disk, -1 when unknown. */
  freeBytes: number
  /**
   * Terrain largest first, then the other kinds in a fixed order, saves largest first in theirs;
   * empty kinds left out.
   */
  items: StorageItem[]
  /** The walk stopped at its ceiling, so the sizes are low. */
  partial: boolean
  /** The tracker or the game log says the game runs; the log half is a guess, shown as a hint. */
  running: boolean
}

/** One game setting the Instance section can reset, from GetGameSettings. */
export interface GameSettingsItem {
  /** What ResetGameSetting takes back. */
  id: string
  name: string
  /** Its files on disk now, relative to the game folder; none after a reset until the next Play. */
  files: string[]
  /** options.txt's key binding lines, for the key bindings item. */
  keys: number
}

/** The game settings a chapter's instance can reset, in the page's order. */
export interface ChapterGameSettings {
  chapterId: string
  items: GameSettingsItem[]
  /** The tracker or the game log says the game runs; a reset is refused while it does. */
  running: boolean
}
