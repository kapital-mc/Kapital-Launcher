import { Suspense, useEffect, useLayoutEffect, useRef, useState } from 'react'
import type { AnimationEvent, ReactNode } from 'react'
import { PageSettledContext } from '../../lib/pageSettled'
import { ms } from '../../lib/smoothResize'
import { CARD } from '../ui/card'

/**
 * The edge a page comes in from: a chapter's pages from the right, the app's settings from the
 * top, under the gear, and the account from the bottom, above the sidebar's account tile
 * (issue 192).
 */
export type PageEdge = 'right' | 'top' | 'bottom'

interface Props {
  edge: PageEdge
  /** Whether the page is to be showing. False plays its exit, and `onExited` follows it. */
  open: boolean
  onExited: () => void
  /** The page, usually a lazy one: its frame shows while its code is on the way. */
  children: ReactNode
}

/** How long the slide is given when no clock can be read for it, in ms. */
const SLIDE_FALLBACK = 1000

/**
 * How long past its start an entrance may take before the page counts as settled without hearing
 * its end: the slide, and the slower motion beyond it. The end is what normally settles it; this
 * is for a webview that never reports one (a window that is not being drawn).
 */
function settleAfter(): number {
  const t = ms('--duration-page') + ms('--duration-slow')
  return Number.isFinite(t) ? t : SLIDE_FALLBACK
}

// A chapter's page is clipped by the card's own rounded frame; the settings
// come down over the whole card area, which the stage clips (.card-stage).
const EDGE = {
  right: { in: 'page-in-right', out: 'page-out-right', frame: 'overflow-hidden rounded-lg' },
  top: { in: 'page-in-top', out: 'page-out-top', frame: '' },
  bottom: { in: 'page-in-bottom', out: 'page-out-bottom', frame: '' },
} as const

/**
 * A page laid over the chapter card, inside the chapter stage (issue 155 and
 * the pages before it): it slides in from its edge, and on close slides back
 * out and stays mounted until that animation ends, then asks to be removed.
 * The chapter card does not move; it is inert beneath (ChapterStage).
 *
 * Focus follows the page. Opening moves it to the layer, so a keyboard player
 * lands in the page, and closing returns it to the control that opened it, the
 * pen, the logs button or the gear, unless the player has already moved it
 * elsewhere (a chapter picked in the sidebar). The opener is read when the
 * layer mounts, before the chapter card goes inert and drops it.
 *
 * The page and its code's fallback share the layer, so the slide starts at
 * once whether or not the page has loaded: the frame slides, and the page
 * fills it when it arrives. A closing page is inert and hidden from the
 * accessibility tree, and clicks fall through to the chapter. The animation is
 * one keyframe set on the layer (style.css), and with reduced motion a fade
 * that still ends, so the exit always completes.
 *
 * The layer also tells its page when the entrance is over (`usePageSettled`, lib/pageSettled).
 * A page with slow reads starts them at once and holds back what they add to its layout until
 * then, so it does not grow while it moves. The end of the entrance settles it; so does a
 * timer past the entrance's length, and a window that is hidden when the page opens is settled
 * at once, as neither would report an end there. A page that closes is unsettled again. The
 * layer carries it as `data-page-settled`, which hides the compact panels beneath (style.css).
 */
export function PageLayer({ edge, open, onExited, children }: Props) {
  const layer = useRef<HTMLDivElement>(null)
  const [opener] = useState(() => document.activeElement)
  const [settled, setSettled] = useState(() => document.visibilityState === 'hidden')
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (!open) setSettled(false)
  }
  const motion = EDGE[edge]

  useEffect(() => {
    if (!open || settled) return
    const timer = setTimeout(() => setSettled(true), settleAfter())
    return () => clearTimeout(timer)
  }, [open, settled])

  useLayoutEffect(() => {
    if (open) {
      layer.current?.focus({ preventScroll: true })
      return
    }
    const active = document.activeElement
    if (
      (active === document.body || layer.current?.contains(active)) &&
      opener instanceof HTMLElement &&
      opener.isConnected
    ) {
      opener.focus({ preventScroll: true })
    }
    // An exit in a hidden window is not seen and its animation would wait
    // there until the window came back: the page is simply removed.
    if (document.visibilityState === 'hidden') onExited()
  }, [open, opener, onExited])

  const onAnimationEnd = (e: AnimationEvent<HTMLDivElement>) => {
    // The page's own animations (its reveal) end inside this element too.
    if (e.target !== e.currentTarget) return
    if (open) setSettled(true)
    else onExited()
  }

  return (
    <div
      inert={!open}
      aria-hidden={!open || undefined}
      data-page-settled={settled || undefined}
      className={`absolute inset-0 ${motion.frame} ${open ? '' : 'pointer-events-none'}`}
    >
      <div
        ref={layer}
        tabIndex={-1}
        onAnimationEnd={onAnimationEnd}
        className={`flex h-full flex-col outline-none ${open ? motion.in : motion.out}`}
      >
        <PageSettledContext value={settled}>
          <Suspense fallback={<div aria-hidden className={CARD} />}>{children}</Suspense>
        </PageSettledContext>
      </div>
    </div>
  )
}
