import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import * as App from '../../../wailsjs/go/main/App'
import { models } from '../../../wailsjs/go/models'
import { useEngineStore } from '../../stores/useEngineStore'
import { useGameStore } from '../../stores/useGameStore'
import { useStorageStore } from '../../stores/useStorageStore'
import { BUNDLED_MANIFEST } from '../../lib/manifest'
import { InstanceSection, KEPT, RESET } from './InstanceSection'

vi.mock('../../../wailsjs/go/main/App')

const frangfurd = BUNDLED_MANIFEST.chapters[2]!
const report = {
  root: 'C:/Prism',
  dir: 'C:/Prism/instances',
  present: { frangfurd: true },
  packUrl: { frangfurd: 'https://example.com/frangfurd/pack.toml' },
  sizeBytes: {},
}

const dialog = () => screen.getByRole('dialog')

describe('InstanceSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useEngineStore.setState({ instances: report, installing: null, reinstalled: {}, error: null })
    useGameStore.setState({ states: {} })
    useStorageStore.setState({
      byChapter: {
        frangfurd: {
          chapterId: 'frangfurd',
          totalBytes: 16 * 1024 ** 3,
          freeBytes: -1,
          partial: false,
          running: false,
          items: [],
        },
      },
      unavailable: false,
      readError: {},
      clearing: null,
    })
  })
  afterEach(() => {
    cleanup()
    Reflect.deleteProperty(window, 'go')
  })

  it('asks before a reinstall, naming what stays and what is reset', async () => {
    vi.mocked(App.ReinstallChapter).mockResolvedValue(
      models.ReinstallResult.createFrom({ instances: report, stash: '' }),
    )
    render(<InstanceSection chapter={frangfurd} />)
    expect(screen.getByText('16 GB')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reinstall' }))
    expect(within(dialog()).getByText(KEPT)).toBeInTheDocument()
    expect(within(dialog()).getByText(RESET)).toBeInTheDocument()
    expect(App.ReinstallChapter).not.toHaveBeenCalled()
    fireEvent.click(within(dialog()).getByRole('button', { name: 'Reinstall' }))
    await screen.findByRole('button', { name: 'Reinstalled' })
    expect(App.ReinstallChapter).toHaveBeenCalledWith('frangfurd')
    expect(App.GetChapterStorage).toHaveBeenCalledWith('frangfurd')
  })

  it('cancels without a call, and shows Go’s refusal of an uninstall under the card', async () => {
    vi.mocked(App.UninstallChapter).mockRejectedValue('Prism is open; close it first')
    render(<InstanceSection chapter={frangfurd} />)
    fireEvent.click(screen.getByRole('button', { name: 'Uninstall' }))
    expect(within(dialog()).getByText('kapital-frangfurd · 16 GB')).toBeInTheDocument()
    fireEvent.click(within(dialog()).getByRole('button', { name: 'Cancel' }))
    expect(App.UninstallChapter).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Uninstall' }))
    fireEvent.click(within(dialog()).getByRole('button', { name: 'Uninstall' }))
    expect(await screen.findByText('Prism is open; close it first')).toBeInTheDocument()
    expect(App.UninstallChapter).toHaveBeenCalledWith('frangfurd')
  })

  it('holds both while the game runs', () => {
    useGameStore.setState({
      states: { frangfurd: { chapterId: 'frangfurd', phase: 'running' } as never },
    })
    render(<InstanceSection chapter={frangfurd} />)
    expect(screen.getByRole('button', { name: 'Reinstall' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Uninstall' })).toBeDisabled()
  })
})
