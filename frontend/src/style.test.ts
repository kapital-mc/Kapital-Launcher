import { describe, expect, it } from 'vitest'
import { STYLE_CSS as css } from './test-css'

/** The declarations of the first rule whose selector is exactly `selector`. */
const rule = (selector: string) => {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return new RegExp(`(?:^|\\n)\\s*${escaped}\\s*\\{([^}]*)\\}`).exec(css)?.[1] ?? ''
}

describe('style.css cursors', () => {
  // Wails draws a frameless window's resize cursor as an inline style on html
  // (runtime/desktop/main.js, setResize): a cursor set on body, a link or a button
  // would cover it, and the window's edges would show the arrow.
  it('sets the arrow on html, where the resize cursor can replace it', () => {
    expect(rule('html')).toMatch(/cursor:\s*default/)
    expect(rule('body')).not.toMatch(/cursor:/)
  })

  it('lets links, buttons and selects inherit the cursor rather than set their own', () => {
    expect(rule('a:any-link,\n  button,\n  select')).toMatch(/cursor:\s*inherit/)
  })
})
