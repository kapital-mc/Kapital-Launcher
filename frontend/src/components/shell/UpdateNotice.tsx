import { useUpdateStore } from '../../stores/useUpdateStore'
import { ExternalLink } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Icon } from '../ui/Icon'

/**
 * The popup for a newer Kapital Launcher (ADR-9 amendment): which version is
 * out and which this is, and a button to the download site, where the player
 * gets the installer as they did the first time. The launcher downloads and
 * installs nothing itself. Not now puts it away until the next start.
 */
export function UpdateNotice() {
  const update = useUpdateStore((s) => s.update)
  const error = useUpdateStore((s) => s.error)
  const dismiss = useUpdateStore((s) => s.dismiss)
  const openPage = useUpdateStore((s) => s.openPage)
  if (!update) return null

  return (
    <Dialog
      title="A new version is out"
      facts={`${update.latest} · you have ${update.current}`}
      onCancel={dismiss}
    >
      <p className="text-fg-soft m-0 leading-normal">
        Download it from the Kapital Launcher site and install it over this one. Your chapters and
        settings stay as they are.
      </p>
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={() => void openPage()}>
          <Icon icon={ExternalLink} size="sm" />
          <span>Open the download page</span>
        </Button>
        <Button onClick={dismiss}>Not now</Button>
      </div>
      {error && <p className="text-danger m-0 text-xs">{error}</p>}
    </Dialog>
  )
}
