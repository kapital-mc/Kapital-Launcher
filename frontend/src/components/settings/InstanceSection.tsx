import { useState } from 'react'
import type { Chapter } from '../../types'
import { errMsg } from '../../lib/ipc'
import { formatBytes } from '../../lib/bytes'
import { Folder, RotateCw, Trash2 } from '../../lib/icons'
import { useEngineStore } from '../../stores/useEngineStore'
import { isActive, selectGame, useGameStore } from '../../stores/useGameStore'
import { useModStore } from '../../stores/useModStore'
import { useStorageStore } from '../../stores/useStorageStore'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Grow } from '../ui/Grow'
import { Icon } from '../ui/Icon'
import { ErrorLine } from '../ui/Notes'
import { SettingsCard, SettingsSection } from '../ui/SettingsLayout'
import { Spinner } from '../ui/Spinner'

/** What Reinstall carries into the new instance, as services.keptEntries lists it. */
export const KEPT =
  'Worlds, screenshots, schematics, Distant Horizons terrain, maps, replays, server list.'
/** What Reinstall leaves behind for the next Play's sync to fetch fresh. */
export const RESET = 'Mods, configs, shader packs, resource packs, game options.'

type Action = 'reinstall' | 'uninstall'

interface Props {
  chapter: Chapter
}

/**
 * The chapter's instance as a whole (ADR-2, twenty-third amendment): Reinstall puts it back to a
 * fresh copy of the pack, keeping the player's own files, and Uninstall removes it from Prism,
 * worlds and all. Each asks first in a popup that names what goes. Go refuses both while the game
 * runs, a terrain download runs or Prism is open, and says why; only the tracker's answer
 * (`playing`) disables the buttons here, and the Storage section above already says to close the game. After a reinstall the memory, mods and storage shown
 * were the old instance's, so all three are read again.
 */
export function InstanceSection({ chapter }: Props) {
  const reinstall = useEngineStore((s) => s.reinstallChapter)
  const uninstall = useEngineStore((s) => s.uninstallChapter)
  const loadSettings = useEngineStore((s) => s.loadChapterSettings)
  const installing = useEngineStore((s) => s.installing !== null)
  const size = useStorageStore((s) => s.byChapter[chapter.id]?.totalBytes)
  const playing = useGameStore((s) => isActive(selectGame(chapter.id)(s)?.phase))
  const [asking, setAsking] = useState<Action | null>(null)
  const [working, setWorking] = useState<Action | null>(null)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = async (action: Action) => {
    setAsking(null)
    setError(null)
    setDone(false)
    setWorking(action)
    try {
      if (action === 'uninstall') {
        await uninstall(chapter.id)
        return
      }
      await reinstall(chapter.id, true)
      setDone(true)
      void loadSettings(chapter.id)
      void useModStore.getState().load(chapter.id)
      void useStorageStore.getState().load(chapter.id)
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setWorking(null)
    }
  }

  const busy = playing || installing || working !== null
  const id = chapter.instance.id
  return (
    <SettingsSection title="Instance" icon={Folder}>
      <SettingsCard>
        <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3">
          <div className="flex min-w-0 flex-col gap-1">
            <span className="text-fg text-sm font-medium break-words">{id}</span>
            {size !== undefined && (
              <span className="text-fg-muted text-xs tabular-nums">{formatBytes(size)}</span>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button onClick={() => setAsking('reinstall')} disabled={busy} live>
              {working === 'reinstall' ? <Spinner /> : <Icon icon={RotateCw} size="sm" />}
              <span>{done ? 'Reinstalled' : 'Reinstall'}</span>
            </Button>
            <Button onClick={() => setAsking('uninstall')} disabled={busy}>
              {working === 'uninstall' ? <Spinner /> : <Icon icon={Trash2} size="sm" />}
              <span>Uninstall</span>
            </Button>
          </div>
        </div>
      </SettingsCard>
      <Grow className="-mt-3" inner="pt-3">
        {error && <ErrorLine>{error}</ErrorLine>}
      </Grow>
      {asking === 'reinstall' && (
        <Dialog title={`Reinstall ${chapter.name}?`} facts={id} onCancel={() => setAsking(null)}>
          <dl className="m-0 flex flex-col gap-2 leading-normal">
            <div>
              <dt className="text-fg font-medium">Kept</dt>
              <dd className="text-fg-soft m-0">{KEPT}</dd>
            </div>
            <div>
              <dt className="text-fg font-medium">Reset</dt>
              <dd className="text-fg-soft m-0">{RESET}</dd>
            </div>
          </dl>
          <Choice
            label="Reinstall"
            onConfirm={() => void run('reinstall')}
            onCancel={() => setAsking(null)}
          />
        </Dialog>
      )}
      {asking === 'uninstall' && (
        <Dialog
          title={`Uninstall ${chapter.name}?`}
          facts={size !== undefined ? `${id} · ${formatBytes(size)}` : id}
          onCancel={() => setAsking(null)}
        >
          <p className="text-fg-soft m-0 leading-normal">
            This will permanently delete your files, worlds included.
          </p>
          <Choice
            label="Uninstall"
            onConfirm={() => void run('uninstall')}
            onCancel={() => setAsking(null)}
          />
        </Dialog>
      )}
    </SettingsSection>
  )
}

function Choice({
  label,
  onConfirm,
  onCancel,
}: {
  label: string
  onConfirm: () => void
  onCancel: () => void
}) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Button onClick={onConfirm}>{label}</Button>
      <Button variant="quiet" onClick={onCancel}>
        Cancel
      </Button>
    </div>
  )
}
