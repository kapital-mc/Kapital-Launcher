import svg from '../assets/brand/kapital-launcher-wordmark.svg?raw'

/**
 * The traced Kapital Launcher wordmark (assets/brand/kapital-launcher-wordmark.svg)
 * as path data for React to draw: its view box, the outline of the whole mark
 * and each letter's face in reading order. Only the `d` attributes are read,
 * so the file reaches the page as path data and never as markup.
 */
export interface Wordmark {
  viewBox: string
  outline: string[]
  faces: string[]
}

export function parseWordmark(source: string): Wordmark {
  const mark: Wordmark = {
    viewBox: /viewBox="([\d. ]+)"/.exec(source)?.[1] ?? '',
    outline: [],
    faces: [],
  }
  for (const [, part, d] of source.matchAll(
    /<path data-part="(outline|face)"[^>]*?\sd="([^"]+)"/g,
  )) {
    if (part === 'outline') mark.outline.push(d ?? '')
    else mark.faces.push(d ?? '')
  }
  return mark
}

export const WORDMARK = parseWordmark(svg)
