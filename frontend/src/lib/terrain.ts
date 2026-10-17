import type { DHDownloadProgress, DHSnapshotInfo } from '../types'

/** The phases after which only a new download moves the progress again. */
const SETTLED: readonly string[] = ['done', 'cancelled', 'failed']

export const isSettled = (phase: string): boolean => SETTLED.includes(phase)

/** Whether a terrain download is under way, by its progress or, before any, by Go's answer. */
export function isDownloading(
  info: DHSnapshotInfo | undefined,
  progress: DHDownloadProgress | undefined,
): boolean {
  if (progress) return !isSettled(progress.phase)
  return info?.downloading ?? false
}

/** How much of the download is done, 0 to 1; the last two steps show it full. */
export function fillOf(p: DHDownloadProgress): number {
  if (p.phase === 'verifying' || p.phase === 'placing') return 1
  if (p.total <= 0) return 0
  return Math.min(1, Math.max(0, p.received / p.total))
}

const STEPS: Record<string, string> = {
  checking: 'Getting ready',
  downloading: 'Downloading',
  verifying: 'Checking the files',
  placing: 'Putting the terrain in place',
}

/** The step a running download is on, as the card says it. */
export const stepLabel = (phase: string): string => STEPS[phase] ?? 'Downloading'

/** The day a snapshot was published, in the player's locale, or as Go sent it. */
export function publishedOn(iso: string, locale?: string): string {
  const at = new Date(iso)
  if (Number.isNaN(at.getTime())) return iso
  return at.toLocaleDateString(locale, { dateStyle: 'medium' })
}
