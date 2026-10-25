import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { SidebarButton } from './SidebarButton'

function button(launcher: { compact: boolean; held: boolean }, onToggleSidebar = vi.fn()) {
  const choose = vi.fn()
  render(
    <SidebarButton
      size="sm"
      sidebarCompact={false}
      onToggleSidebar={onToggleSidebar}
      launcher={{ ...launcher, choose }}
    />,
  )
  return { choose, onToggleSidebar }
}

describe('SidebarButton', () => {
  afterEach(cleanup)

  it('narrows the sidebar on a click and opens the launcher menu on a right-click', () => {
    const { choose, onToggleSidebar } = button({ compact: false, held: false })
    const trigger = screen.getByRole('button', { name: 'Compact Sidebar' })
    fireEvent.click(trigger)
    expect(onToggleSidebar).toHaveBeenCalledOnce()
    expect(screen.queryByRole('menu')).toBeNull()

    fireEvent.contextMenu(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    const item = screen.getByRole('menuitem', { name: 'Compact launcher' })
    expect(item).toHaveFocus()
    fireEvent.click(item)
    expect(choose).toHaveBeenCalledWith(true)
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('goes back to the full launcher on a click while compact', () => {
    const { choose, onToggleSidebar } = button({ compact: true, held: false })
    fireEvent.click(screen.getByRole('button', { name: 'Full Launcher' }))
    expect(choose).toHaveBeenCalledWith(false)
    expect(onToggleSidebar).not.toHaveBeenCalled()
  })

  it('greys out the full launcher while the window is too small for it', () => {
    const { choose } = button({ compact: true, held: true })
    fireEvent.contextMenu(screen.getByRole('button', { name: 'Full Launcher' }))
    const item = screen.getByRole('menuitem', { name: 'Full launcher' })
    expect(item).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(item)
    expect(choose).not.toHaveBeenCalled()
  })

  it('closes the menu on Escape and on a press outside it', () => {
    button({ compact: false, held: false })
    const trigger = screen.getByRole('button', { name: 'Compact Sidebar' })
    fireEvent.contextMenu(trigger)
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(trigger).toHaveFocus()
    fireEvent.contextMenu(trigger)
    fireEvent.pointerDown(document.body)
    expect(screen.queryByRole('menu')).toBeNull()
  })
})
