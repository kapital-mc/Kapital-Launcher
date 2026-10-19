import { describe, expect, it } from 'vitest'
import { tailscaleStep, tailscaleStepLine } from './tailscale'
import type { TailscaleInfo } from '../types/tailscale'

const info = (state: string, connected: string[] = []): TailscaleInfo => ({
  state,
  version: '1.104.1',
  canInstall: true,
  signingIn: false,
  connected,
})

describe('Tailscale lines', () => {
  it('offers Install, Connect or the switch', () => {
    expect(tailscaleStep(null, 'frangfurd')).toBeNull()
    expect(tailscaleStep(info('missing'), 'frangfurd')).toBe('install')
    for (const s of ['stopped', 'starting', 'needsLogin', 'needsApproval', 'odd']) {
      expect(tailscaleStep(info(s), 'frangfurd')).toBe('connect')
    }
    expect(tailscaleStep(info('running'), 'frangfurd')).toBe('connect')
    expect(tailscaleStep(info('running', ['lichdenstein']), 'frangfurd')).toBe('connect')
    expect(tailscaleStep(info('running', ['frangfurd']), 'frangfurd')).toBe('ready')
  })

  it('describes each install step', () => {
    const p = { received: 1024 * 1024, total: 2 * 1024 * 1024, error: '' }
    expect(tailscaleStepLine({ ...p, phase: 'downloading' })).toBe('Downloading 1.0 MB of 2.0 MB')
    expect(tailscaleStepLine({ ...p, phase: 'verifying' })).toContain('Checking')
    expect(tailscaleStepLine({ ...p, phase: 'installing' })).toContain('installer is open')
    expect(tailscaleStepLine({ ...p, phase: 'done' })).toBe('')
  })
})
