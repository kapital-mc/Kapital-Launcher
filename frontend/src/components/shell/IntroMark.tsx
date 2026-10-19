import logo from '../../assets/brand/kapital-launcher-logo-compact.png'
import { WORDMARK, type Wordmark, type WordmarkShape } from '../../lib/wordmark'
import { LAYOUT } from '../../styles/tokens'

/**
 * The launch intro's wordmark (`LaunchIntro` animates it), in a chunk of its
 * own with the traced shapes: the lines, each set to draw from nothing, and
 * the picture under two masks that start empty.
 *
 * The fill is two masks over the picture, outside first: the wordmark's dark
 * rim, clipped to itself, fills from its outer edge inward, then each letter's
 * face, clipped to the faces, from its own edge. A mask is the shape's edge
 * stroked, growing from nothing to twice the shape's reach (its inside's
 * furthest point from the edge), so the stroke's inner half closes in on the
 * middle; the clip keeps its outer half off the neighbouring part.
 */
export function IntroMark({ id }: { id: string }) {
  const mark = WORDMARK
  const shapes = [...mark.outline, ...mark.faces]
  return (
    <svg viewBox={mark.viewBox} className="block h-auto w-120 overflow-visible">
      <defs>
        {/* The rim is the outline less the faces, which sit inside it: even-odd over all of them. */}
        <clipPath id={`${id}-rim`}>
          <path d={shapes.map((s) => s.d).join(' ')} clipRule="evenodd" />
        </clipPath>
        <clipPath id={`${id}-faces`}>
          {mark.faces.map(({ d }, i) => (
            <path key={i} d={d} clipRule="evenodd" />
          ))}
        </clipPath>
        <FillMask id={`${id}-rim-fill`} kind="rim" mark={mark} shapes={mark.outline} />
        <FillMask id={`${id}-faces-fill`} kind="faces" mark={mark} shapes={mark.faces} />
      </defs>
      <g clipPath={`url(#${id}-rim)`}>
        <image href={logo} width={mark.width} height={mark.height} mask={`url(#${id}-rim-fill)`} />
      </g>
      <g clipPath={`url(#${id}-faces)`}>
        <image
          href={logo}
          width={mark.width}
          height={mark.height}
          mask={`url(#${id}-faces-fill)`}
        />
      </g>
      <g
        data-lines
        fill="none"
        stroke="currentColor"
        strokeWidth={LAYOUT.icon.stroke}
        strokeLinejoin="round"
        className="text-fg-soft"
      >
        {shapes.map(({ d }, i) => (
          <path
            key={i}
            d={d}
            data-line
            pathLength={1}
            strokeDasharray={1}
            strokeDashoffset={1}
            vectorEffect="non-scaling-stroke"
          />
        ))}
      </g>
    </svg>
  )
}

/** A fill's mask: each shape's edge, stroked at nothing until the timeline grows it. */
function FillMask({
  id,
  kind,
  mark,
  shapes,
}: {
  id: string
  kind: 'rim' | 'faces'
  mark: Wordmark
  shapes: WordmarkShape[]
}) {
  return (
    <mask
      id={id}
      data-fill={kind}
      maskUnits="userSpaceOnUse"
      x={0}
      y={0}
      width={mark.width}
      height={mark.height}
      className="mask-type-alpha"
    >
      {shapes.map(({ d, reach }, i) => (
        <path
          key={i}
          d={d}
          data-reach={reach}
          fill="none"
          stroke="currentColor"
          strokeWidth={0}
          strokeLinejoin="round"
        />
      ))}
    </mask>
  )
}
