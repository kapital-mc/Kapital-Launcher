import { useEffect } from 'react'
import { downloadRows } from '../../lib/downloads'
import { useChapterStore } from '../../stores/useChapterStore'
import { useEngineStore } from '../../stores/useEngineStore'
import { useTailscaleStore } from '../../stores/useTailscaleStore'
import { useTerrainStore } from '../../stores/useTerrainStore'
import { Fold } from '../ui/Fold'
import { ProgressBar } from '../ui/ProgressBar'

/**
 * What the launcher is downloading, above the account tile: Prism, Tailscale's
 * installer and a chapter's terrain, each with its bar and step. It shows only
 * while something downloads, so a popup or a settings card can be closed and
 * the download still be followed. The sidebar is always on screen, so it also
 * holds the terrain and Tailscale listeners for the app's lifetime (Prism's is
 * App's).
 *
 * Compact, it keeps each bar and its percentage; the heading, the names and
 * the steps fold away with the rest of the sidebar's text.
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
  if (rows.length === 0) return null
  return (
    <section
      aria-label="Downloads"
      className="bg-raised border-line flex flex-col gap-3 rounded-lg border p-3"
    >
      <Fold open={!compact} axis="y" className="-mb-3">
        <span className="text-fg-faint text-2xs tracking-eyebrow block pb-3 font-mono whitespace-nowrap uppercase">
          Downloads
        </span>
      </Fold>
      {rows.map((r) => (
        <div key={r.key} className="flex flex-col">
          <div className="text-2xs flex justify-between gap-2 pb-1.5 whitespace-nowrap">
            <Fold open={!compact} axis="x" className="min-w-0 grow">
              <span className="text-fg block truncate font-medium">{r.title}</span>
            </Fold>
            <span className="text-fg-muted font-mono">{Math.floor(r.value * 100)}%</span>
          </div>
          <ProgressBar value={r.value} label={r.title} />
          <Fold open={!compact} axis="y">
            <span className="text-fg-faint text-2xs block truncate pt-1.5" title={r.line}>
              {r.line}
            </span>
          </Fold>
        </div>
      ))}
    </section>
  )
}
