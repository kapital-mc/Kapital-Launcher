/** Where an install the popup is about stands. */
export type InstallStage =
  | { kind: 'offer' }
  | { kind: 'working'; value: number; line: string }
  | { kind: 'done' }
  | { kind: 'failed'; error: string }

interface Progress {
  phase: string
  received: number
  total: number
  error: string
}

/** A download's share done, from its bytes; the steps after it count as full. */
export function downloadValue(p: Progress): number {
  if (p.phase !== 'downloading') return 1
  return p.total > 0 ? p.received / p.total : 0
}

/**
 * The popup's stage for an install's latest step, null before Download.
 * `line` words a step that is still going.
 */
export function installStage<P extends Progress>(
  install: P | null,
  line: (p: P) => string,
): InstallStage {
  if (install === null) return { kind: 'offer' }
  if (install.phase === 'done') return { kind: 'done' }
  if (install.phase === 'failed') return { kind: 'failed', error: install.error }
  return { kind: 'working', value: downloadValue(install), line: line(install) }
}
