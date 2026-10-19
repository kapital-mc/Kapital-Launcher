import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
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
    const { container } = render(<Downloads />)
    expect(container).toBeEmptyDOMElement()
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
})
