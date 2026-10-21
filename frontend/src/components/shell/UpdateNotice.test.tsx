import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import * as Bindings from '../../../wailsjs/go/main/App'
import {
  selectUpdateReady,
  selectUpdateShown,
  selectUpdateWorking,
  useUpdateStore,
} from '../../stores/useUpdateStore'
import type { UpdateInfo } from '../../types'
import { UpdateNotice } from './UpdateNotice'

vi.mock('../../../wailsjs/go/main/App')
vi.mock('../../../wailsjs/runtime/runtime')

const newer: UpdateInfo = {
  current: 'v1.0.0-beta.2',
  latest: 'v1.0.0-beta.3',
  available: true,
  page: 'https://kapital-launcher.pages.dev/',
  reason: '',
  installable: false,
  blocked: '',
}

const installable: UpdateInfo = { ...newer, installable: true }

const step = (phase: string, received = 0, total = 0) => ({
  phase,
  received,
  total,
  version: newer.latest,
  error: '',
})

describe('useUpdateStore', () => {
  beforeEach(() => {
    useUpdateStore.setState({ update: null, dismissed: false, error: null, progress: null })
    vi.mocked(Bindings.GetUpdate).mockReset()
    vi.mocked(Bindings.OpenExternal).mockReset()
    vi.mocked(Bindings.DownloadUpdate).mockReset()
    vi.mocked(Bindings.CancelUpdateDownload).mockReset()
  })

  it('downloads in the background, puts the popup away and follows the steps to ready', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(installable)
    await useUpdateStore.getState().load()
    vi.mocked(Bindings.DownloadUpdate).mockResolvedValue()
    await useUpdateStore.getState().download()
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(false)
    const { receive } = useUpdateStore.getState()
    receive(step('downloading', 5, 10))
    expect(selectUpdateWorking(useUpdateStore.getState())).toBe(true)
    receive(step('ready'))
    // A late step does not undo ready.
    receive(step('downloading', 10, 10))
    expect(selectUpdateReady(useUpdateStore.getState())).toBe(true)
  })

  it('keeps a failed download with its reason, and a cancel as no failure', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(installable)
    await useUpdateStore.getState().load()
    vi.mocked(Bindings.DownloadUpdate).mockRejectedValueOnce('no answer')
    await useUpdateStore.getState().download()
    expect(useUpdateStore.getState().progress).toMatchObject({
      phase: 'failed',
      error: 'no answer',
    })

    let reject: (e: unknown) => void = () => {}
    vi.mocked(Bindings.DownloadUpdate).mockReturnValueOnce(
      new Promise<void>((_, r) => {
        reject = r
      }),
    )
    const running = useUpdateStore.getState().download()
    vi.mocked(Bindings.CancelUpdateDownload).mockResolvedValue(true)
    await useUpdateStore.getState().cancelDownload()
    reject('the download was cancelled')
    await running
    expect(useUpdateStore.getState().progress).toBeNull()
  })

  it('shows the popup for a newer build until it is put away', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(newer)
    await useUpdateStore.getState().load()
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(true)
    useUpdateStore.getState().dismiss()
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(false)
  })

  it('shows nothing when this build is current, unknown, or there is no bridge', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue({ ...newer, available: false })
    await useUpdateStore.getState().load()
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(false)
    vi.mocked(Bindings.GetUpdate).mockRejectedValue('no bridge')
    await useUpdateStore.getState().load()
    expect(useUpdateStore.getState().update).toBeNull()
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(false)
  })

  it('opens the page Go named and keeps the popup when that fails', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(newer)
    await useUpdateStore.getState().load()
    vi.mocked(Bindings.OpenExternal).mockRejectedValue('no window')
    await useUpdateStore.getState().openPage()
    expect(Bindings.OpenExternal).toHaveBeenCalledWith(newer.page)
    expect(useUpdateStore.getState().error).toBe('no window')
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(true)
    vi.mocked(Bindings.OpenExternal).mockResolvedValue()
    await useUpdateStore.getState().openPage()
    expect(selectUpdateShown(useUpdateStore.getState())).toBe(false)
  })
})

