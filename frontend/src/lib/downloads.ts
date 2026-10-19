import type { DHDownloadProgress, PrismInstallProgress } from '../types'
import type { TailscaleInstallProgress } from '../types/tailscale'
import { fillOf, isSettled, stepLabel } from './terrain'
import { tailscaleStepLine } from './tailscale'
import { installLine } from './prismInstall'

/** One running download in the sidebar's section. */
export interface DownloadRow {
  key: string
  title: string
  /** 0 to 1. */
  value: number
  line: string
}

const PRISM_WORKING = ['downloading', 'unpacking', 'verifying']
const TAILSCALE_WORKING = ['downloading', 'verifying', 'installing']

const share = (received: number, total: number) =>
  total > 0 ? Math.min(1, Math.max(0, received / total)) : 0

/**
 * Everything the launcher is downloading now, from the stores that follow
 * each download's events: Prism, Tailscale's installer and each chapter's
 * terrain. A download that has ended is not a row.
 */
export function downloadRows(
  prism: PrismInstallProgress | null,
  prismUpdate: boolean,
  tailscale: TailscaleInstallProgress | null,
  terrain: Record<string, DHDownloadProgress | undefined>,
  chapterName: (id: string) => string,
): DownloadRow[] {
  const rows: DownloadRow[] = []
  if (prism && PRISM_WORKING.includes(prism.phase)) {
    rows.push({
      key: 'prism',
      title: prismUpdate ? 'Prism Launcher update' : 'Prism Launcher',
      value: prism.phase === 'downloading' ? share(prism.received, prism.total) : 1,
      line: installLine(prism, prismUpdate ? 'Updating' : 'Getting')[1],
    })
  }
  if (tailscale && TAILSCALE_WORKING.includes(tailscale.phase)) {
    rows.push({
      key: 'tailscale',
      title: 'Tailscale',
      value: tailscale.phase === 'downloading' ? share(tailscale.received, tailscale.total) : 1,
      line: tailscaleStepLine(tailscale),
    })
  }
  for (const [id, p] of Object.entries(terrain)) {
    if (!p || isSettled(p.phase)) continue
    rows.push({
      key: `terrain-${id}`,
      title: `${chapterName(id)} terrain`,
      value: fillOf(p),
      line: stepLabel(p.phase),
    })
  }
  return rows
}
