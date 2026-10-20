import svg from '../assets/brand/kapital-launcher-wordmark.svg?raw'

/** One traced shape: its path data, and how far its inside reaches from its edge, in view box units. */
export interface WordmarkShape {
  d: string
  reach: number
}

/**
 * The traced Kapital Launcher wordmark (assets/brand/kapital-launcher-wordmark.svg)
 * as path data for React to draw: its view box and size, the outline of the
 * whole mark and each letter's face in reading order. Only the attributes
 * named here are read, so the file reaches the page as data and never as
 * markup.
 */
export interface Wordmark {
  viewBox: string
  width: number
  height: number
  outline: WordmarkShape[]
  faces: WordmarkShape[]
}

export function parseWordmark(source: string): Wordmark {
  const viewBox = /viewBox="([\d. ]+)"/.exec(source)?.[1] ?? ''
  const [, , width = 0, height = 0] = viewBox.split(' ').map(Number)
  const mark: Wordmark = { viewBox, width, height, outline: [], faces: [] }
  const shapes = /<path data-part="(outline|face)" data-reach="(\d+)"[^>]*?\sd="([^"]+)"/g
  for (const [, part, reach, d] of source.matchAll(shapes)) {
    const shape = { d: d ?? '', reach: Number(reach) }
    if (part === 'outline') mark.outline.push(shape)
    else mark.faces.push(shape)
  }
  return mark
}

export const WORDMARK = parseWordmark(svg)
