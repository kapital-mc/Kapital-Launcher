import { useEffect, useState } from 'react'
import { downloadRows } from '../../lib/downloads'
import type { DownloadRow } from '../../lib/downloads'
import { errMsg } from '../../lib/ipc'
import { useChapterStore } from '../../stores/useChapterStore'
import { useEngineStore } from '../../stores/useEngineStore'
import { useTailscaleStore } from '../../stores/useTailscaleStore'
import { useTerrainStore } from '../../stores/useTerrainStore'
import { Fold } from '../ui/Fold'
import { Grow } from '../ui/Grow'
import { TextLink } from '../ui/TextLink'
import { ProgressBar } from '../ui/ProgressBar'

/**
 * What the launcher is downloading, above the account tile: Prism, Tailscale's
 * installer and a chapter's terrain, each with its bar and step. It shows only
 * while something downloads, so a popup or a settings card can be closed and
 * the download still be followed. The sidebar is always on screen, so it also
 * holds the terrain and Tailscale listeners for the app's lifetime (Prism's is
 * App's).
 *
 * Compact, it keeps each bar and its percentage; the heading, the names, the
 * steps and Cancel fold away with the rest of the sidebar's text. The numbers
 * are drawn with even-width digits, so a count going up never moves the words
 * after it.
 */
export function Downloads({ compact = false }: { compact?: boolean }) {
  const prism = useEngineStore((s) => s.install)
  const prismUpdate = useEngineStore((s) => s.engine?.found === true)
  const tailscale = useTailscaleStore((s) => s.install)
  const terrain = useTerrainStore((s) => s.progress)
  const chapters = useChapterStore((s) => s.manifest.chapters)
  const listenTerrain = useTerrainStore((s) => s.listen)
  const listenTailscale = useTailscaleStore((s) => s.listen)
  useEffect(() => listenTerrain(), [listenTerrain])
  useEffect(() => listenTailscale(), [listenTailscale])

  const name = (id: string) => chapters.find((c) => c.id === id)?.name ?? id
  const rows = downloadRows(prism, prismUpdate, tailscale, terrain, name)
  // The section opens and closes as the first download starts and the last ends, with its gap
  // to the account tile under it: the tile's own 16px to the sidebar's sides.
  return (
    <Grow inner="pb-4">
      {rows.length > 0 && (
        <section
          aria-label="Downloads"
          className="bg-raised border-line flex flex-col rounded-lg border p-3"
        >
          <Fold open={!compact} axis="y">
            <span className="text-fg-faint text-2xs tracking-eyebrow block pb-3 font-mono whitespace-nowrap uppercase">
              Downloads
            </span>
          </Fold>
          {rows.map((r, i) => (
            <Row key={r.key} row={r} compact={compact} first={i === 0} />
          ))}
        </section>
      )}
    </Grow>
  )
}

/** One download: its name and share over the bar, and its step and Cancel under it. */
function Row({ row, compact, first }: { row: DownloadRow; compact: boolean; first: boolean }) {
  const [error, setError] = useState<string | null>(null)
  const cancel = row.cancel
  const onCancel = async () => {
    if (!cancel) return
    setError(null)
    try {
      if (cancel.kind === 'prism') await useEngineStore.getState().cancelInstall()
      else if (cancel.kind === 'tailscale') await useTailscaleStore.getState().cancelInstall()
      else await useTerrainStore.getState().cancel(cancel.chapterId)
    } catch (e) {
      setError(errMsg(e))
    }
  }
  return (
    <div className={`flex flex-col ${first ? '' : 'pt-3'}`}>
      <div className="text-2xs flex justify-between gap-2 pb-1.5 whitespace-nowrap">
        <Fold open={!compact} axis="x" className="min-w-0 grow">
          <span className="text-fg block truncate font-medium">{row.title}</span>
        </Fold>
        <span className="text-fg-muted font-mono tabular-nums">{Math.floor(row.value * 100)}%</span>
      </div>
      <ProgressBar value={row.value} label={row.title} />
      <Fold open={!compact} axis="y">
        <span className="text-2xs flex items-baseline justify-between gap-2 pt-1.5">
          <span className="text-fg-faint min-w-0 truncate tabular-nums" title={row.line}>
            {row.line}
          </span>
          {cancel && (
            <TextLink
              size="xs"
              className="text-2xs shrink-0"
              aria-label={`Cancel ${row.title}`}
              onClick={() => void onCancel()}
            >
              Cancel
            </TextLink>
          )}
        </span>
        {error && <span className="text-danger text-2xs block pt-1">{error}</span>}
      </Fold>
    </div>
  )
}
