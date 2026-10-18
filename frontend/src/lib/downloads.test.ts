import { describe, expect, it } from 'vitest'
import { downloadRows } from './downloads'

const name = (id: string) => (id === 'frangfurd' ? 'Frangfurd' : id)

describe('downloadRows', () => {
  it('is empty when nothing downloads', () => {
    expect(downloadRows(null, false, null, {}, name)).toEqual([])
    expect(
      downloadRows(
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
  })

  it('counts the steps after a download as full', () => {
    const [row] = downloadRows(
      { phase: 'unpacking', received: 0, total: 0, error: '' },
      false,
      null,
      {},
      name,
    )
    expect(row).toMatchObject({ title: 'Prism Launcher', value: 1, line: 'Unpacking' })
  })
})
