import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import logo from '../../assets/brand/kapital-launcher-logo-compact.png'
import { ms, reducedMotion, token } from '../../lib/smoothResize'
import type { Wordmark } from '../../lib/wordmark'
import { LAYOUT } from '../../styles/tokens'

/**
 * The second the launcher opens on: the wordmark's lines draw in, letter by
 * letter, the picture fills them, and the whole fades into the launcher, which
 * has been loading under it. The lines are the traced wordmark
 * (`lib/wordmark`), loaded in a chunk of their own while the cover is up, so
 * only the cover is in the first paint's bundle. It plays on every start and
 * blocks clicks while it does; with reduced motion, without the Web Animations
 * API (the tests), or if the lines fail to load, there is none.
 *
 * The timeline, from the motion tokens: each line draws over the slow 420 ms,
 * a letter a sixteenth of the fast step after the one before; the picture comes
 * in over the reveal from the slow mark while the lines go; the cover fades
 * over the grace from the slow mark plus the grace, about a second in all.
 */
export function LaunchIntro() {
  const root = useRef<HTMLDivElement>(null)
  const [mark, setMark] = useState<Wordmark | null>(null)
  const [done, setDone] = useState(false)

  useEffect(() => {
    if (reducedMotion() || typeof root.current?.animate !== 'function') {
      setDone(true)
      return
    }
    let live = true
    import('../../lib/wordmark')
      .then((m) => live && setMark(m.WORDMARK))
      .catch(() => live && setDone(true))
    return () => {
      live = false
    }
  }, [])

  useLayoutEffect(() => {
    const el = root.current
    if (!el || !mark) return
    const fast = ms('--duration-fast')
    const slow = ms('--duration-slow')
    const reveal = ms('--duration-reveal')
    const grace = ms('--duration-grace')
    const glide = token('--ease-glide')
    const standard = token('--ease-standard')
    const lines = [...el.querySelectorAll<SVGPathElement>('path')]
    const step = fast / 16
    const anims = lines.map((path, i) =>
      path.animate([{ strokeDashoffset: 1 }, { strokeDashoffset: 0 }], {
        duration: slow,
        delay: i * step,
        easing: glide,
        fill: 'both',
      }),
    )
    const drawing = el.querySelector('svg')
    const picture = el.querySelector('img')
    if (drawing) {
      anims.push(
        drawing.animate([{ opacity: 1 }, { opacity: 0 }], {
          duration: reveal,
          delay: slow,
          easing: standard,
          fill: 'forwards',
        }),
      )
    }
    if (picture) {
      anims.push(
        picture.animate([{ opacity: 0 }, { opacity: 1 }], {
          duration: reveal,
          delay: slow,
          easing: standard,
          fill: 'both',
        }),
      )
    }
    const cover = el.animate(
      [
        { opacity: 1, transform: 'scale(1)' },
        { opacity: 0, transform: 'scale(1.04)' },
      ],
      { duration: grace, delay: slow + grace, easing: standard, fill: 'forwards' },
    )
    cover.onfinish = () => setDone(true)
    anims.push(cover)
    return () => anims.forEach((a) => a.cancel())
  }, [mark])

  if (done) return null
  return (
    <div
      ref={root}
      aria-hidden
      data-testid="launch-intro"
      className="bg-canvas fixed inset-0 z-50 flex items-center justify-center"
    >
      {mark && (
        <div className="relative w-120">
          <svg
            viewBox={mark.viewBox}
            fill="none"
            stroke="currentColor"
            strokeWidth={LAYOUT.icon.stroke}
            strokeLinejoin="round"
            className="text-fg-soft absolute inset-0 size-full overflow-visible"
          >
            {[...mark.outline, ...mark.faces].map((d, i) => (
              <path
                key={i}
                d={d}
                fillRule="evenodd"
                pathLength={1}
                strokeDasharray={1}
                strokeDashoffset={1}
                vectorEffect="non-scaling-stroke"
              />
            ))}
          </svg>
          <img src={logo} alt="" className="block w-full opacity-0" />
        </div>
      )}
    </div>
  )
}
