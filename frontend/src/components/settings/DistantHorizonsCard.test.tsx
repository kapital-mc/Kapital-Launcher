import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import * as App from '../../../wailsjs/go/main/App'
import * as Runtime from '../../../wailsjs/runtime/runtime'
import { BUNDLED_MANIFEST } from '../../lib/manifest'
import { useGameStore } from '../../stores/useGameStore'
import { useTerrainStore } from '../../stores/useTerrainStore'
import type { DHSnapshotInfo } from '../../types'
import { DistantHorizonsCard } from './DistantHorizonsCard'

vi.mock('../../../wailsjs/go/main/App')
vi.mock('../../../wailsjs/runtime/runtime')

const frangfurd = BUNDLED_MANIFEST.chapters.find((c) => c.id === 'frangfurd')!
const published: DHSnapshotInfo = {
  chapterId: 'frangfurd',
  available: true,
  reason: '',
  createdAt: '2026-10-13T12:03:41Z',
  sizeBytes: 6 * 1024 ** 3,
  dimensions: 11,
  serverDhVersion: '3.3.3',
  clientDhVersion: '3.3.3',
  existingBytes: 0,
  partialBytes: 0,
  running: false,
  downloading: false,
}

describe('The Distant Horizons terrain card', () => {
  let hear: (p: unknown) => void
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useTerrainStore.setState({
      byChapter: {},
      progress: {},
      unavailable: false,
      readError: {},
      cancelling: {},
    })
    useGameStore.setState({ states: {} })
    vi.mocked(App.GetDHSnapshot).mockResolvedValue(published as never)
    vi.mocked(Runtime.EventsOn).mockImplementation(((_: string, cb: (p: unknown) => void) => {
      hear = cb
      return () => undefined
    }) as never)
    // The sidebar's downloads section holds the listener for the app's lifetime.
    useTerrainStore.getState().listen()
  })
  afterEach(() => {
    cleanup()
    Reflect.deleteProperty(window, 'go')
  })

  it('says what is published before anything downloads', async () => {
    render(<DistantHorizonsCard chapter={frangfurd} />)
    expect(await screen.findByText(/6\.0 GB · 11 dimensions/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download' })).toBeEnabled()
    expect(App.DownloadDHSnapshot).not.toHaveBeenCalled()
  })

  it('downloads on click, follows the progress and says when it is done', async () => {
    let finish!: (v: unknown) => void
    vi.mocked(App.DownloadDHSnapshot).mockReturnValueOnce(new Promise((r) => (finish = r)) as never)
    render(<DistantHorizonsCard chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Download' }))
    expect(App.DownloadDHSnapshot).toHaveBeenCalledWith('frangfurd', false)
    act(() =>
      hear({
        chapterId: 'frangfurd',
        phase: 'downloading',
        received: 1024 ** 3,
        total: 4 * 1024 ** 3,
      }),
    )
    const bar = screen.getByRole('progressbar', { name: 'Terrain download' })
    expect(bar).toHaveAttribute('aria-valuenow', '25')
    expect(bar.firstElementChild).toHaveStyle({ '--progress-fill': '0.25' })
    expect(screen.getByText('Downloading · 1.0 GB of 4.0 GB · 25%')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    await act(async () => finish({ ...published, existingBytes: published.sizeBytes }))
    expect(screen.getByText(/The terrain is in place/)).toBeInTheDocument()
    expect(screen.queryByRole('progressbar')).toBeNull()
  })

  it('asks before replacing terrain the instance has, and keeps it on Keep mine', async () => {
    vi.mocked(App.GetDHSnapshot).mockResolvedValue({ ...published, existingBytes: 2048 } as never)
    vi.mocked(App.DownloadDHSnapshot).mockResolvedValue(published as never)
    render(<DistantHorizonsCard chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Download' }))
    expect(screen.getByText(/already has 2\.0 KB of terrain/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Keep mine' }))
    expect(App.DownloadDHSnapshot).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    fireEvent.click(screen.getByRole('button', { name: 'Replace' }))
    await waitFor(() => expect(App.DownloadDHSnapshot).toHaveBeenCalledWith('frangfurd', true))
  })

  it('cancels, keeping what was downloaded, and offers to continue', async () => {
    let refuse!: (e: Error) => void
    vi.mocked(App.DownloadDHSnapshot).mockReturnValueOnce(
      new Promise((_, r) => (refuse = r)) as never,
    )
    vi.mocked(App.CancelDHSnapshot).mockResolvedValue(true)
    render(<DistantHorizonsCard chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Download' }))
    vi.mocked(App.GetDHSnapshot).mockResolvedValue({ ...published, partialBytes: 1024 } as never)
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(App.CancelDHSnapshot).toHaveBeenCalledWith('frangfurd'))
    await act(async () => refuse(new Error('the download was cancelled')))
    expect(await screen.findByText(/Cancelled\. What was downloaded is kept/)).toBeInTheDocument()
    expect(screen.queryByRole('alert')).toBeNull()
    expect(await screen.findByRole('button', { name: 'Continue' })).toBeEnabled()
    expect(screen.getByText(/1\.0 KB already downloaded/)).toBeInTheDocument()
  })

  it("shows Go's refusal", async () => {
    vi.mocked(App.DownloadDHSnapshot).mockRejectedValueOnce(new Error('not enough free space'))
    render(<DistantHorizonsCard chapter={frangfurd} />)
    fireEvent.click(await screen.findByRole('button', { name: 'Download' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('not enough free space')
  })

  it('holds Download while the game runs, and says why Go would refuse', async () => {
    vi.mocked(App.GetDHSnapshot).mockResolvedValue({
      ...published,
      available: false,
      running: true,
      reason: 'Close the game first: Distant Horizons keeps its terrain open while it runs.',
    } as never)
    render(<DistantHorizonsCard chapter={frangfurd} />)
    expect(await screen.findByText(/Close the game first/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download' })).toBeDisabled()
  })

  it('reads again when the window comes back', async () => {
    render(<DistantHorizonsCard chapter={frangfurd} />)
    await screen.findByRole('button', { name: 'Download' })
    const before = vi.mocked(App.GetDHSnapshot).mock.calls.length
    act(() => void window.dispatchEvent(new Event('focus')))
    await waitFor(() => expect(App.GetDHSnapshot).toHaveBeenCalledTimes(before + 1))
  })

  it('says so where there is no launcher behind the page', () => {
    Reflect.deleteProperty(window, 'go')
    render(<DistantHorizonsCard chapter={frangfurd} />)
    expect(screen.getByText('Not available here')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Download' })).toBeDisabled()
  })
})
