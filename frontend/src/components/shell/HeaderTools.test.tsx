import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import * as Bindings from '../../../wailsjs/go/main/App'
import { useEngineStore } from '../../stores/useEngineStore'
import { useUpdateStore } from '../../stores/useUpdateStore'
import { useChapterStore } from '../../stores/useChapterStore'
import { HeaderTools } from './HeaderTools'

vi.mock('../../../wailsjs/go/main/App')
vi.mock('../../../wailsjs/runtime/runtime')

describe('HeaderTools', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(Bindings.OpenPrism).mockResolvedValue()
    vi.mocked(Bindings.OpenExternal).mockResolvedValue()
    useEngineStore.setState({ engine: null })
    useUpdateStore.setState({ update: null, dismissed: false, error: null })
  })
  afterEach(cleanup)

  const tools = () =>
    render(<HeaderTools size="sm" sidebarCompact={false} onOpenSettings={() => undefined} />)

  it('opens Prism only once there is one, and the wiki home page', () => {
    tools()
    expect(screen.getByRole('button', { name: 'Open Prism' })).toBeDisabled()
    act(() =>
      useEngineStore.setState({
        engine: { found: true, executable: 'prism', version: '11.1.0', root: '', source: 'path' },
      }),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Open Prism' }))
    expect(Bindings.OpenPrism).toHaveBeenCalledOnce()
    fireEvent.click(screen.getByRole('button', { name: 'Open Wiki' }))
    expect(Bindings.OpenExternal).toHaveBeenCalledWith(
      useChapterStore.getState().manifest.wiki.baseUrl,
    )
    expect(screen.getByRole('button', { name: 'Settings' })).toBeEnabled()
  })

  it('shows the update only while one is out, and brings its panel back', async () => {
    const out = {
      current: 'v1.0.0',
      latest: 'v1.1.0',
      available: true,
      page: '',
      reason: '',
      installable: false,
      blocked: '',
    }
    vi.mocked(Bindings.GetUpdate).mockResolvedValue(out as never)
    tools()
    expect(screen.queryByRole('button', { name: /^Update Launcher/ })).toBeNull()
    const button = await screen.findByRole('button', { name: 'Update Launcher' })
    expect(screen.getByRole('dialog', { name: 'Update available' })).toBeInTheDocument()
    act(() => useUpdateStore.getState().dismiss())
    fireEvent.click(button)
    expect(useUpdateStore.getState().dismissed).toBe(false)
  })
})
