import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import type { Chapter, DHDownloadProgress, DHSnapshotInfo } from '../../types'
import { errMsg } from '../../lib/ipc'
import { Download, X } from '../../lib/icons'
import { formatBytes } from '../../lib/bytes'
import { fillOf, isDownloading, publishedOn, stepLabel } from '../../lib/terrain'
import { isActive, selectGame, useGameStore } from '../../stores/useGameStore'
import { useTerrainStore } from '../../stores/useTerrainStore'
import { Button } from '../ui/Button'
import { Icon } from '../ui/Icon'
import { ErrorLine, Hint } from '../ui/Notes'
import { SubHeading } from '../ui/Section'
import { SettingsCard } from '../ui/SettingsLayout'

interface Props {
  chapter: Chapter
}

/**
 * The server's Distant Horizons terrain (issue 136), under the chapter's Server settings, for a
 * chapter whose pack has the mod: a one-time download of the far terrain the server has already
 * generated, which otherwise fills in slowly while the player plays. The card reads what is
 * published when it opens (its date, size and dimensions) and downloads only on Download. Terrain
 * the instance already holds for the server is replaced only after the player confirms. While it
 * runs, the bar follows dh:download and Cancel keeps what was verified for the next try. Go
 * refuses while the game runs, and says why.
 */
export function DistantHorizonsCard({ chapter }: Props) {
  const info = useTerrainStore((s) => s.byChapter[chapter.id])
  const progress = useTerrainStore((s) => s.progress[chapter.id])
  const readError = useTerrainStore((s) => s.readError[chapter.id])
  const unavailable = useTerrainStore((s) => s.unavailable)
  const load = useTerrainStore((s) => s.load)
  const listen = useTerrainStore((s) => s.listen)
  const playing = useGameStore((s) => isActive(selectGame(chapter.id)(s)?.phase))
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const downloading = isDownloading(info, progress)

  useEffect(() => listen(), [listen])
  useEffect(() => {
    void load(chapter.id)
  }, [chapter.id, load, playing])
  // The running guess goes stale when the game closes: read again when the window comes back.
  useEffect(() => {
    const onFocus = () => {
      if (!isDownloading(info, progress)) void load(chapter.id)
    }
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [chapter.id, load, info, progress])

  const start = async (replace: boolean) => {
    setConfirming(false)
    setError(null)
    try {
      await useTerrainStore.getState().download(chapter.id, replace)
    } catch (e) {
      setError(errMsg(e))
    }
  }
  const cancel = async () => {
    try {
      await useTerrainStore.getState().cancel(chapter.id)
    } catch (e) {
      setError(errMsg(e))
    }
  }
  const onDownload = () => {
    if (info && info.existingBytes > 0) setConfirming(true)
    else void start(false)
  }

  const ready = !unavailable && !!info?.available && !playing
  return (
    <div className="flex flex-col gap-3 pt-3">
      <SubHeading>World data</SubHeading>
      <SettingsCard>
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
            <div className="flex min-w-0 flex-col gap-1">
              <span className="text-fg flex items-center gap-2 text-sm font-medium">
                Distant Horizons terrain
                {unavailable && (
                  <span className="border-line text-fg-muted rounded-pill text-2xs border px-2 font-normal">
                    Not available here
                  </span>
                )}
              </span>
              {info && <Summary info={info} />}
            </div>
            {downloading ? (
              <Button onClick={() => void cancel()}>
                <Icon icon={X} size="sm" />
                <span>Cancel</span>
              </Button>
            ) : (
              <Button onClick={onDownload} disabled={!ready || confirming}>
                <Icon icon={Download} size="sm" />
                <span>{info && info.partialBytes > 0 ? 'Continue' : 'Download'}</span>
              </Button>
            )}
          </div>
          {confirming && info && (
            <Confirm
              info={info}
              onReplace={() => void start(true)}
              onKeep={() => setConfirming(false)}
            />
          )}
          {downloading && progress && <Progress progress={progress} />}
          <Outcome
            info={info}
            progress={downloading ? undefined : progress}
            error={error ?? readError}
          />
        </div>
      </SettingsCard>
    </div>
  )
}

/** What is published, and what of it the instance already has. */
function Summary({ info }: { info: DHSnapshotInfo }) {
  if (!info.createdAt) return null
  const dims = info.dimensions === 1 ? '1 dimension' : `${info.dimensions} dimensions`
  return (
    <span className="text-fg-muted text-xs">
      {`Published ${publishedOn(info.createdAt)} · ${formatBytes(info.sizeBytes)} · ${dims}`}
      {info.partialBytes > 0 && ` · ${formatBytes(info.partialBytes)} already downloaded`}
    </span>
  )
}

function Confirm({
  info,
  onReplace,
  onKeep,
}: {
  info: DHSnapshotInfo
  onReplace: () => void
  onKeep: () => void
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
      <Hint>
        {`This instance already has ${formatBytes(info.existingBytes)} of terrain for this server. The download replaces it.`}
      </Hint>
      <div className="flex gap-2">
        <Button onClick={onReplace}>Replace</Button>
        <Button onClick={onKeep}>Keep mine</Button>
      </div>
    </div>
  )
}

/** The bar, filled through a custom property: an inline `style` prop is refused. */
function Progress({ progress }: { progress: DHDownloadProgress }) {
  const fill = useRef<HTMLDivElement>(null)
  const value = fillOf(progress)
  useLayoutEffect(() => {
    fill.current?.style.setProperty('--dh-fill', String(value))
  }, [value])
  const percent = Math.floor(value * 100)
  const label =
    progress.phase === 'downloading'
      ? `${stepLabel(progress.phase)} · ${formatBytes(progress.received)} of ${formatBytes(progress.total)}`
      : stepLabel(progress.phase)
  return (
    <div className="flex flex-col gap-1.5">
      <div
        role="progressbar"
        aria-label="Terrain download"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
        className="bg-line-strong rounded-pill h-1 overflow-hidden"
      >
        <div
          ref={fill}
          className="bg-accent duration-fast ease-standard size-full origin-left scale-x-(--dh-fill) transition-transform motion-reduce:transition-none"
        />
      </div>
      <span className="text-fg-muted text-xs">{`${label} · ${percent}%`}</span>
    </div>
  )
}

/** Why Download is held, or how the last download ended. */
function Outcome({
  info,
  progress,
  error,
}: {
  info: DHSnapshotInfo | undefined
  progress: DHDownloadProgress | undefined
  error: string | null | undefined
}) {
  const failed = progress?.phase === 'failed' ? progress.error : undefined
  if (error || failed) return <ErrorLine>{error ?? failed}</ErrorLine>
  if (progress?.phase === 'done') {
    return <Hint>The terrain is in place. It shows the next time you join the server.</Hint>
  }
  if (progress?.phase === 'cancelled') {
    return (
      <Hint>
        Cancelled. What was downloaded is kept, so the next download continues from there.
      </Hint>
    )
  }
  if (info && !info.available && !info.downloading && info.reason) return <Hint>{info.reason}</Hint>
  return null
}
