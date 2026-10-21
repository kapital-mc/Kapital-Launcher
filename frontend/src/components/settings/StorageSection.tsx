import { useEffect, useRef, useState } from 'react'
import type { Chapter, ChapterStorage, StorageItem } from '../../types'
import { errMsg } from '../../lib/ipc'
import { formatBytes } from '../../lib/bytes'
import { FolderOpen, HardDrive, Trash2 } from '../../lib/icons'
import { canDelete, confirmText, storageDetail, storageKey, storageTitle } from '../../lib/storage'
import { isActive, selectGame, useGameStore } from '../../stores/useGameStore'
import { useStorageStore } from '../../stores/useStorageStore'
import { useTerrainStore } from '../../stores/useTerrainStore'
import { Button } from '../ui/Button'
import { Grow } from '../ui/Grow'
import { Icon } from '../ui/Icon'
import { ErrorLine, Hint, WarningLine } from '../ui/Notes'
import { SettingsCard, SettingsSection } from '../ui/SettingsLayout'
import { RunningHint } from './RunningHint'

interface Props {
  chapter: Chapter
}

/**
 * What the chapter's instance holds on disk, by kind: each server's Distant Horizons terrain, an
 * unfinished terrain download, the world map, screenshots, logs, mods and the rest, with the
 * disk's free space. Terrain, the unfinished download, the world map and the logs can be deleted
 * after the player confirms; screenshots open in the file manager. Go measures and deletes, and
 * refuses a delete while the game runs (Distant Horizons holds its database open) or a terrain
 * download runs. Only the tracker's answer (`playing`) disables the buttons; `running` is the
 * log's guess, shown as a hint. The view is read again when the game ends and when the window
 * regains focus, and the World data card is read again after terrain is deleted.
 */
export function StorageSection({ chapter }: Props) {
  const view = useStorageStore((s) => s.byChapter[chapter.id])
  const unavailable = useStorageStore((s) => s.unavailable)
  const readError = useStorageStore((s) => s.readError[chapter.id])
  const clearing = useStorageStore((s) => s.clearing)
  const load = useStorageStore((s) => s.load)
  const playing = useGameStore((s) => isActive(selectGame(chapter.id)(s)?.phase))
  const wasPlaying = useRef(playing)
  const [confirming, setConfirming] = useState<StorageItem | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    void load(chapter.id)
  }, [chapter.id, load])
  // The log's guess and the sizes go stale while the game runs: read again when the run ends
  // and when the window comes back to the front, never on a timer.
  useEffect(() => {
    const ended = wasPlaying.current && !playing
    wasPlaying.current = playing
    if (ended) void load(chapter.id)
  }, [playing, chapter.id, load])
  useEffect(() => {
    const onFocus = () => void load(chapter.id)
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [chapter.id, load])

  if (unavailable) return null
  if (!view) {
    return readError ? (
      <SettingsSection title="Storage" icon={HardDrive}>
        <ErrorLine>{readError}</ErrorLine>
      </SettingsSection>
    ) : null
  }

  const remove = async (item: StorageItem) => {
    setConfirming(null)
    setError(null)
    try {
      await useStorageStore.getState().clear(chapter.id, item.kind, item.name)
      if (item.kind === 'terrain' || item.kind === 'terrainDownload') {
        void useTerrainStore.getState().load(chapter.id)
      }
    } catch (e) {
      setError(errMsg(e))
    }
  }
  const openScreenshots = async () => {
    setError(null)
    try {
      await useStorageStore.getState().openScreenshots(chapter.id)
    } catch (e) {
      setError(errMsg(e))
    }
  }

  const busy = playing || clearing !== null
  return (
    <SettingsSection title="Storage" icon={HardDrive} aside={formatBytes(view.totalBytes)}>
      <SettingsCard>
        {view.items.map((item) => (
          <Row
            key={storageKey(item)}
            item={item}
            busy={busy || confirming !== null}
            onDelete={() => setConfirming(item)}
            onOpen={() => void openScreenshots()}
          />
        ))}
      </SettingsCard>
      <Grow className="-mt-3" inner="pt-3">
        {confirming && (
          <Confirm
            text={confirmText(confirming)}
            onDelete={() => void remove(confirming)}
            onCancel={() => setConfirming(null)}
          />
        )}
      </Grow>
      <Grow className="-mt-3" inner="pt-3">
        {playing && <WarningLine>Close the game to delete anything.</WarningLine>}
      </Grow>
      <Grow className="-mt-3" inner="pt-3">
        {!playing && view.running && <RunningHint chapterName={chapter.name} />}
      </Grow>
      <Grow className="-mt-3" inner="pt-3">
        {error && <ErrorLine>{error}</ErrorLine>}
      </Grow>
      <Footer view={view} />
    </SettingsSection>
  )
}

interface RowProps {
  item: StorageItem
  busy: boolean
  onDelete: () => void
  onOpen: () => void
}

function Row({ item, busy, onDelete, onOpen }: RowProps) {
  const title = storageTitle(item)
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
      <div className="flex min-w-0 flex-col gap-1">
        <span className="text-fg text-sm font-medium break-words">{title}</span>
        <span className="text-fg-muted text-xs tabular-nums">{storageDetail(item)}</span>
      </div>
      {canDelete(item) && (
        <Button onClick={onDelete} disabled={busy} title={`Delete ${title.toLowerCase()}`}>
          <Icon icon={Trash2} size="sm" />
          <span>Delete</span>
        </Button>
      )}
      {item.kind === 'screenshots' && (
        <Button onClick={onOpen}>
          <Icon icon={FolderOpen} size="sm" />
          <span>Open folder</span>
        </Button>
      )}
    </div>
  )
}

function Confirm({
  text,
  onDelete,
  onCancel,
}: {
  text: string
  onDelete: () => void
  onCancel: () => void
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
      <Hint>{text}</Hint>
      <div className="flex gap-2">
        <Button onClick={onDelete}>Delete</Button>
        <Button variant="quiet" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  )
}

/** The disk's free space, and a word when the walk stopped early. */
function Footer({ view }: { view: ChapterStorage }) {
  const free = view.freeBytes >= 0 ? `${formatBytes(view.freeBytes)} free on this disk.` : ''
  const partial = view.partial ? ' Some files were not counted.' : ''
  if (!free && !partial) return null
  return <Hint>{`${free}${partial}`.trim()}</Hint>
}
