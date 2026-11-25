import type { Contraption, ContraptionKind } from '../types'
import { formatBytes } from './bytes'

/** The kinds in the order the page lists them, and each one's heading. */
export const CONTRAPTION_KINDS: readonly { kind: ContraptionKind; title: string }[] = [
  { kind: 'toolgun', title: 'Tool Gun' },
  { kind: 'schematic', title: 'Schematics' },
]

const plural = (n: number, one: string, many: string, locale?: string) =>
  `${n.toLocaleString(locale)} ${n === 1 ? one : many}`

/** A row's facts: blocks, sub-levels or size, the file's size and its date. */
export function contraptionDetail(c: Contraption, locale?: string): string {
  const parts: string[] = []
  if (c.blocks >= 0) parts.push(plural(c.blocks, 'block', 'blocks', locale))
  if (c.subLevels > 0) parts.push(plural(c.subLevels, 'sub-level', 'sub-levels', locale))
  if (c.size?.length === 3) parts.push(c.size.join(' × '))
  parts.push(formatBytes(c.bytes))
  parts.push(new Date(c.modified).toLocaleDateString(locale, { dateStyle: 'medium' }))
  return parts.join(' · ')
}

/**
 * A block or item id as a name: `create:copycat_slab` is "Copycat slab", with the mod's namespace
 * beside it when it is not Minecraft's.
 */
export function materialName(id: string): { name: string; mod: string | null } {
  const colon = id.indexOf(':')
  const ns = colon < 0 ? 'minecraft' : id.slice(0, colon)
  const path = (colon < 0 ? id : id.slice(colon + 1)).replaceAll('_', ' ')
  const name = path.charAt(0).toUpperCase() + path.slice(1)
  return { name, mod: ns === 'minecraft' ? null : ns }
}

/** The key a row is known by across calls. */
export const contraptionKey = (c: Pick<Contraption, 'kind' | 'file'>) => `${c.kind}/${c.file}`
