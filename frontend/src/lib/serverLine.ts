import type { Chapter, ServerStatus } from '../types'
import { PLACEHOLDER, isPlaceholder } from './manifest'

/**
 * The state line's two rows for a chapter with a server: what the last ping
 * said, and the detail that goes with that answer. The address is not on it:
 * which of a server's addresses is in use is a choice in settings, and the
 * line is about whether the server is up. An address nobody has settled yet is
 * never pinged for real, so the line says so rather than reporting the
 * placeholder host as offline. The second row is always there, blank while the
 * first ping is out, so the bar keeps its two lines.
 */
export function serverLine(status: ServerStatus | undefined, pending: boolean): [string, string] {
  if (pending) return ['○ No server yet', 'Address pending']
  if (!status?.checked) return ['○ Checking server', '']
  if (!status.online) return ['○ Server offline', `as of ${clock(status.checkedAt)}`]
  return ['● Server online', `${status.players}/${status.max} players · ${status.latencyMs} ms`]
}

/** A local wall-clock time for "as of", or ? when the timestamp cannot be read. */
export function clock(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime())
    ? '?'
    : d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

/**
 * What runs the chapter's server: what its last ping showed, else what the
 * manifest says, else unsettled. A chapter with no server says None.
 */
export function serverSoftware(chapter: Chapter, status: ServerStatus | undefined): string {
  if (!chapter.server) return 'None'
  if (status?.online && status.software) return status.software
  const named = chapter.server.software
  return isPlaceholder(named) ? PLACEHOLDER : named
}
