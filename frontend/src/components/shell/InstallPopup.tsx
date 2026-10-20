import type { ReactNode } from 'react'
import type { InstallStage } from '../../lib/installStage'
import { Download } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Icon } from '../ui/Icon'
import { ErrorLine } from '../ui/Notes'
import { ProgressBar } from '../ui/ProgressBar'

interface Props {
  title: string
  /** Version, size and licence, under the title. */
  facts: string
  stage: InstallStage
  /** What is downloaded and from where, shown before the player confirms. */
  intro: ReactNode
  /** What comes next once the install is done: a sign-in, a system prompt. */
  done: ReactNode
  /** Links under the buttons: release notes, the project's site. */
  links?: ReactNode
  onConfirm: () => void
  onClose: () => void
}

/**
 * The one popup for getting something the launcher needs and does not ship:
 * Prism Launcher (ADR-11) and Tailscale. It floats over the window, so
 * nothing in the chapter card moves, and it stays through the install: the
 * offer, then the bar and the step, then what to do next or what went wrong.
 * Closing it while the download runs only hides it; the download carries on
 * and the sidebar's downloads section shows it.
 */
export function InstallPopup({
  title,
  facts,
  stage,
  intro,
  done,
  links,
  onConfirm,
  onClose,
}: Props) {
  return (
    <Dialog title={title} facts={facts} onCancel={onClose}>
      {stage.kind === 'offer' && (
        <>
          <div className="text-fg-soft m-0 flex flex-col gap-2 leading-normal">{intro}</div>
          <div className="flex flex-wrap items-center gap-3">
            <Button onClick={onConfirm}>
              <Icon icon={Download} size="sm" />
              <span>Download</span>
            </Button>
            <Button variant="quiet" onClick={onClose}>
              Not now
            </Button>
          </div>
        </>
      )}
      {stage.kind === 'working' && (
        <>
          <div className="flex flex-col gap-1.5">
            <ProgressBar value={stage.value} label={title} />
            <span className="text-fg-muted text-xs">{stage.line}</span>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Button variant="quiet" onClick={onClose}>
              Hide
            </Button>
          </div>
        </>
      )}
      {stage.kind === 'done' && (
        <>
          <div className="text-fg-soft m-0 flex flex-col gap-2 leading-normal">{done}</div>
          <div className="flex flex-wrap items-center gap-3">
            <Button variant="quiet" onClick={onClose}>
              Close
            </Button>
          </div>
        </>
      )}
      {stage.kind === 'failed' && (
        <>
          <ErrorLine>{stage.error || 'Something went wrong'}</ErrorLine>
          <div className="flex flex-wrap items-center gap-3">
            <Button onClick={onConfirm}>Try again</Button>
            <Button variant="quiet" onClick={onClose}>
              Close
            </Button>
          </div>
        </>
      )}
      {links && <div className="flex flex-wrap gap-4">{links}</div>}
    </Dialog>
  )
}
