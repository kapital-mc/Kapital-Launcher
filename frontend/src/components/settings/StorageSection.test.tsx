import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import * as App from '../../../wailsjs/go/main/App'
import { useGameStore } from '../../stores/useGameStore'
import { useStorageStore } from '../../stores/useStorageStore'
import { BUNDLED_MANIFEST } from '../../lib/manifest'
import type { ChapterStorage, StorageItem } from '../../types'
import { StorageSection } from './StorageSection'

vi.mock('../../../wailsjs/go/main/App')

const frangfurd = BUNDLED_MANIFEST.chapters[2]!
const GB = 1024 ** 3
const MB = 1024 ** 2

const terrain: StorageItem = {
  kind: 'terrain',
  name: '1790374988_frangfurd',
  label: '1790374988_frangfurd',
  bytes: 13 * GB,
  files: 3,
  lastModified: '2026-10-20T19:40:00Z',
}
const shots: StorageItem = {
  kind: 'screenshots',
  name: '',
  label: '',
  bytes: 215 * MB,
  files: 40,
  lastModified: '',
}
const mods: StorageItem = { ...shots, kind: 'mods', bytes: 400 * MB, files: 300 }

const view = (items: StorageItem[], running = false): ChapterStorage => ({
  chapterId: 'frangfurd',
  totalBytes: items.reduce((n, i) => n + i.bytes, 0),
  freeBytes: 50 * GB,
  partial: false,
  running,
  items,
})

describe('StorageSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useStorageStore.setState({ byChapter: {}, unavailable: false, readError: {}, clearing: null })
    useGameStore.setState({ states: {} })
    vi.mocked(App.GetChapterStorage).mockResolvedValue(view([terrain, shots, mods]) as never)
  })
  afterEach(() => {
    cleanup()
    Reflect.deleteProperty(window, 'go')
  })

  it('lists the instance by kind with the total and the disk’s free space', async () => {
    render(<StorageSection chapter={frangfurd} />)
    expect(await screen.findByText('Distant Horizons terrain: frangfurd')).toBeInTheDocument()
    expect(screen.getByText('14 GB')).toBeInTheDocument()
    expect(screen.getByText('215 MB · 40 pictures')).toBeInTheDocument()
    expect(screen.getByText('50 GB free on this disk.')).toBeInTheDocument()
    // One delete (the terrain) and one way to the screenshots; mods have neither.
    expect(screen.getAllByRole('button', { name: 'Delete' })).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Open folder' })).toBeInTheDocument()
  })

  it('deletes only after the player confirms, and shows Go’s new measure', async () => {
    vi.mocked(App.ClearChapterStorage).mockResolvedValue(view([shots, mods]) as never)
    render(<StorageSection chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Delete' }))
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
    expect(
      screen.getByText(
        'Delete 13 GB of terrain for frangfurd? It comes back from the server as you play.',
      ),
    ).toBeInTheDocument()
    const buttons = screen.getAllByRole('button', { name: 'Delete' })
    await act(async () => fireEvent.click(buttons[buttons.length - 1]!))
    expect(App.ClearChapterStorage).toHaveBeenCalledWith(
      'frangfurd',
      'terrain',
      '1790374988_frangfurd',
    )
    await waitFor(() =>
      expect(screen.queryByText('Distant Horizons terrain: frangfurd')).not.toBeInTheDocument(),
    )
    // The World data card is read again.
    expect(App.GetDHSnapshot).toHaveBeenCalledWith('frangfurd')
  })

  it('cancels a confirmation without deleting', async () => {
    render(<StorageSection chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Delete' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull())
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
  })

  it('shows Go’s refusal', async () => {
    vi.mocked(App.ClearChapterStorage).mockRejectedValue('Frangfurd is starting or running')
    render(<StorageSection chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Delete' }))
    const buttons = screen.getAllByRole('button', { name: 'Delete' })
    await act(async () => fireEvent.click(buttons[buttons.length - 1]!))
    expect(await screen.findByRole('alert')).toHaveTextContent('Frangfurd is starting or running')
  })

  it('holds the deletes while the game runs, and hints at the log’s guess', async () => {
    useGameStore.setState({
      states: {
        frangfurd: {
          chapterId: 'frangfurd',
          phase: 'running',
          since: '2026-10-02T10:00:00Z',
          startedAt: '2026-10-02T09:59:00Z',
        },
      },
    })
    render(<StorageSection chapter={frangfurd} />)
    expect(await screen.findByRole('button', { name: 'Delete' })).toBeDisabled()
    expect(screen.getByText('Close the game to delete anything.')).toBeInTheDocument()
    cleanup()
    useGameStore.setState({ states: {} })
    vi.mocked(App.GetChapterStorage).mockResolvedValue(view([terrain], true) as never)
    render(<StorageSection chapter={frangfurd} />)
    expect(await screen.findByText(/may be running/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeEnabled()
  })

  it('opens the screenshots folder through Go, and shows a refusal', async () => {
    render(<StorageSection chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Open folder' }))
    expect(App.OpenScreenshotsFolder).toHaveBeenCalledWith('frangfurd')
    vi.mocked(App.OpenScreenshotsFolder).mockRejectedValue('Frangfurd has no screenshots yet')
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Open folder' })))
    expect(await screen.findByRole('alert')).toHaveTextContent('Frangfurd has no screenshots yet')
  })

  it('shows a read refusal, and nothing without a bridge', async () => {
    vi.mocked(App.GetChapterStorage).mockRejectedValue('Frangfurd is not installed')
    render(<StorageSection chapter={frangfurd} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Frangfurd is not installed')
    cleanup()
    Reflect.deleteProperty(window, 'go')
    const { container } = render(<StorageSection chapter={frangfurd} />)
    await waitFor(() => expect(useStorageStore.getState().unavailable).toBe(true))
    expect(container).toBeEmptyDOMElement()
  })
})
