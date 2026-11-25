import { useEffect, useState } from 'react'
import type { Chapter, Contraption } from '../../types'
import { selectInstalled, useEngineStore } from '../../stores/useEngineStore'
import { useContraptionStore } from '../../stores/useContraptionStore'
import { CONTRAPTION_KINDS, contraptionDetail, contraptionKey } from '../../lib/contraptions'
import { errMsg } from '../../lib/ipc'
import { Download, FolderOpen, Upload } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Grow } from '../ui/Grow'
import { Icon } from '../ui/Icon'
import { ErrorLine } from '../ui/Notes'
import { Page } from '../ui/Page'
import { SettingsCard } from '../ui/SettingsLayout'
import { DeleteConfirm } from '../settings/DeleteConfirm'
import { ContraptionRow } from './ContraptionRow'
import { ExportDialog } from './ExportDialog'

interface Props {
  chapter: Chapter
  onClose: () => void
}

/**
 * A chapter's saved contraptions: the Tool Gun's blueprints and Create's schematics in its game
 * folder, each kind in a card, newest first. A row opens to rename, export or delete its file;
 * Import copies files in, Export zips the files the player ticks. Go reads and writes every file (app_contraptions.go); the list is read
 * when the page opens and after each change, and emptied when it closes.
 */
export function ContraptionsPanel({ chapter, onClose }: Props) {
  const installed = useEngineStore(selectInstalled(chapter.id))
  const list = useContraptionStore((s) => s.list)
  const unavailable = useContraptionStore((s) => s.unavailable)
  const listError = useContraptionStore((s) => s.listError)
  const busy = useContraptionStore((s) => s.busy)
  const load = useContraptionStore((s) => s.load)
  const clear = useContraptionStore((s) => s.clear)
  const [openKey, setOpenKey] = useState<string | null>(null)
  const [renameError, setRenameError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [asking, setAsking] = useState<Contraption | null>(null)
  const [exporting, setExporting] = useState(false)

  useEffect(() => {
    if (installed) void load(chapter.id)
    return clear
  }, [installed, chapter.id, load, clear])

  const store = useContraptionStore.getState
  const run = async (what: () => Promise<unknown>) => {
    setError(null)
    try {
      await what()
    } catch (e) {
      setError(errMsg(e))
    }
  }
  const rename = async (c: Contraption, name: string) => {
    setRenameError(null)
    try {
      await store().rename(chapter.id, c, name)
      setOpenKey(
        contraptionKey({ kind: c.kind, file: `${name}${c.file.slice(c.file.lastIndexOf('.'))}` }),
      )
    } catch (e) {
      setRenameError(errMsg(e))
    }
  }
  const remove = (c: Contraption) => {
    setAsking(null)
    setOpenKey(null)
    void run(() => store().remove(chapter.id, c))
  }
  const toggle = (key: string) => {
    setRenameError(null)
    setOpenKey((was) => (was === key ? null : key))
  }

  let body
  if (installed === false) {
    body = <p className="text-fg-muted m-0 text-sm">{chapter.name} is not installed yet.</p>
  } else if (installed === undefined) {
    body = (
      <p className="text-fg-muted m-0 text-sm leading-normal">
        The instance could not be looked for. Set the Prism data folder in Settings.
      </p>
    )
  } else if (listError) {
    body = <ErrorLine>{listError}</ErrorLine>
  } else if (unavailable) {
    body = (
      <p className="text-fg-muted m-0 text-sm">Contraptions can only be read in the app window.</p>
    )
  } else if (!list) {
    body = null
  } else if (list.length === 0) {
    body = <p className="text-fg-muted m-0 text-sm">No saved contraptions.</p>
  } else {
    body = CONTRAPTION_KINDS.map(({ kind, title }) => {
      const items = list.filter((c) => c.kind === kind)
      if (items.length === 0) return null
      return (
        <section key={kind} aria-label={title} className="flex flex-col gap-3">
          <div className="flex items-baseline gap-2.5">
            <h2 className="font-display text-fg m-0 text-lg font-semibold">{title}</h2>
            <span className="text-fg-muted text-xs tabular-nums">{items.length}</span>
            <button
              type="button"
              onClick={() => void run(() => store().openFolder(chapter.id, kind))}
              title={`Open the ${title} folder`}
              className="text-fg-muted hover:text-fg duration-fast ease-standard ml-auto flex size-8 items-center justify-center transition-colors"
            >
              <Icon icon={FolderOpen} size="sm" />
            </button>
          </div>
          <SettingsCard label={title}>
            {items.map((c) => {
              const key = contraptionKey(c)
              return (
                <ContraptionRow
                  key={key}
                  item={c}
                  open={openKey === key}
                  busy={busy !== null}
                  renameError={openKey === key ? renameError : null}
                  onToggle={() => toggle(key)}
                  onRename={(name) => void rename(c, name)}
                  onExport={() => void run(() => store().exportFile(chapter.id, c))}
                  onDelete={() => setAsking(c)}
                />
              )
            })}
          </SettingsCard>
        </section>
      )
    })
  }

  return (
    <Page
      ready={installed !== true || listError !== null || unavailable || list !== null}
      label={`${chapter.name} contraptions`}
      eyebrow={chapter.name}
      title="Contraptions"
      onBack={onClose}
      actions={
        <div className="flex items-center gap-3">
          <Button
            onClick={() => void run(() => store().importFiles(chapter.id))}
            disabled={installed !== true || busy !== null}
          >
            <Icon icon={Download} size="sm" />
            <span>Import</span>
          </Button>
          <Button onClick={() => setExporting(true)} disabled={!list?.length || busy !== null}>
            <Icon icon={Upload} size="sm" />
            <span>Export</span>
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-8">
        <Grow className="-mb-8" inner="pb-8">
          {error && <ErrorLine>{error}</ErrorLine>}
        </Grow>
        {body}
      </div>
      {exporting && list && (
        <ExportDialog
          list={list}
          onExport={(picks) => {
            setExporting(false)
            void run(() => store().exportZip(chapter.id, picks))
          }}
          onCancel={() => setExporting(false)}
        />
      )}
      {asking && (
        <DeleteConfirm
          facts={`${asking.name} · ${contraptionDetail(asking)}`}
          onConfirm={() => remove(asking)}
          onCancel={() => setAsking(null)}
          skippable={false}
        />
      )}
    </Page>
  )
}
