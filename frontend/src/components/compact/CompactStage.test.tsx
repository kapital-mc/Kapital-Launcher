import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { CompactStage } from './CompactStage'
import { BUNDLED_MANIFEST } from '../../lib/manifest'

vi.mock('../../../wailsjs/go/main/App')

const chapters = BUNDLED_MANIFEST.chapters
const noop = () => undefined

function stage(selectedId: string, shown = true) {
  const onSelect = vi.fn()
  const onOpenSettings = vi.fn()
  render(
    <CompactStage
      chapters={chapters}
      selectedId={selectedId}
      onSelect={onSelect}
      onOpenSettings={onOpenSettings}
      onOpenLogs={noop}
      onOpenMap={noop}
      shown={shown}
    />,
  )
  return { onSelect, onOpenSettings }
}

describe('CompactStage', () => {
  afterEach(cleanup)

  it('shows every chapter as a panel in the nav order, the open one wide with its tools and Play', () => {
    const lux = chapters[0]!
    const { onSelect, onOpenSettings } = stage(lux.id)
    const panels = screen.getAllByRole('group').filter((g) => g.dataset.morph === 'panel')
    expect(panels.map((p) => p.getAttribute('aria-label'))).toEqual(chapters.map((c) => c.name))
    expect(panels[0]).toHaveAttribute('data-active')
    expect(panels[1]).not.toHaveAttribute('data-active')

    const open = within(panels[0]!)
    fireEvent.click(open.getByRole('button', { name: `${lux.name} settings` }))
    expect(onOpenSettings).toHaveBeenCalledWith(lux.id)
    expect(open.getByRole('button', { name: `Play ${lux.name}` })).toBeInTheDocument()
    // The other chapters' tools and Play are there for the switch, but out of reach.
    const idleTools = within(panels[1]!).getByRole('button', {
      name: `${chapters[1]!.name} settings`,
    })
    expect(idleTools.closest('[inert]')).not.toBeNull()

    fireEvent.click(screen.getByRole('button', { name: `Open ${chapters[2]!.name}` }))
    expect(onSelect).toHaveBeenCalledWith(chapters[2]!.id)
    expect(screen.queryByRole('button', { name: `Open ${lux.name}` })).toBeNull()
  })

  it('stays laid out but hidden and inert while it is not the layout shown', () => {
    stage(chapters[0]!.id, false)
    const root = document.querySelector('[data-layout="compact"]')!
    expect(root).toHaveClass('invisible')
    expect(root).toHaveAttribute('inert')
    expect(root).toHaveAttribute('aria-hidden', 'true')
  })
})
