import type { DHDownloadProgress, PrismInstallProgress } from '../types'
import type { LauncherUpdateProgress } from '../types/update'
import type { TailscaleInstallProgress } from '../types/tailscale'
import { formatBytes } from './bytes'
import { fillOf, isSettled, stepLabel } from './terrain'
import { tailscaleStepLine } from './tailscale'
import { installLine } from './prismInstall'

/** What a row's Cancel ends: the launcher's update, Prism's or Tailscale's install, or a chapter's terrain. */
export type DownloadCancel =
  | { kind: 'launcher' }
  | { kind: 'prism' }
  | { kind: 'tailscale' }
  | { kind: 'terrain'; chapterId: string }

/** One running download in the sidebar's section. */
export interface DownloadRow {
  key: string
  title: string
  /** 0 to 1. */
  value: number
  line: string
  /**
   * What Cancel ends, while it can: an install only while it downloads (the steps after it run
   * to their end), the terrain at any step.
   */
  cancel: DownloadCancel | null
}

const LAUNCHER_WORKING = ['downloading', 'verifying']
const PRISM_WORKING = ['downloading', 'unpacking', 'verifying']
const TAILSCALE_WORKING = ['downloading', 'verifying', 'installing']

const share = (received: number, total: number) =>
  total > 0 ? Math.min(1, Math.max(0, received / total)) : 0

/**
 * Everything the launcher is downloading now, from the stores that follow
 * each download's events: its own update, Prism, Tailscale's installer and
 * each chapter's terrain. A download that has ended is not a row.
 */
export function downloadRows(
  launcher: LauncherUpdateProgress | null,
  prism: PrismInstallProgress | null,
  prismUpdate: boolean,
  tailscale: TailscaleInstallProgress | null,
  terrain: Record<string, DHDownloadProgress | undefined>,
  chapterName: (id: string) => string,
): DownloadRow[] {
  const rows: DownloadRow[] = []
  if (launcher && LAUNCHER_WORKING.includes(launcher.phase)) {
    const downloading = launcher.phase === 'downloading'
    rows.push({
      key: 'launcher',
      title: 'Kapital Launcher update',
      value: downloading ? share(launcher.received, launcher.total) : 1,
      line:
        downloading && launcher.total > 0
          ? `Downloading ${formatBytes(launcher.received)} of ${formatBytes(launcher.total)}`
          : downloading
            ? 'Downloading'
            : 'Checking the signature',
      cancel: downloading ? { kind: 'launcher' } : null,
    })
  }
  if (prism && PRISM_WORKING.includes(prism.phase)) {
    rows.push({
      key: 'prism',
      title: prismUpdate ? 'Prism Launcher update' : 'Prism Launcher',
      value: prism.phase === 'downloading' ? share(prism.received, prism.total) : 1,
      line: installLine(prism, prismUpdate ? 'Updating' : 'Getting')[1],
      cancel: prism.phase === 'downloading' ? { kind: 'prism' } : null,
    })
  }
  if (tailscale && TAILSCALE_WORKING.includes(tailscale.phase)) {
    rows.push({
      key: 'tailscale',
      title: 'Tailscale',
      value: tailscale.phase === 'downloading' ? share(tailscale.received, tailscale.total) : 1,
      line: tailscaleStepLine(tailscale),
      cancel: tailscale.phase === 'downloading' ? { kind: 'tailscale' } : null,
    })
  }
  for (const [id, p] of Object.entries(terrain)) {
    if (!p || isSettled(p.phase)) continue
    rows.push({
      key: `terrain-${id}`,
      title: `${chapterName(id)} terrain`,
      value: fillOf(p),
      line:
        p.phase === 'downloading' && p.total > 0
          ? `${stepLabel(p.phase)} ${formatBytes(p.received)} of ${formatBytes(p.total)}`
          : stepLabel(p.phase),
      cancel: { kind: 'terrain', chapterId: id },
    })
  }
  return rows
}
