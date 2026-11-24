import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import * as App from '../../../wailsjs/go/main/App'
import { useGameSettingsStore } from '../../stores/useGameSettingsStore'
import { useGameStore } from '../../stores/useGameStore'
import { BUNDLED_MANIFEST } from '../../lib/manifest'
import type { ChapterGameSettings, GameSettingsItem } from '../../types'
import { GameSettingsCard } from './GameSettingsCard'

vi.mock('../../../wailsjs/go/main/App')

const frangfurd = BUNDLED_MANIFEST.chapters[2]!

const item = (id: string, name: string, files: string[], keys = 0): GameSettingsItem => ({
  id,
  name,
  files,
  keys,
})

const view = (items: GameSettingsItem[]): ChapterGameSettings => ({
  chapterId: 'frangfurd',
  running: false,
  items,
})

const full = view([
  item('options', 'Game options', ['options.txt']),
  item('keys', 'Key bindings', ['options.txt'], 102),
  item('video', 'Video settings', [
    'config/sodium-options.json',
    'config/sodium-extra-options.json',
  ]),
  item('distantHorizons', 'Distant Horizons', []),
])

const resetButton = (name: string) => screen.getByTitle(`Reset ${name}`)

describe('GameSettingsCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useGameStore.setState({ states: {} })
    useGameSettingsStore.setState({
      byChapter: { frangfurd: full },
      unavailable: false,
      readError: {},
      resetting: null,
    })
    vi.mocked(App.GetGameSettings).mockResolvedValue(full as never)
  })
  afterEach(() => {
    cleanup()
    Reflect.deleteProperty(window, 'go')
  })

  it('lists each setting with one fact, and Default for one with nothing left', () => {
    render(<GameSettingsCard chapter={frangfurd} />)
    expect(screen.getByText('options.txt')).toBeInTheDocument()
    expect(screen.getByText('102 bindings')).toBeInTheDocument()
    expect(screen.getByText('2 files')).toHaveAttribute(
      'title',
      'config/sodium-options.json\nconfig/sodium-extra-options.json',
    )
    expect(screen.getByText('Default')).toBeInTheDocument()
    expect(resetButton('Distant Horizons')).toBeDisabled()
    expect(resetButton('Game options')).toBeEnabled()
  })

  it('asks in place, resets on Confirm and takes Go’s list', async () => {
    const after = view([item('options', 'Game options', [])])
    vi.mocked(App.ResetGameSetting).mockResolvedValue(after as never)
    render(<GameSettingsCard chapter={frangfurd} />)
    fireEvent.click(resetButton('Game options'))
    expect(App.ResetGameSetting).not.toHaveBeenCalled()
    fireEvent.click(screen.getAllByRole('button', { name: 'Cancel' })[0]!)
    expect(screen.queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument()
    fireEvent.click(resetButton('Game options'))
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    expect(App.ResetGameSetting).toHaveBeenCalledWith('frangfurd', 'options')
    await screen.findByText('Default')
    expect(resetButton('Game options')).toBeDisabled()
  })

  it('shows Go’s refusal under the card', async () => {
    vi.mocked(App.ResetGameSetting).mockRejectedValue('Frangfurd is starting or running')
    render(<GameSettingsCard chapter={frangfurd} />)
    fireEvent.click(resetButton('Key bindings'))
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    expect(await screen.findByText('Frangfurd is starting or running')).toBeInTheDocument()
  })

  it('disables every reset while the game runs', () => {
    useGameStore.setState({
      states: {
        frangfurd: { chapterId: 'frangfurd', phase: 'running', since: '', startedAt: '' },
      },
    })
    render(<GameSettingsCard chapter={frangfurd} />)
    expect(resetButton('Game options')).toBeDisabled()
    expect(resetButton('Key bindings')).toBeDisabled()
  })

  it('draws nothing for a chapter with nothing to reset or without a bridge', () => {
    useGameSettingsStore.setState({ byChapter: { frangfurd: view([]) } })
    vi.mocked(App.GetGameSettings).mockResolvedValue(view([]) as never)
    const { container } = render(<GameSettingsCard chapter={frangfurd} />)
    expect(container).toBeEmptyDOMElement()
    cleanup()
    useGameSettingsStore.setState({ unavailable: true })
    const second = render(<GameSettingsCard chapter={frangfurd} />)
    expect(second.container).toBeEmptyDOMElement()
  })
})