describe('UpdateNotice', () => {
  beforeEach(() => {
    useUpdateStore.setState({ update: null, dismissed: false, error: null, progress: null })
    vi.mocked(Bindings.GetUpdate).mockReset().mockResolvedValue(newer)
    vi.mocked(Bindings.OpenExternal).mockReset()
    vi.mocked(Bindings.DownloadUpdate).mockReset().mockResolvedValue()
    vi.mocked(Bindings.RestartToUpdate).mockReset()
  })
  afterEach(cleanup)

  it('offers Update where this copy can update itself, and Restart once it is ready', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(installable)
    render(<UpdateNotice size="sm" />)
    await act(async () => {
      fireEvent.click(await screen.findByRole('button', { name: 'Update' }))
    })
    expect(Bindings.DownloadUpdate).toHaveBeenCalledOnce()
    expect(screen.queryByRole('dialog')).toBeNull()
    act(() => useUpdateStore.getState().receive(step('ready')))
    fireEvent.click(screen.getByRole('button', { name: 'Restart to Update' }))
    expect(screen.getByRole('dialog', { name: 'Update ready' })).toBeInTheDocument()
    vi.mocked(Bindings.RestartToUpdate).mockRejectedValue(
      'Wait until Frangfurd has finished loading',
    )
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Restart' }))
    })
    expect(Bindings.RestartToUpdate).toHaveBeenCalledOnce()
    expect(screen.getByText('Wait until Frangfurd has finished loading')).toBeInTheDocument()
  })

  it('shows the bar and Cancel while the download runs', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(installable)
    render(<UpdateNotice size="sm" />)
    await screen.findByRole('dialog')
    act(() => useUpdateStore.getState().receive(step('downloading', 1, 4)))
    expect(screen.getByRole('progressbar', { name: 'Kapital Launcher update' })).toHaveAttribute(
      'aria-valuenow',
      '25',
    )
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
  })

  it('says why a copy cannot update itself, and offers the site', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue({
      ...newer,
      blocked: 'move Kapital Launcher to your Applications folder and open it from there',
    })
    render(<UpdateNotice size="sm" />)
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent(
      'Move Kapital Launcher to your Applications folder and open it from there. Install it over',
    )
    expect(screen.getByRole('button', { name: 'Download' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update' })).toBeNull()
  })

  it('names both versions and opens the download page', async () => {
    vi.mocked(Bindings.OpenExternal).mockResolvedValue()
    render(<UpdateNotice size="sm" />)
    const dialog = await screen.findByRole('dialog', { name: 'Update available' })
    expect(dialog).toHaveTextContent('v1.0.0-beta.3 · you have v1.0.0-beta.2')
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Download' }))
    })
    expect(Bindings.OpenExternal).toHaveBeenCalledWith('https://kapital-launcher.pages.dev/')
    expect(useUpdateStore.getState().dismissed).toBe(true)
  })

  it('goes away for this run on Not now, Escape and a press elsewhere, and the button brings it back', async () => {
    render(<UpdateNotice size="sm" />)
    fireEvent.click(await screen.findByRole('button', { name: 'Not now' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Update Launcher' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    act(() => useUpdateStore.getState().reopen())
    fireEvent.pointerDown(document.body)
    expect(screen.queryByRole('dialog')).toBeNull()
    act(() => useUpdateStore.getState().reopen())
    fireEvent.pointerDown(screen.getByRole('dialog'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(Bindings.OpenExternal).not.toHaveBeenCalled()
  })

  it('renders nothing when this build is current', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue({ ...newer, available: false })
    render(<UpdateNotice size="sm" />)
    await act(async () => {})
    expect(Bindings.GetUpdate).toHaveBeenCalledOnce()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('says why the page did not open', async () => {
    vi.mocked(Bindings.OpenExternal).mockRejectedValue('The window is not ready')
    render(<UpdateNotice size="sm" />)
    await act(async () => {
      fireEvent.click(await screen.findByRole('button', { name: 'Download' }))
    })
    expect(screen.getByText('The window is not ready')).toBeInTheDocument()
  })
})
