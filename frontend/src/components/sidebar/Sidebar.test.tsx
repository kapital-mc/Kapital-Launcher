import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, screen } from '@testing-library/react'
import { DEFAULT_SETTINGS, useSettingsStore } from '../../stores/useSettingsStore'
import { Sidebar } from './Sidebar'

vi.mock('../../../wailsjs/go/main/App')
vi.mock('../../../wailsjs/runtime/runtime')

describe('Sidebar', () => {
  afterEach(() => {
    cleanup()
    useSettingsStore.setState({ settings: DEFAULT_SETTINGS, loaded: false })
  })

  const aside = () => screen.getByRole('complementary')

  it('takes the saved width without a transition, and animates a change after that', async () => {
    useSettingsStore.setState({
      settings: { ...DEFAULT_SETTINGS, sidebarCompact: true },
      loaded: true,
    })
    render(<Sidebar />)
    expect(aside()).toHaveClass('w-(--layout-sidebar-compact)')
    expect(aside()).not.toHaveClass('transition-[width,padding]')
    // Two frames later the width is settled and a change is a transition.
    await act(() => new Promise((r) => setTimeout(r, 50)))
    expect(aside()).toHaveClass('transition-[width,padding]')
    act(() => useSettingsStore.setState({ settings: DEFAULT_SETTINGS }))
    expect(aside()).toHaveClass('w-(--layout-sidebar)')
  })
})
