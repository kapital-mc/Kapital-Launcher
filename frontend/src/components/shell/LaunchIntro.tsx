import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { ms, reducedMotion, token } from '../../lib/smoothResize'
import type { IntroMark } from './IntroMark'

/**
 * The second the launcher opens on: the wordmark's lines draw in, letter by
 * letter, then each shape fills from its edges inward with the picture, every
 * letter at a slightly different moment and pace, and the whole fades into the
 * launcher, which has been loading under it. The drawing is `IntroMark`, the
 * traced wordmark (`lib/wordmark`), loaded in a chunk of its own while the
 * cover is up, so only the cover and this timeline are in the first paint's
 * bundle. It plays on every start and blocks clicks while it does; with reduced
 * motion, without the Web Animations API (the tests), or if the drawing fails
 * to load, there is none.
 *
 * The timeline, from the motion tokens: each line draws over the slow 420 ms,
 * a letter a sixteenth of the fast step after the one before; the rim fills
 * over the page from half the slow mark; each face from half the reveal after
 * that, over the reveal to the page, up to half the fast step late; the lines
 * fade over the reveal from the slow mark; and the cover fades over the grace
 * once the last face is full, about a second in all.
 */
export function LaunchIntro() {
  const root = useRef<HTMLDivElement>(null)
  const [Mark, setMark] = useState<typeof IntroMark | null>(null)
  const [done, setDone] = useState(false)
  // An id fit for url(#…): React's own may hold characters a URL reference does not.
  const id = `intro-${useId().replace(/[^\w-]/g, '')}`

  useEffect(() => {
    if (reducedMotion() || typeof root.current?.animate !== 'function') {
      setDone(true)
      return
    }
    let live = true
    import('./IntroMark')
      .then((m) => live && setMark(() => m.IntroMark))
      .catch(() => live && setDone(true))
    return () => {
      live = false
    }
  }, [])

  useLayoutEffect(() => {
    const el = root.current
    if (!el || !Mark) return
    const fast = ms('--duration-fast')
    const slow = ms('--duration-slow')
    const reveal = ms('--duration-reveal')
    const page = ms('--duration-page')
    const grace = ms('--duration-grace')
    const glide = token('--ease-glide')
    const standard = token('--ease-standard')
    const step = fast / 16
    const anims: Animation[] = []
    el.querySelectorAll<SVGPathElement>('[data-line]').forEach((path, i) => {
      anims.push(
        path.animate([{ strokeDashoffset: 1 }, { strokeDashoffset: 0 }], {
          duration: slow,
          delay: i * step,
          easing: glide,
          fill: 'both',
        }),
      )
    })
    let full = 0
    const fill = (path: SVGPathElement, delay: number, duration: number) => {
      full = Math.max(full, delay + duration)
      const width = 2 * Number(path.dataset.reach) + 2
      anims.push(
        path.animate([{ strokeWidth: 0 }, { strokeWidth: width }], {
          duration,
          delay,
          easing: standard,
          fill: 'both',
        }),
      )
    }
    el.querySelectorAll<SVGPathElement>('[data-fill="rim"] path').forEach((path) => {
      fill(path, slow / 2, page)
    })
    el.querySelectorAll<SVGPathElement>('[data-fill="faces"] path').forEach((path, i) => {
      const delay = slow / 2 + reveal / 2 + i * step + (Math.random() * fast) / 2
      fill(path, delay, reveal + Math.random() * (page - reveal))
    })
    const lines = el.querySelector('[data-lines]')
    if (lines) {
      anims.push(
        lines.animate([{ opacity: 1 }, { opacity: 0 }], {
          duration: reveal,
          delay: slow,
          easing: standard,
          fill: 'forwards',
        }),
      )
    }
    const cover = el.animate(
      [
        { opacity: 1, transform: 'scale(1)' },
        { opacity: 0, transform: 'scale(1.04)' },
      ],
      { duration: grace, delay: full, easing: standard, fill: 'forwards' },
    )
    cover.onfinish = () => setDone(true)
    anims.push(cover)
    return () => anims.forEach((a) => a.cancel())
  }, [Mark])

  if (done) return null
  return (
    <div
      ref={root}
      aria-hidden
      data-testid="launch-intro"
      className="bg-canvas fixed inset-0 z-50 flex items-center justify-center"
    >
      {Mark && <Mark id={id} />}
    </div>
  )
}
