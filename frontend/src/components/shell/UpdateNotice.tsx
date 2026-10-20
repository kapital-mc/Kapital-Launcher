import { useEffect, useId, useRef, useState } from 'react'
import { selectUpdateShown, useUpdateStore } from '../../stores/useUpdateStore'
import { Download, ExternalLink } from '../../lib/icons'
import { ms } from '../../lib/smoothResize'
import { useEscapeClose } from '../../lib/useEscapeClose'
import { Button } from '../ui/Button'
import { Icon } from '../ui/Icon'
import { IconButton } from '../ui/IconButton'

/**
 * The header's update button and what it opens (ADR-9 amendment): while a
 * newer Kapital Launcher is out, a button in the accent, and under it a panel
 * that names the version and opens the download site, where the player gets
 * the installer as they did the first time. The launcher downloads and
 * installs nothing itself. The panel opens by itself once a start; Not now,
 * Escape or a click anywhere else puts it away until the button opens it again.
 *
 * The header mounts it in a chunk of its own: it asks Go on mount (Go reads
 * the site once a start) and renders nothing unless a newer build is out.
 */
export function UpdateNotice({ size }: { size: 'xs' | 'sm' }) {
  const load = useUpdateStore((s) => s.load)
  const out = useUpdateStore((s) => s.update?.available === true)
  const shown = useUpdateStore(selectUpdateShown)
  const reopen = useUpdateStore((s) => s.reopen)
  useEffect(() => {
    void load()
  }, [load])
  if (!out) return null

  return (
    <span className="relative flex">
      <IconButton
        icon={Download}
        title="Update Launcher"
        size={size}
        tooltip={!shown}
        tone="accent"
        onClick={reopen}
      />
      {shown && <UpdatePanel />}
    </span>
  )
}

/** The panel under the button: it drops out of it, and folds back up into it as it goes. */
function UpdatePanel() {
  const update = useUpdateStore((s) => s.update)
  const error = useUpdateStore((s) => s.error)
  const dismiss = useUpdateStore((s) => s.dismiss)
  const openPage = useUpdateStore((s) => s.openPage)
  const panel = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const [leaving, setLeaving] = useState(false)

  const close = () => {
    // Without the Web Animations API (the tests) nothing would end the fold.
    if (typeof panel.current?.getAnimations !== 'function') return dismiss()
    setLeaving(true)
    // WebKit has been seen to drop an animationend; the panel goes regardless.
    window.setTimeout(dismiss, ms('--duration-reveal') + ms('--duration-fast'))
  }
  useEscapeClose(close)
  // A press anywhere else puts it away; the button's own click then finds it leaving.
  useEffect(() => {
    const onDown = (e: PointerEvent) => {
      if (!panel.current?.contains(e.target as Node)) close()
    }
    window.addEventListener('pointerdown', onDown)
    return () => window.removeEventListener('pointerdown', onDown)
  })

  if (!update) return null
  return (
    <div
      ref={panel}
      role="dialog"
      aria-labelledby={titleId}
      onAnimationEnd={(e) => {
        if (e.target === e.currentTarget && leaving) dismiss()
      }}
      className={`bg-raised-2 border-line-strong absolute top-full left-0 z-30 mt-2 flex w-88 flex-col gap-3 rounded-lg border p-5 text-sm ${
        leaving ? 'drop-out' : 'drop-in'
      }`}
    >
      <div className="flex flex-col gap-1">
        <h2 id={titleId} className="font-display m-0 text-lg font-semibold">
          Update available
        </h2>
        <span className="text-fg-muted font-mono text-xs">
          {update.latest} · you have {update.current}
        </span>
      </div>
      <p className="text-fg-soft m-0 leading-normal">
        Install it over this one from the download site. Your chapters and settings stay.
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <Button onClick={() => void openPage()}>
          <Icon icon={ExternalLink} size="sm" />
          <span>Download</span>
        </Button>
        <Button variant="quiet" onClick={close}>
          Not now
        </Button>
      </div>
      {error && <p className="text-danger m-0 text-xs">{error}</p>}
    </div>
  )
}
