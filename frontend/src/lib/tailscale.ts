import type { TailscaleInfo, TailscaleInstallProgress } from '../types/tailscale'
import { megabytes } from './prismInstall'

/** What the Tailscale row's button does now, or nothing. */
export type TailscaleAction = 'get' | 'open' | 'signIn' | null

/** The button the player's Tailscale calls for: get it, start it or sign in. */
export function tailscaleAction(info: TailscaleInfo | null): TailscaleAction {
  if (info === null || info.signingIn) return null
  switch (info.state) {
    case 'missing':
      return 'get'
    case 'stopped':
      return 'open'
    case 'needsLogin':
      return 'signIn'
    default:
      return null
  }
}

/** The line under the Tailscale switch: where the player's Tailscale stands. */
export function tailscaleHint(info: TailscaleInfo | null, on: boolean): string {
  if (info === null) return 'Checking Tailscale'
  if (info.signingIn) return 'Finish signing in to Tailscale in your browser'
  switch (info.state) {
    case 'missing':
      return 'Joins through Tailscale. Tailscale is not installed yet'
    case 'stopped':
      return 'Tailscale is installed but not running'
    case 'starting':
      return 'Tailscale is starting'
    case 'needsLogin':
      return 'Sign in to Tailscale to reach the server'
    case 'needsApproval':
      return "Waiting for the server's owner to approve this device"
    case 'running':
      return on
        ? 'Connected · in the in-game server list'
        : 'Connected · adds the server to the in-game server list'
    default:
      return 'Tailscale could not be read'
  }
}

/** The step line of a Tailscale install. */
export function tailscaleStepLine(p: TailscaleInstallProgress): string {
  switch (p.phase) {
    case 'downloading':
      return `Downloading ${megabytes(p.received)} of ${megabytes(p.total)}`
    case 'verifying':
      return "Checking Tailscale's installer"
    case 'installing':
      return "Tailscale's installer is open. Finish it there"
    default:
      return ''
  }
}
