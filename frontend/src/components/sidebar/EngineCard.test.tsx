import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { DEFAULT_SETTINGS, useSettingsStore } from '../../stores/useSettingsStore'
import { EngineCard } from './EngineCard'

describe('EngineCard', () => {
  afterEach(() => {
    cleanup()
    useSettingsStore.setState({ settings: DEFAULT_SETTINGS })
  })

  it('says the account is a Microsoft one through Prism, until the player plays offline', () => {
    useSettingsStore.setState({ settings: { ...DEFAULT_SETTINGS, profileName: 'Sandro_G' } })
    const { rerender } = render(<EngineCard />)
    expect(screen.getByText('Microsoft account · via Prism')).toBeInTheDocument()
    useSettingsStore.setState({
      settings: {
        ...DEFAULT_SETTINGS,
        profileName: 'Sandro_G',
        offline: true,
        offlineName: 'Steve',
      },
    })
    rerender(<EngineCard />)
    expect(screen.getByText('Offline')).toBeInTheDocument()
    expect(screen.queryByText('Microsoft account · via Prism')).toBeNull()
  })

  it('folds to the initials when compact, which still open the account page', () => {
    useSettingsStore.setState({ settings: { ...DEFAULT_SETTINGS, profileName: 'Sandro_G' } })
    const onOpenAccount = vi.fn()
    render(<EngineCard compact onOpenAccount={onOpenAccount} />)
    // The name, Re-detect and the engine line are folded away and out of reach.
    expect(screen.getByText('Sandro_G').closest('[inert]')).not.toBeNull()
    expect(
      screen.getByRole('button', { name: 'Re-detect Prism', hidden: true }).closest('[inert]'),
    ).not.toBeNull()
    expect(screen.getByText('engine').closest('[inert]')).not.toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Account, Sandro_G' }))
    expect(onOpenAccount).toHaveBeenCalledOnce()
  })

  it('shows everything when not compact', () => {
    render(<EngineCard />)
    expect(document.querySelector('[inert]')).toBeNull()
  })
})
