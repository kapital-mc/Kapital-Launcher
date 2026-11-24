import { useEffect, useRef, useState } from 'react'
import type { Chapter, GameSettingsItem } from '../../types'
import { errMsg } from '../../lib/ipc'
import { RotateCcw } from '../../lib/icons'
import { canReset, settingsDetail, settingsFiles } from '../../lib/gameSettings'
import { isActive, selectGame, useGameStore } from '../../stores/useGameStore'
import { useGameSettingsStore } from '../../stores/useGameSettingsStore'
import { Button } from '../ui/Button'
import { ConfirmButton } from '../ui/ConfirmButton'
import { Fold } from '../ui/Fold'
import { Grow } from '../ui/Grow'
import { ErrorLine } from '../ui/Notes'
import { SettingsCard } from '../ui/SettingsLayout'
import { PendingRow } from './PendingSection'

interface Props {
  chapter: Chapter
}

/**
 * The game settings the Instance section resets one at a time, in a card above Reinstall (ADR-2,
 * twenty-fifth amendment): each row is a setting with a short fact beside its name, and a reset
 * button that grows into Confirm in place, as Storage's delete does. A setting with no files left
 * reads Default and cannot be reset again until the game writes it. Go resets and refuses while
 * the game runs; only the tracker's answer (`playing`) disables the buttons here. Rows keep the
 * Mods card's height: the button sits in a box as tall as a switch and overhangs it.
 *
 * The list is read when the card opens, when the game ends and when the window regains focus.
 * Until the first read answers the card holds one empty row (PendingSection); a chapter with
 * nothing to reset draws no card.
 */
export function GameSettingsCard({ chapter }: Props) {
  const view = useGameSettingsStore((s) => s.byChapter[chapter.id])
  const unavailable = useGameSettingsStore((s) => s.unavailable)
  const readError = useGameSettingsStore((s) => s.readError[chapter.id])
  const resetting = useGameSettingsStore((s) => s.resetting)
  const load = useGameSettingsStore((s) => s.load)
  const playing = useGameStore((s) => isActive(selectGame(chapter.id)(s)?.phase))
  const wasPlaying = useRef(playing)
  const [confirming, setConfirming] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    void load(chapter.id)
  }, [chapter.id, load])
  // The game writes these files as it closes: read again when the run ends and when the window
  // comes back to the front, never on a timer.
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
      <ErrorLine>{readError}</ErrorLine>
    ) : (
      <SettingsCard>
        <PendingRow />
      </SettingsCard>
    )
  }
  if (view.items.length === 0) return null

  const reset = async (item: GameSettingsItem) => {
    setConfirming(null)
    setError(null)
    try {
      await useGameSettingsStore.getState().reset(chapter.id, item.id)
    } catch (e) {
      setError(errMsg(e))
    }
  }

  return (
    <>
      <SettingsCard label="Game settings">
        {view.items.map((item) => (
          <Row
            key={item.id}
            item={item}
            busy={playing || resetting !== null}
            confirming={confirming === item.id}
            onStart={() => setConfirming(item.id)}
            onConfirm={() => void reset(item)}
            onCancel={() => setConfirming(null)}
          />
        ))}
      </SettingsCard>
      <Grow className="-mt-3" inner="pt-3">
        {error && <ErrorLine>{error}</ErrorLine>}
      </Grow>
    </>
  )
}

interface RowProps {
  item: GameSettingsItem
  busy: boolean
  confirming: boolean
  onStart: () => void
  onConfirm: () => void
  onCancel: () => void
}

function Row({ item, busy, confirming, onStart, onConfirm, onCancel }: RowProps) {
  const files = settingsFiles(item)
  return (
    <div className="flex items-center justify-between gap-6">
      <div className="flex min-w-0 items-baseline gap-2.5">
        <span className="text-fg text-sm font-medium">{item.name}</span>
        <span className="text-fg-muted truncate text-xs" title={files || undefined}>
          {settingsDetail(item)}
        </span>
      </div>
      <div className="flex h-6 shrink-0 items-center gap-2">
        <Fold open={confirming} axis="x">
          <Button variant="quiet" onClick={onCancel}>
            Cancel
          </Button>
        </Fold>
        <ConfirmButton
          icon={RotateCcw}
          action="Reset"
          title={`Reset ${item.name}`}
          confirming={confirming}
          disabled={busy || !canReset(item)}
          onStart={onStart}
          onConfirm={onConfirm}
        />
      </div>
    </div>
  )
}
