import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import * as Bindings from '../../../wailsjs/go/main/App'
import { selectUpdateShown, useUpdateStore } from '../../stores/useUpdateStore'
import type { UpdateInfo } from '../../types'
import { UpdateNotice } from './UpdateNotice'

vi.mock('../../../wailsjs/go/main/App')

const newer: UpdateInfo = {
  current: 'v1.0.0-beta.2',
  latest: 'v1.0.0-beta.3',
  available: true,
  page: 'https://kapital-launcher.pages.dev/',
  reason: '',
}

describe('useUpdateStore', () => {
  beforeEach(() => {
    useUpdateStore.setState({ update: null, dismissed: false, error: null })
    vi.mocked(Bindings.GetUpdate).mockReset()
    vi.mocked(Bindings.OpenExternal).mockReset()
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
    useUpdateStore.setState({ update: null, dismissed: false, error: null })
    vi.mocked(Bindings.GetUpdate).mockReset().mockResolvedValue(newer)
    vi.mocked(Bindings.OpenExternal).mockReset()
  })
  afterEach(cleanup)

  it('names both versions and opens the download page', async () => {
    vi.mocked(Bindings.OpenExternal).mockResolvedValue()
    render(<UpdateNotice />)
    const dialog = await screen.findByRole('dialog', { name: 'A new version is out' })
    expect(dialog).toHaveTextContent('v1.0.0-beta.3 · you have v1.0.0-beta.2')
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Open the download page' }))
    })
    expect(Bindings.OpenExternal).toHaveBeenCalledWith('https://kapital-launcher.pages.dev/')
    expect(useUpdateStore.getState().dismissed).toBe(true)
  })

  it('goes away for this run on Not now and on Escape', async () => {
    render(<UpdateNotice />)
    fireEvent.click(await screen.findByRole('button', { name: 'Not now' }))
    expect(screen.queryByRole('dialog')).toBeNull()
    act(() => useUpdateStore.setState({ dismissed: false }))
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(Bindings.OpenExternal).not.toHaveBeenCalled()
  })

  it('renders nothing when this build is current', async () => {
    vi.mocked(Bindings.GetUpdate).mockResolvedValue({ ...newer, available: false })
    render(<UpdateNotice />)
    await act(async () => {})
    expect(Bindings.GetUpdate).toHaveBeenCalledOnce()
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('says why the page did not open', async () => {
    vi.mocked(Bindings.OpenExternal).mockRejectedValue('The window is not ready')
    render(<UpdateNotice />)
    await act(async () => {
      fireEvent.click(await screen.findByRole('button', { name: 'Open the download page' }))
    })
    expect(screen.getByText('The window is not ready')).toBeInTheDocument()
  })
})
