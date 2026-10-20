import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import * as Bindings from '../../../wailsjs/go/main/App'
import { selectWhatsNewShown, useWhatsNewStore } from '../../stores/useWhatsNewStore'
import type { Changelog, WhatsNew as WhatsNewInfo } from '../../types'
import { WhatsNew } from './WhatsNew'

vi.mock('../../../wailsjs/go/main/App')

const news: WhatsNewInfo = {
  version: 'v1.0.0',
  releases: [
    {
      version: 'v1.0.0',
      highlights: [
        { title: 'Tailscale', text: 'Join over Tailscale.' },
        { title: 'Bug fixes', items: ['A fix', 'Another fix'] },
      ],
    },
  ],
}

const changelog: Changelog = {
  releases: [
    {
      tag: 'v1.0.0',
      date: '2026-10-18',
      sections: [{ title: 'Features', items: ['Join the server over Tailscale'] }],
    },
  ],
  reason: '',
  page: 'https://kapital-launcher.pages.dev/',
}

beforeEach(() => {
  useWhatsNewStore.setState({ whatsNew: null, changelog: null, error: null })
  vi.mocked(Bindings.GetWhatsNew)
    .mockReset()
    .mockResolvedValue(news as never)
  vi.mocked(Bindings.GetWhatsNewChangelog)
    .mockReset()
    .mockResolvedValue(changelog as never)
  vi.mocked(Bindings.MarkWhatsNewSeen).mockReset().mockResolvedValue(true)
  vi.mocked(Bindings.OpenExternal).mockReset().mockResolvedValue()
})

afterEach(cleanup)

describe('useWhatsNewStore', () => {
  it('shows nothing without releases or a bridge', async () => {
    vi.mocked(Bindings.GetWhatsNew).mockResolvedValue({
      version: 'v1.0.0',
      releases: null,
    } as never)
    await useWhatsNewStore.getState().load()
    expect(selectWhatsNewShown(useWhatsNewStore.getState())).toBe(false)
    vi.mocked(Bindings.GetWhatsNew).mockRejectedValue('no bridge')
    await useWhatsNewStore.getState().load()
    expect(selectWhatsNewShown(useWhatsNewStore.getState())).toBe(false)
  })

  it('stays up with the error when the version cannot be recorded', async () => {
    await useWhatsNewStore.getState().load()
    vi.mocked(Bindings.MarkWhatsNewSeen).mockRejectedValue('disk full')
    await useWhatsNewStore.getState().close()
    expect(selectWhatsNewShown(useWhatsNewStore.getState())).toBe(true)
    expect(useWhatsNewStore.getState().error).toBe('disk full')
  })
})

describe('WhatsNew', () => {
  it('shows the highlights with grouped items, and the X records them as seen', async () => {
    render(<WhatsNew />)
    expect(await screen.findByText('Tailscale')).toBeInTheDocument()
    expect(screen.getByText('Join over Tailscale.')).toBeInTheDocument()
    expect(screen.getByText('Another fix')).toBeInTheDocument()
    expect(screen.getByText('Kapital Launcher v1.0.0')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Continue' })).toBeNull()
    fireEvent.click(document.querySelector('.scrim-wash')!)
    expect(screen.getByText('Tailscale')).toBeInTheDocument()
    expect(Bindings.MarkWhatsNewSeen).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    await vi.waitFor(() => expect(screen.queryByText('Tailscale')).toBeNull())
    expect(Bindings.MarkWhatsNewSeen).toHaveBeenCalledTimes(1)
  })

  it('reads the full changelog only when it is opened', async () => {
    render(<WhatsNew />)
    const toggle = await screen.findByRole('button', { name: 'Full changelog' })
    expect(Bindings.GetWhatsNewChangelog).not.toHaveBeenCalled()
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(await screen.findByText('Join the server over Tailscale')).toBeInTheDocument()
    expect(screen.getByText('18 October 2026')).toBeInTheDocument()
    expect(Bindings.GetWhatsNewChangelog).toHaveBeenCalledTimes(1)
  })

  it('links to the download site when the changelog could not be read', async () => {
    vi.mocked(Bindings.GetWhatsNewChangelog).mockResolvedValue({
      releases: null,
      reason: 'no answer',
      page: changelog.page,
    } as never)
    render(<WhatsNew />)
    fireEvent.click(await screen.findByRole('button', { name: 'Full changelog' }))
    fireEvent.click(await screen.findByRole('button', { name: /download site/ }))
    expect(screen.getByText(/could not be read/)).toBeInTheDocument()
    expect(Bindings.OpenExternal).toHaveBeenCalledWith(changelog.page)
  })
})
