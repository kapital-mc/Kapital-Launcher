import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import * as App from '../../../wailsjs/go/main/App'
import { Downloads } from './Downloads'
import { useEngineStore } from '../../stores/useEngineStore'
import { useTailscaleStore } from '../../stores/useTailscaleStore'
import { useTerrainStore } from '../../stores/useTerrainStore'

vi.mock('../../../wailsjs/go/main/App')
vi.mock('../../../wailsjs/runtime/runtime')

describe('Downloads', () => {
  beforeEach(() => {
    useEngineStore.setState({ install: null, engine: null })
    useTailscaleStore.setState({ install: null })
    useTerrainStore.setState({ progress: {} })
  })
  afterEach(cleanup)

  it('is not there while nothing downloads', () => {
    render(<Downloads />)
    expect(screen.queryByRole('region', { name: 'Downloads' })).toBeNull()
  })

  it('shows each running download with its bar', () => {
    useEngineStore.setState({
      install: { phase: 'downloading', received: 50, total: 100, error: '' },
    })
    useTerrainStore.setState({
      progress: {
        frangfurd: { chapterId: 'frangfurd', phase: 'downloading', received: 1, total: 4 },
      },
    })
    render(<Downloads />)
    const section = screen.getByRole('region', { name: 'Downloads' })
    expect(section).toHaveTextContent('Prism Launcher')
    expect(section).toHaveTextContent('Frangfurd terrain')
    expect(screen.getByRole('progressbar', { name: 'Prism Launcher' })).toHaveAttribute(
      'aria-valuenow',
      '50',
    )
  })
  it('keeps each bar and folds the names away when compact', () => {
    useEngineStore.setState({
      install: { phase: 'downloading', received: 50, total: 100, error: '' },
    })
    render(<Downloads compact />)
    expect(screen.getByRole('progressbar', { name: 'Prism Launcher' })).toBeInTheDocument()
    expect(screen.getByText('50%')).toBeVisible()
    expect(screen.getByText('Prism Launcher').closest('[inert]')).not.toBeNull()
  })

  it('cancels each download from its row, an install only while it downloads', async () => {
    vi.mocked(App.CancelTailscaleInstall).mockResolvedValue(true)
    vi.mocked(App.CancelDHSnapshot).mockResolvedValue(true)
    useEngineStore.setState({
      install: { phase: 'verifying', received: 100, total: 100, error: '' },
    })
    useTailscaleStore.setState({
      install: { phase: 'downloading', received: 10, total: 20, error: '' },
    })
    useTerrainStore.setState({
      progress: {
        frangfurd: { chapterId: 'frangfurd', phase: 'downloading', received: 1, total: 4 },
      },
    })
    render(<Downloads />)
    expect(screen.queryByRole('button', { name: 'Cancel Prism Launcher' })).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel Tailscale' }))
    await waitFor(() => expect(App.CancelTailscaleInstall).toHaveBeenCalledOnce())
    fireEvent.click(screen.getByRole('button', { name: 'Cancel Frangfurd terrain' }))
    await waitFor(() => expect(App.CancelDHSnapshot).toHaveBeenCalledWith('frangfurd'))
  })

  it('draws its numbers with even-width digits', () => {
    useTailscaleStore.setState({
      install: { phase: 'downloading', received: 10, total: 20, error: '' },
    })
    render(<Downloads />)
    expect(screen.getByText('Downloading 0.0 MB of 0.0 MB')).toHaveClass('tabular-nums')
    expect(screen.getByText('50%')).toHaveClass('tabular-nums')
  })
})
