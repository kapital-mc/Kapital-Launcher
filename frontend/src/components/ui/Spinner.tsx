import { LAYOUT } from '../../styles/tokens'

const SIZE = {
  '2xs': { cls: 'size-(--layout-icon-2xs)', px: LAYOUT.icon['2xs'] },
  sm: { cls: 'size-(--layout-icon-sm)', px: LAYOUT.icon.sm },
} as const

/**
 * The busy mark: an open ring whose arc runs round it. Nothing rotates: the arc
 * is moved along the circle by its dash offset (`spin-arc`), because WebKit
 * draws an SVG turned by CSS at a fraction of a pixel that changes as it turns,
 * so a rotated icon wobbles from side to side (WebKit bug 252385). The ring is
 * lucide's loader-circle ring, at the icons' stroke. Decorative: whatever
 * it sits beside says what is busy.
 */
export function Spinner({ size = 'sm' }: { size?: keyof typeof SIZE }) {
  const { cls, px } = SIZE[size]
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden className={`${cls} block shrink-0`}>
      <circle
        cx={12}
        cy={12}
        r={9}
        stroke="currentColor"
        strokeWidth={(LAYOUT.icon.stroke * 24) / px}
        strokeLinecap="round"
        className="spin-arc"
      />
    </svg>
  )
}
