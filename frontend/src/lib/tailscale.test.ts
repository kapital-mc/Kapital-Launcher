import { describe, expect, it } from 'vitest'
import { tailscaleAction, tailscaleHint, tailscaleStepLine } from './tailscale'
import type { TailscaleInfo } from '../types/tailscale'

const info = (state: string, signingIn = false): TailscaleInfo => ({
  state,
  version: '1.104.1',
  canInstall: true,
  signingIn,
})

describe('Tailscale lines', () => {
  it('names the next step for each state', () => {
    expect(tailscaleAction(null)).toBeNull()
    expect(tailscaleAction(info('missing'))).toBe('get')
    expect(tailscaleAction(info('stopped'))).toBe('open')
    expect(tailscaleAction(info('needsLogin'))).toBe('signIn')
    expect(tailscaleAction(info('needsLogin', true))).toBeNull()
    expect(tailscaleAction(info('running'))).toBeNull()
    expect(tailscaleAction(info('needsApproval'))).toBeNull()
  })

  it('says where Tailscale stands under the switch', () => {
    expect(tailscaleHint(null, false)).toBe('Checking Tailscale')
    expect(tailscaleHint(info('missing'), false)).toContain('not installed')
    expect(tailscaleHint(info('running'), true)).toBe('Connected · in the in-game server list')
    expect(tailscaleHint(info('running'), false)).toContain('adds the server')
    expect(tailscaleHint(info('needsLogin', true), false)).toContain('browser')
    for (const s of ['stopped', 'starting', 'needsLogin', 'needsApproval', 'odd']) {
      expect(tailscaleHint(info(s), false)).not.toBe('')
    }
  })

  it('describes each install step', () => {
    const p = { received: 1024 * 1024, total: 2 * 1024 * 1024, error: '' }
    expect(tailscaleStepLine({ ...p, phase: 'downloading' })).toBe('Downloading 1.0 MB of 2.0 MB')
    expect(tailscaleStepLine({ ...p, phase: 'verifying' })).toContain('Checking')
    expect(tailscaleStepLine({ ...p, phase: 'installing' })).toContain('installer is open')
    expect(tailscaleStepLine({ ...p, phase: 'done' })).toBe('')
  })
})
