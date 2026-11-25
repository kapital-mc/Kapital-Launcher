import type { GameSettingsItem } from '../types'

/** The key bindings item: the key lines of options.txt rather than files of its own. */
const KEYS = 'keys'

/** Whether there is anything to reset: files on disk, or key lines in options.txt. */
export const canReset = (item: GameSettingsItem): boolean =>
  item.id === KEYS ? item.keys > 0 : item.files.length > 0

const baseName = (path: string): string => path.slice(path.lastIndexOf('/') + 1)

/**
 * The fact beside a setting's name: the key count, its one file, how many files, or Default once
 * nothing is left to reset.
 */
export function settingsDetail(item: GameSettingsItem): string {
  if (!canReset(item)) return 'Default'
  if (item.id === KEYS) return `${item.keys} ${item.keys === 1 ? 'binding' : 'bindings'}`
  if (item.files.length === 1) return baseName(item.files[0]!)
  return `${item.files.length} files`
}

/** The item's files, one per line, for the fact's tooltip; "" when there are none. */
export const settingsFiles = (item: GameSettingsItem): string => item.files.join('\n')
