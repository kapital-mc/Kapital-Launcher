import type { PrismRelease } from '../../types'
import { megabytes } from '../../lib/prismInstall'
import { Download } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Icon } from '../ui/Icon'
import { TextLink } from '../ui/TextLink'

interface Props {
  release: PrismRelease
  onConfirm: () => void
  onCancel: () => void
  onOpenReleasePage: () => void
  onOpenPrismSite: () => void
}

/**
 * The approval for getting Prism (ADR-11), as a popup over the window: what is
 * downloaded, how big it is, where from and under which licence, before
 * anything is fetched. Nothing happens until the player confirms. It floats
 * over the launcher rather than opening in the action bar, so the chapter
 * card keeps its layout (on a short window the inline card pushed the hero's
 * title off the top). Escape, Not now and a click on the cover all cancel.
 */
export function GetPrism({
  release,
  onConfirm,
  onCancel,
  onOpenReleasePage,
  onOpenPrismSite,
}: Props) {
  return (
    <Dialog
      title="Get Prism Launcher"
      facts={`Version ${release.version} · ${megabytes(release.size)} · GPL-3.0`}
      onCancel={onCancel}
    >
      <p className="text-fg-soft m-0 leading-normal">
        Downloaded from Prism&apos;s releases on GitHub, apart from any Prism you install yourself.
        Prism asks you to sign in with Microsoft on your first Play; Kapital Launcher never sees
        your account.
      </p>
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={onConfirm}>
          <Icon icon={Download} size="sm" />
          <span>Download</span>
        </Button>
        <Button onClick={onCancel}>Not now</Button>
      </div>
      <div className="flex flex-wrap gap-4">
        <TextLink size="xs" onClick={onOpenReleasePage}>
          What&apos;s in {release.version}
        </TextLink>
        <TextLink size="xs" onClick={onOpenPrismSite}>
          Prism&apos;s website
        </TextLink>
      </div>
    </Dialog>
  )
}
