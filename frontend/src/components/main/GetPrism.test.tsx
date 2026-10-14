import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { GetPrism } from './GetPrism'
import type { PrismRelease } from '../../types'

const release: PrismRelease = {
  version: '11.1.1',
  asset: 'PrismLauncher-macOS-11.1.1.zip',
  url: 'https://github.com/PrismLauncher/PrismLauncher/releases/download/11.1.1/x.zip',
  size: 41.6 * 1024 * 1024,
  digest: 'sha256:00',
  page: 'https://github.com/PrismLauncher/PrismLauncher/releases/tag/11.1.1',
  installed: '',
  updateAvailable: false,
}

function open() {
  const calls = {
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
    onOpenReleasePage: vi.fn(),
    onOpenPrismSite: vi.fn(),
  }
  render(<GetPrism release={release} {...calls} />)
  return calls
}

describe('GetPrism', () => {
  afterEach(cleanup)

  it('is a modal dialog with the version, size, source and licence (ADR-11)', () => {
    open()
    const dialog = screen.getByRole('dialog', { name: 'Get Prism Launcher' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveTextContent('Version 11.1.1 · 41.6 MB · GPL-3.0')
    expect(dialog).toHaveTextContent("Prism's releases on GitHub")
    expect(dialog).toHaveFocus()
  })

  it('floats over the window instead of taking room in the action bar', () => {
    const { container } = render(<div />)
    open()
    const dialog = screen.getByRole('dialog')
    expect(container).not.toContainElement(dialog)
    expect(dialog.parentElement).toHaveClass('fixed', 'inset-0', 'scrim-wash')
  })

  it('downloads only on Download', () => {
    const calls = open()
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(calls.onConfirm).toHaveBeenCalledOnce()
    expect(calls.onCancel).not.toHaveBeenCalled()
  })

  it('cancels on Not now, Escape and a click on the cover, and not on a click inside', () => {
    const calls = open()
    fireEvent.click(screen.getByRole('dialog'))
    expect(calls.onCancel).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Not now' }))
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.click(screen.getByRole('dialog').parentElement!)
    expect(calls.onCancel).toHaveBeenCalledTimes(3)
    expect(calls.onConfirm).not.toHaveBeenCalled()
  })

  it('opens the release notes and the website from its links', () => {
    const calls = open()
    fireEvent.click(screen.getByRole('button', { name: "What's in 11.1.1" }))
    fireEvent.click(screen.getByRole('button', { name: "Prism's website" }))
    expect(calls.onOpenReleasePage).toHaveBeenCalledOnce()
    expect(calls.onOpenPrismSite).toHaveBeenCalledOnce()
  })

  it('gives focus back to what had it when it closes', () => {
    render(<button type="button">Get Prism Launcher</button>)
    const trigger = screen.getByRole('button', { name: 'Get Prism Launcher' })
    trigger.focus()
    const calls = {
      onConfirm: vi.fn(),
      onCancel: vi.fn(),
      onOpenReleasePage: vi.fn(),
      onOpenPrismSite: vi.fn(),
    }
    const { unmount } = render(<GetPrism release={release} {...calls} />)
    expect(trigger).not.toHaveFocus()
    unmount()
    expect(trigger).toHaveFocus()
  })
})
