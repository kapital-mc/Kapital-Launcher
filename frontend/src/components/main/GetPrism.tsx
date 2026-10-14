import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { PrismRelease } from '../../types'
import { megabytes } from '../../lib/prismInstall'
import { Download } from '../../lib/icons'
import { useEscapeClose } from '../../lib/useEscapeClose'
import { Button } from '../ui/Button'
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
 * title off the top). It is portalled to the body, so no transformed ancestor
 * can clip it; the accent and theme are on the root element and still apply.
 * Escape, Not now and a click on the cover all cancel, and focus goes back to
 * where it was.
 */
export function GetPrism({
  release,
  onConfirm,
  onCancel,
  onOpenReleasePage,
  onOpenPrismSite,
}: Props) {
  const dialog = useRef<HTMLDivElement>(null)
  useEscapeClose(onCancel)
  useEffect(() => {
    const before = document.activeElement
    dialog.current?.focus()
    return () => {
      if (before instanceof HTMLElement) before.focus()
    }
  }, [])

  return createPortal(
    <div
      className="scrim-wash fixed inset-0 z-20 flex items-center justify-center p-10"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel()
      }}
    >
      <div
        ref={dialog}
        role="dialog"
        aria-modal="true"
        aria-labelledby="get-prism-title"
        tabIndex={-1}
        className="bg-raised-2 border-line-strong flex w-full max-w-120 flex-col gap-4 rounded-lg border p-6 text-sm outline-none"
      >
        <div className="flex flex-col gap-1">
          <h2 id="get-prism-title" className="font-display m-0 text-xl font-semibold">
            Get Prism Launcher
          </h2>
          <span className="text-fg-muted font-mono text-xs">
            Version {release.version} · {megabytes(release.size)} · GPL-3.0
          </span>
        </div>
        <p className="text-fg-soft m-0 leading-normal">
          Downloaded from Prism&apos;s releases on GitHub, apart from any Prism you install
          yourself. Prism asks you to sign in with Microsoft on your first Play; Kapital Launcher
          never sees your account.
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
      </div>
    </div>,
    document.body,
  )
}
