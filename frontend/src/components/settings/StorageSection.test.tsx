import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import * as App from '../../../wailsjs/go/main/App'
import { useGameStore } from '../../stores/useGameStore'
import { DEFAULT_SETTINGS, useSettingsStore } from '../../stores/useSettingsStore'
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
const world: StorageItem = { ...terrain, kind: 'saves', name: 'Convert', label: '', bytes: 90 * MB }

/** Opens a folded group by its heading. */
const unfold = async (title: string) =>
  fireEvent.click(await screen.findByRole('button', { name: new RegExp(`^${title}`) }))

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
    useSettingsStore.setState({ settings: DEFAULT_SETTINGS })
    vi.mocked(App.GetChapterStorage).mockResolvedValue(view([terrain, shots, mods]) as never)
  })
  afterEach(() => {
    cleanup()
    Reflect.deleteProperty(window, 'go')
  })

  it('splits the instance into coloured groups, with the total and the disk’s free space', async () => {
    render(<StorageSection chapter={frangfurd} />)
    expect(await screen.findByText('Distant Horizons')).toBeInTheDocument()
    await unfold('Distant Horizons')
    expect(screen.getByText('frangfurd')).toBeInTheDocument()
    expect(screen.getByText('14 GB')).toBeInTheDocument()
    expect(screen.getByText('215 MB · 40 pictures')).toBeInTheDocument()
    expect(screen.getByText('50 GB free on disk')).toBeInTheDocument()
    expect(screen.getByRole('img')).toHaveAccessibleName(
      'Distant Horizons 13 GB, Screenshots 215 MB, Mods 400 MB',
    )
    // The terrain and the screenshots can go; the mods cannot. Every row has its folder.
    expect(screen.getAllByRole('button', { name: 'Delete' })).toHaveLength(2)
    expect(screen.getAllByRole('button', { name: /^Open the .* folder$/ })).toHaveLength(3)
  })

  it('turns Delete into Confirm in place, and shows Go’s new measure', async () => {
    vi.mocked(App.ClearChapterStorage).mockResolvedValue(view([shots, mods]) as never)
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Distant Horizons')
    fireEvent.click((await screen.findAllByRole('button', { name: 'Delete' }))[0]!)
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
    // Only the screenshots' Delete is left; the terrain's is Confirm now.
    expect(screen.getAllByRole('button', { name: 'Delete' })).toHaveLength(1)
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    // Terrain asks once more, as screenshots do.
    const dialog = await screen.findByRole('dialog', { name: 'Are you sure?' })
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
    await act(async () => fireEvent.click(within(dialog).getByRole('button', { name: 'Confirm' })))
    expect(App.ClearChapterStorage).toHaveBeenCalledWith(
      'frangfurd',
      'terrain',
      '1790374988_frangfurd',
    )
    await waitFor(() => expect(screen.queryByText('Distant Horizons')).not.toBeInTheDocument())
    // The World data card is read again.
    expect(App.GetDHSnapshot).toHaveBeenCalledWith('frangfurd')
  })

  it('does not take the second click of a double click as the confirmation', async () => {
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Distant Horizons')
    const trash = (await screen.findAllByRole('button', { name: 'Delete' }))[0]!
    fireEvent.click(trash, { detail: 1 })
    fireEvent.click(trash, { detail: 2 })
    expect(trash).toHaveAccessibleName('Confirm')
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
  })

  it('cancels a confirmation without deleting', async () => {
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Distant Horizons')
    fireEvent.click((await screen.findAllByRole('button', { name: 'Delete' }))[1]!)
    // Every deletable row holds a Cancel, folded away and inert until its row asks.
    const cancel = screen
      .getAllByRole('button', { name: 'Cancel' })
      .filter((b) => !b.closest('[inert]'))
    expect(cancel).toHaveLength(1)
    fireEvent.click(cancel[0]!)
    expect(screen.queryByRole('button', { name: 'Confirm' })).toBeNull()
    expect(screen.getByText('215 MB · 40 pictures')).toBeInTheDocument()
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
  })

  it('asks once more before deleting screenshots, and can stop asking', async () => {
    render(<StorageSection chapter={frangfurd} />)
    const shotsTrash = () => screen.getByRole('button', { name: 'Delete' })
    await screen.findByText('Screenshots')
    fireEvent.click(shotsTrash())
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    const dialog = await screen.findByRole('dialog', { name: 'Are you sure?' })
    expect(dialog).toHaveTextContent('This will permanently delete your files.')
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()

    fireEvent.click(shotsTrash())
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    const again = await screen.findByRole('dialog', { name: 'Are you sure?' })
    fireEvent.click(within(again).getByRole('checkbox', { name: "Don't ask me again" }))
    await act(async () => fireEvent.click(within(again).getByRole('button', { name: 'Confirm' })))
    expect(App.ClearChapterStorage).toHaveBeenCalledWith('frangfurd', 'screenshots', '')
    expect(App.SaveSettings).toHaveBeenCalledWith(
      expect.objectContaining({ skipDeleteConfirm: true }),
    )
  })

  it('deletes at once when the player stopped the popup', async () => {
    useSettingsStore.setState({ settings: { ...DEFAULT_SETTINGS, skipDeleteConfirm: true } })
    render(<StorageSection chapter={frangfurd} />)
    await screen.findByText('Screenshots')
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Confirm' })))
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(App.ClearChapterStorage).toHaveBeenCalledWith('frangfurd', 'screenshots', '')
  })

  it('shows Go’s refusal', async () => {
    vi.mocked(App.ClearChapterStorage).mockRejectedValue('Frangfurd is starting or running')
    useSettingsStore.setState({ settings: { ...DEFAULT_SETTINGS, skipDeleteConfirm: true } })
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Distant Horizons')
    fireEvent.click((await screen.findAllByRole('button', { name: 'Delete' }))[0]!)
    await act(async () => fireEvent.click(screen.getByRole('button', { name: 'Confirm' })))
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
    await unfold('Distant Horizons')
    for (const b of await screen.findAllByRole('button', { name: 'Delete' }))
      expect(b).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Open the frangfurd folder' })).toBeEnabled()
    expect(screen.getByText('Close the game to delete anything.')).toBeInTheDocument()
    cleanup()
    useGameStore.setState({ states: {} })
    vi.mocked(App.GetChapterStorage).mockResolvedValue(view([terrain], true) as never)
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Distant Horizons')
    expect(await screen.findByText(/may be running/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeEnabled()
  })

  it('opens a row’s folder through Go, and shows a refusal', async () => {
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Distant Horizons')
    fireEvent.click(await screen.findByRole('button', { name: 'Open the frangfurd folder' }))
    expect(App.OpenStorageFolder).toHaveBeenCalledWith(
      'frangfurd',
      'terrain',
      '1790374988_frangfurd',
    )
    vi.mocked(App.OpenStorageFolder).mockRejectedValue('Frangfurd: that folder is not there yet')
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: 'Open the Screenshots folder' })),
    )
    expect(App.OpenStorageFolder).toHaveBeenLastCalledWith('frangfurd', 'screenshots', '')
    expect(await screen.findByRole('alert')).toHaveTextContent('that folder is not there yet')
  })

  it('folds the groups at first, and lists each save with its icon, folder and delete', async () => {
    const icon = 'data:image/png;base64,iVBORw0KGgo='
    vi.mocked(App.GetChapterStorage).mockResolvedValue(
      view([terrain, { ...world, icon }, shots]) as never,
    )
    const { container } = render(<StorageSection chapter={frangfurd} />)
    const saves = await screen.findByRole('button', { name: /^Saves/ })
    expect(saves).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('frangfurd')).toBeNull()
    expect(screen.queryByText('Convert')).toBeNull()
    fireEvent.click(saves)
    expect(saves).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Convert')).toBeInTheDocument()
    expect(container.querySelector('img')).toHaveAttribute('src', icon)
    // The save's and the screenshots' Delete: the terrain's group is still folded.
    expect(screen.getAllByRole('button', { name: 'Delete' })).toHaveLength(2)
    fireEvent.click(screen.getByRole('button', { name: 'Open the Convert folder' }))
    expect(App.OpenStorageFolder).toHaveBeenCalledWith('frangfurd', 'saves', 'Convert')
  })

  it('always asks before deleting a world, without "Don\'t ask me again"', async () => {
    useSettingsStore.setState({ settings: { ...DEFAULT_SETTINGS, skipDeleteConfirm: true } })
    vi.mocked(App.GetChapterStorage).mockResolvedValue(view([world, shots]) as never)
    render(<StorageSection chapter={frangfurd} />)
    await unfold('Saves')
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[0]!)
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    const dialog = await screen.findByRole('dialog', { name: 'Are you sure?' })
    expect(within(dialog).queryByRole('checkbox')).toBeNull()
    expect(App.ClearChapterStorage).not.toHaveBeenCalled()
    await act(async () => fireEvent.click(within(dialog).getByRole('button', { name: 'Confirm' })))
    expect(App.ClearChapterStorage).toHaveBeenCalledWith('frangfurd', 'saves', 'Convert')
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

  describe('before the first read answers', () => {
    const card = () => screen.getByRole('region', { name: 'Storage' }).querySelector('.rounded-lg')
    const pending = () => document.querySelector('[class~="h-(--layout-pending)"]')

    it("holds the section's place with an empty card, which the rows then come into", async () => {
      let answer: (v: Awaited<ReturnType<typeof App.GetChapterStorage>>) => void = () => undefined
      vi.mocked(App.GetChapterStorage).mockReturnValue(new Promise((r) => (answer = r)))
      render(<StorageSection chapter={frangfurd} />)
      expect(screen.getByRole('heading', { level: 2, name: 'Storage' })).toBeInTheDocument()
      expect(pending()).toHaveAttribute('aria-hidden', 'true')
      const before = card()
      await act(async () => answer(view([shots, mods]) as never))
      expect(screen.getByText('Screenshots')).toBeInTheDocument()
      expect(pending()).toBeNull()
      // The card the rows come into is the one that held their place, so it grows into them.
      expect(card()).toBe(before)
    })

    it('draws a view kept from an earlier visit at once, and reads it again', () => {
      useStorageStore.setState({ byChapter: { frangfurd: view([shots, mods]) } })
      render(<StorageSection chapter={frangfurd} />)
      expect(screen.getByText('Screenshots')).toBeInTheDocument()
      expect(pending()).toBeNull()
      expect(App.GetChapterStorage).toHaveBeenCalledWith('frangfurd')
    })
  })
})
