import type { TailscaleInfo, TailscaleInstallProgress } from '../types/tailscale'
import { megabytes } from './prismInstall'

/** What the Tailscale row of a chapter offers: Install, Connect, the switch, or nothing yet. */
export type TailscaleStep = 'install' | 'connect' | 'ready' | null

/**
 * The row's step for the player's Tailscale and this chapter's server: Install
 * while Tailscale is missing, the switch once it reaches the server, Connect
 * for everything between (not running, not signed in, the share link not yet
 * accepted). Nothing while the first look is still out.
 */
export function tailscaleStep(info: TailscaleInfo | null, chapterId: string): TailscaleStep {
  if (info === null) return null
  if (info.state === 'missing') return 'install'
  if (info.state === 'running' && (info.connected ?? []).includes(chapterId)) return 'ready'
  return 'connect'
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
