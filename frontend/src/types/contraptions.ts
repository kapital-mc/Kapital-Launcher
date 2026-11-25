// What the Contraptions page's bound methods carry (app_contraptions.go).

/** The Tool Gun's blueprints (.excraft) and Create's schematics (.nbt). */
export type ContraptionKind = 'toolgun' | 'schematic'

/** One block or item id and how many. */
export interface MaterialCount {
  id: string
  count: number
}

/** One saved contraption file and the facts read from it. */
export interface Contraption {
  kind: ContraptionKind | string
  /** The file's name in its kind's folder, the handle every call takes back. */
  file: string
  /** The file name without its extension, as the game lists it. */
  name: string
  bytes: number
  /** Unix milliseconds. */
  modified: number
  /** -1 when the file does not say. */
  blocks: number
  blockTypes: number
  subLevels: number
  /** A schematic's bounding box, x y z. */
  size?: number[]
  /** The blocks, most first. */
  materials: MaterialCount[]
  /** What its containers hold, from the Tool Gun's summary. */
  items: MaterialCount[]
  /** The facts could not be read; the file is still there. */
  unread: boolean
}
