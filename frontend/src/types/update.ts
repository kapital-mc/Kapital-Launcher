/**
 * One launcher:update step of the launcher's own update, downloading in the
 * background: "ready" is staged and waiting for Restart.
 */
export interface LauncherUpdateProgress {
  phase: 'downloading' | 'verifying' | 'ready' | 'failed' | 'cancelled' | string
  received: number
  total: number
  version: string
  error: string
}
