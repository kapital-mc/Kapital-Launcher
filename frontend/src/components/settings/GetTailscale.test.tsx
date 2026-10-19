import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { GetTailscale } from './GetTailscale'
import type { TailscaleInfo, TailscaleInstallProgress } from '../../types/tailscale'

const release = {
  version: '1.104.1',
  file: 'Tailscale-1.104.1-macos.pkg',
  url: 'https://pkgs.tailscale.com/stable/Tailscale-1.104.1-macos.pkg',
  size: 22332629,
  digest: 'sha256:00',
}

function popup(
  install: TailscaleInstallProgress | null,
  info: TailscaleInfo | null,
  platform = 'darwin',
) {
  const calls = { onConfirm: vi.fn(), onClose: vi.fn(), onConnect: vi.fn() }
  render(
    <GetTailscale release={release} install={install} info={info} platform={platform} {...calls} />,
  )
  return calls
}

const done = { phase: 'done', received: 0, total: 0, error: '' }
const info = (state: string, connected: string[] = []): TailscaleInfo => ({
  state,
  version: '',
  canInstall: true,
  signingIn: false,
  connected,
})

describe('GetTailscale', () => {
  afterEach(cleanup)

  it('says what it downloads and what macOS and Windows will ask', () => {
    const calls = popup(null, info('missing'))
    const dialog = screen.getByRole('dialog', { name: 'Get Tailscale' })
    expect(dialog).toHaveTextContent('Version 1.104.1 · 21.3 MB · pkgs.tailscale.com')
    expect(dialog).toHaveTextContent('system extension')
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    expect(calls.onConfirm).toHaveBeenCalledOnce()
    cleanup()
    popup(null, info('missing'), 'windows')
    expect(screen.getByRole('dialog')).toHaveTextContent('administrator approval')
  })

  it('shows the installer step while it runs', () => {
    popup({ phase: 'installing', received: 1, total: 1, error: '' }, info('missing'))
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100')
    expect(screen.getByText(/installer is open/)).toBeInTheDocument()
  })

  it('offers the next step once installed', () => {
    for (const state of ['needsLogin', 'stopped', 'running']) {
      const calls = popup(done, info(state))
      fireEvent.click(screen.getByRole('button', { name: 'Connect' }))
      expect(calls.onConnect).toHaveBeenCalledOnce()
      cleanup()
    }
    popup(done, info('running', ['frangfurd']))
    expect(screen.getByRole('dialog')).toHaveTextContent('Tailscale is connected')
    cleanup()
    popup(done, { ...info('needsLogin'), signingIn: true })
    expect(screen.getByRole('dialog')).toHaveTextContent('in your browser')
    cleanup()
    popup(done, info('missing'))
    expect(screen.getByRole('dialog')).toHaveTextContent('Finish its installer')
  })
})
