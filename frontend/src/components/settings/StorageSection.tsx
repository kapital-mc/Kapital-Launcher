import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import type { Chapter, ChapterStorage, StorageItem } from '../../types'
import { errMsg } from '../../lib/ipc'
import { formatBytes } from '../../lib/bytes'
import { FolderOpen, HardDrive, Trash2 } from '../../lib/icons'
import {
  canDelete,
  confirmText,
  GROUP_COLOUR,
  hasRows,
  storageDetail,
  storageGroups,
  storageKey,
  storageTitle,
} from '../../lib/storage'
import type { StorageGroup } from '../../lib/storage'
import { isActive, selectGame, useGameStore } from '../../stores/useGameStore'
import { useStorageStore } from '../../stores/useStorageStore'
import { useTerrainStore } from '../../stores/useTerrainStore'
import { Button } from '../ui/Button'
import { Grow } from '../ui/Grow'
import { Icon } from '../ui/Icon'
import { IconButton } from '../ui/IconButton'
import { ErrorLine, Hint, WarningLine } from '../ui/Notes'
import { SettingsCard, SettingsSection } from '../ui/SettingsLayout'
import { RunningHint } from './RunningHint'

interface Props {
  chapter: Chapter
}

/**
 * What the chapter's instance holds on disk: a bar split by colour, then the same groups as rows.
 * Distant Horizons (each server's terrain, an unfinished download) and Xaero's maps (the world
 * map, the minimap) are headings over their own rows; screenshots, logs, mods and the rest are one
 * row each. Every row opens its folder, and terrain, the download, the world map, screenshots and
 * logs can be deleted: the Delete button turns into Confirm in place, and the row says what is
 * lost. Go measures, opens and deletes, and refuses a delete while the game runs (Distant Horizons
 * holds its database open) or a terrain download runs. Only the tracker's answer (`playing`)
 * disables the deletes; `running` is the log's guess, shown as a hint. The view is read again when
 * the game ends and when the window regains focus, and the World data card is read again after
 * terrain is deleted.
 */
export function StorageSection({ chapter }: Props) {
  const view = useStorageStore((s) => s.byChapter[chapter.id])
  const unavailable = useStorageStore((s) => s.unavailable)
  const readError = useStorageStore((s) => s.readError[chapter.id])
  const clearing = useStorageStore((s) => s.clearing)
  const load = useStorageStore((s) => s.load)
  const playing = useGameStore((s) => isActive(selectGame(chapter.id)(s)?.phase))
  const wasPlaying = useRef(playing)
  const [confirming, setConfirming] = useState<string | null>(null)
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
  const open = async (item: StorageItem) => {
    setError(null)
    try {
      await useStorageStore.getState().openFolder(chapter.id, item.kind, item.name)
    } catch (e) {
      setError(errMsg(e))
    }
  }

  const groups = storageGroups(view.items)
  const row = (item: StorageItem, group: StorageGroup, title: string) => {
    const key = storageKey(item)
    return (
      <Row
        key={key}
        item={item}
        title={title}
        colour={hasRows(group) ? null : GROUP_COLOUR[group.id]}
        busy={playing || clearing !== null}
        confirming={confirming === key}
        onDelete={() => setConfirming(key)}
        onConfirm={() => void remove(item)}
        onCancel={() => setConfirming(null)}
        onOpen={() => void open(item)}
      />
    )
  }
  return (
    <SettingsSection title="Storage" icon={HardDrive} aside={formatBytes(view.totalBytes)}>
      <SplitBar groups={groups} />
      <SettingsCard>
        {groups.map((group) =>
          hasRows(group) ? (
            <div key={group.id} className="flex flex-col gap-3">
              <Heading group={group} />
              <div className="flex flex-col gap-3 pl-5">
                {group.items.map((item) => row(item, group, storageTitle(item)))}
              </div>
            </div>
          ) : (
            row(group.items[0]!, group, group.title)
          ),
        )}
      </SettingsCard>
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

/**
 * The instance split by group, each segment as wide as its share. A share is a measured value, so
 * it is set as a custom property, as ProgressBar sets its fill; a sliver stays visible.
 */
function SplitBar({ groups }: { groups: readonly StorageGroup[] }) {
  const label = groups.map((g) => `${g.title} ${formatBytes(g.bytes)}`).join(', ')
  return (
    <div role="img" aria-label={label} className="rounded-pill flex h-2 gap-0.5 overflow-hidden">
      {groups.map((group) => (
        <Segment key={group.id} group={group} />
      ))}
    </div>
  )
}

function Segment({ group }: { group: StorageGroup }) {
  const el = useRef<HTMLDivElement>(null)
  useLayoutEffect(() => {
    el.current?.style.setProperty('--storage-share', String(group.bytes))
  }, [group.bytes])
  return (
    <div
      ref={el}
      title={`${group.title}: ${formatBytes(group.bytes)}`}
      className={`${GROUP_COLOUR[group.id]} min-w-1 grow-(--storage-share) basis-0`}
    />
  )
}

function Swatch({ colour }: { colour: string }) {
  return <span aria-hidden className={`${colour} rounded-pill size-2.5 shrink-0`} />
}

/** A group's line over its rows: its colour, name and size. */
function Heading({ group }: { group: StorageGroup }) {
  return (
    <div className="flex items-center gap-2.5">
      <Swatch colour={GROUP_COLOUR[group.id]} />
      <span className="text-fg text-sm font-medium">{group.title}</span>
      <span className="text-fg-muted text-xs tabular-nums">{formatBytes(group.bytes)}</span>
    </div>
  )
}

interface RowProps {
  item: StorageItem
  title: string
  /** The group's colour for a row that is a group of its own; null under a heading. */
  colour: string | null
  busy: boolean
  confirming: boolean
  onDelete: () => void
  onConfirm: () => void
  onCancel: () => void
  onOpen: () => void
}

function Row({
  item,
  title,
  colour,
  busy,
  confirming,
  onDelete,
  onConfirm,
  onCancel,
  onOpen,
}: RowProps) {
  // The folder button is last so it lines up down the card, with or without a Delete beside it.
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
      <div className="flex min-w-0 items-center gap-2.5">
        {colour && <Swatch colour={colour} />}
        <div className="flex min-w-0 flex-col gap-1">
          <span className="text-fg text-sm font-medium break-words">{title}</span>
          <span
            className={`${confirming ? 'text-warning' : 'text-fg-muted'} text-xs tabular-nums`}
            aria-live="polite"
          >
            {confirming ? confirmText(item) : storageDetail(item)}
          </span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {canDelete(item) &&
          (confirming ? (
            <>
              <Button variant="quiet" onClick={onCancel}>
                Cancel
              </Button>
              <Button onClick={onConfirm} disabled={busy}>
                <Icon icon={Trash2} size="sm" />
                <span>Confirm</span>
              </Button>
            </>
          ) : (
            <Button onClick={onDelete} disabled={busy} title={`Delete ${title}`}>
              <Icon icon={Trash2} size="sm" />
              <span>Delete</span>
            </Button>
          ))}
        <IconButton
          icon={FolderOpen}
          size="sm"
          title={`Open the ${title} folder`}
          onClick={onOpen}
        />
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
