import { describe, expect, it } from 'vitest'
import { downloadRows } from './downloads'

const name = (id: string) => (id === 'frangfurd' ? 'Frangfurd' : id)

describe('downloadRows', () => {
  it('is empty when nothing downloads', () => {
    expect(downloadRows(null, null, false, null, {}, name)).toEqual([])
    expect(
      downloadRows(
        null,
        { phase: 'done', received: 0, total: 0, error: '' },
        false,
        { phase: 'failed', received: 0, total: 0, error: 'x' },
        { frangfurd: { chapterId: 'frangfurd', phase: 'cancelled', received: 0, total: 0 } },
        name,
      ),
    ).toEqual([])
  })

  it('lists Prism, Tailscale and terrain, each with its share and step', () => {
    const rows = downloadRows(
      null,
      { phase: 'downloading', received: 25, total: 100, error: '' },
      true,
      { phase: 'installing', received: 10, total: 10, error: '' },
      { frangfurd: { chapterId: 'frangfurd', phase: 'downloading', received: 1, total: 4 } },
      name,
    )
    expect(rows.map((r) => [r.title, r.value])).toEqual([
      ['Prism Launcher update', 0.25],
      ['Tailscale', 1],
      ['Frangfurd terrain', 0.25],
    ])
    expect(rows[1]!.line).toContain('installer is open')
    expect(rows[2]!.line).toBe('Downloading 1 B of 4 B')
    // An install is cancelled only while it downloads; the terrain at any step.
    expect(rows.map((r) => r.cancel)).toEqual([
      { kind: 'prism' },
      null,
      { kind: 'terrain', chapterId: 'frangfurd' },
    ])
  })

  it("lists the launcher's own update first, cancellable while it downloads", () => {
    const rows = downloadRows(
      { phase: 'downloading', received: 1, total: 4, version: 'v1.1.0', error: '' },
      null,
      false,
      null,
      {},
      name,
    )
    expect(rows).toEqual([
      {
        key: 'launcher',
        title: 'Kapital Launcher update',
        value: 0.25,
        line: 'Downloading 1 B of 4 B',
        cancel: { kind: 'launcher' },
      },
    ])
    const [checking] = downloadRows(
      { phase: 'verifying', received: 4, total: 4, version: 'v1.1.0', error: '' },
      null,
      false,
      null,
      {},
      name,
    )
    expect(checking).toMatchObject({ value: 1, cancel: null })
    for (const phase of ['ready', 'failed']) {
      const p = { phase, received: 0, total: 0, version: 'v1.1.0', error: '' }
      expect(downloadRows(p, null, false, null, {}, name)).toEqual([])
    }
  })

  it('counts the steps after a download as full', () => {
    const [row] = downloadRows(
      null,
      { phase: 'unpacking', received: 0, total: 0, error: '' },
      false,
      null,
      {},
      name,
    )
    expect(row).toMatchObject({ title: 'Prism Launcher', value: 1, line: 'Unpacking' })
  })
})
