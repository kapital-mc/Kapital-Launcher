/**
 * The motion between the full launcher and the compact one
 * (docs/adr/0015-compact-launcher.md). Both layouts are laid out at all times,
 * the one not shown hidden, so each can be measured as it lies. A layer of
 * actors over both, copies of the pieces that move, carries one into the other:
 *
 * - The sidebar leaves to the left.
 * - The chapter's picture keeps its place and is cropped from the hero's frame
 *   to its panel's (a clip-path), while the picture, at its own aspect, scales
 *   from the hero's cover to the panel's. It grows down over the pack, the
 *   changelog and the wiki.
 * - The other chapters slide in from the side they sit on.
 * - The title and its label travel to the panel's foot with the picture, then
 *   rise into place while Play and the server line fade and slide in beneath.
 *   The blurb, the pills and the card's own Play row fade first.
 *
 * Leaving is the same keyframes played from the other end, with each curve
 * mirrored, so it is the arrival run backwards, except that nothing waits for
 * the compact layout's last touches to undo: Play and the server line fade out
 * as the travel starts, the title travels straight from its place, and the
 * travel keeps its own curve so it starts at once. The way out is the travel's
 * length. Everything is described once, as keyframes from
 * the full layout to the compact one (`planMorph`), and the travel uses one
 * even curve, so the picture, the panels and the title move as one. Nothing starts before every picture in the layer is decoded, so the
 * first frames are not dropped. A window resized mid-way is measured again and
 * every piece's end moves, keeping its progress.
 */

export interface Box {
  x: number
  y: number
  w: number
  h: number
}

/** The rectangle `el` takes, relative to `area`. */
export function boxOf(el: Element, area: Element): Box {
  const a = area.getBoundingClientRect()
  const r = el.getBoundingClientRect()
  return { x: r.left - a.left, y: r.top - a.top, w: r.width, h: r.height }
}

/**
 * Where a picture of natural size `nat` is drawn in `box` with `object-fit:
 * cover` at `px`/`py` (the hero's and the panels' `object-[center_40%]`).
 */
export function cover(box: Box, nat: readonly [number, number], px = 0.5, py = 0.4): Box {
  const s = Math.max(box.w / nat[0], box.h / nat[1])
  const w = nat[0] * s
  const h = nat[1] * s
  return { x: box.x + (box.w - w) * px, y: box.y + (box.h - h) * py, w, h }
}

/** A cubic-bezier played backwards: what eased out now eases in. Anything else is kept. */
export function mirror(easing: string): string {
  const v = /cubic-bezier\(([^)]*)\)/.exec(easing)
  if (!v?.[1]) return easing
  const [x1, y1, x2, y2] = v[1].split(',').map(Number) as [number, number, number, number]
  const f = (n: number) => Number((1 - n).toFixed(4))
  return `cubic-bezier(${f(x2)}, ${f(y2)}, ${f(x1)}, ${f(y1)})`
}

/**
 * The same keyframes played from the other end: their order, their offsets and
 * the curve of each segment, which moves to the keyframe that now starts it.
 */
export function reverseKeyframes(kf: readonly Keyframe[]): Keyframe[] {
  const n = kf.length
  return kf.map((_, i) => {
    const { easing: _own, offset, ...rest } = kf[n - 1 - i]!
    const prev = kf[n - 2 - i]
    const out: Keyframe = { ...rest }
    if (offset != null) out.offset = 1 - offset
    if (prev?.easing) out.easing = mirror(prev.easing)
    return out
  })
}

/** Every rectangle the morph needs, read from both layouts. */
export interface MorphMeasure {
  /** The area both layouts fill, which the actors cover. */
  W: number
  H: number
  hero: Box
  panel: Box
  /** The picture's own rectangle in the hero and in the panel. */
  artFull: Box
  artCompact: Box
  /** Every panel, in the nav's order, and which one is the chapter's. */
  panels: Box[]
  active: number
  label: [Box, Box]
  title: [Box, Box]
  tools: [Box, Box]
  play: Box
  status: Box
  blurb: Box
  /** How far the title sits above the panel's foot, where it lands before rising into place. */
  rise: number
  /** The card's corner, at the hero's top, and a panel's. */
  heroRadius: string
  panelRadius: string
}

/** The pieces of the morph, each an actor in the layer or a real element of the full layout. */
export type ActorKey =
  | 'backdrop'
  | 'clip'
  | 'art'
  | 'wash'
  | 'scrim'
  | 'shade'
  | 'label'
  | 'title'
  | 'tools'
  | 'play'
  | 'status'
  | 'blurb'
  | 'actions'
  | 'side'
  | `panel${number}`

/** One piece's keyframes from full (first) to compact (last), and its share of the morph. */
export interface Step {
  key: ActorKey
  /** Where the actor is laid out, at its compact place; absent for a real element. */
  box?: Box
  kf: Keyframe[]
  /** The start and end of its window, as shares of the whole. */
  t: [number, number]
  ease: string
  /**
   * How it leaves, where the arrival run backwards would not do: its window (for
   * a step that ends after the travel), its own keyframes, its own curve.
   */
  away?: { t?: [number, number]; kf?: Keyframe[]; ease?: string }
}

export interface Curves {
  /** The travel: an even ease in and out, so nothing races through the middle. */
  travel: string
  /** Starts from rest, so the rise picks up where the travel stopped without a jolt. */
  settle: string
  /** Away and gone: the sidebar's exit. */
  leave: string
  /** Arrives fast and settles: Play and the server line. */
  out: string
}

/** The travel ends here. After it the title rises and Play fades in. */
export const TRAVEL_END = 0.62

const HOME = 'translate(0px, 0px) scale(1)'
const px = (n: number) => `${Math.round(n * 100) / 100}px`
const inset = (r: Box, m: MorphMeasure, radius: string) =>
  `inset(${px(r.y)} ${px(m.W - r.x - r.w)} ${px(m.H - r.y - r.h)} ${px(r.x)} round ${radius})`
const fromTo = (a: Box, b: Box, scale: boolean) =>
  `translate(${px(a.x - b.x)}, ${px(a.y - b.y)}) scale(${scale ? a.w / b.w : 1})`
/** A box as layout, not a transform: animated on the same thread as the clip-path. */
const rect = (r: Box): Keyframe => ({
  left: px(r.x),
  top: px(r.y),
  width: px(r.w),
  height: px(r.h),
})

/** The picture, its frame, the scrims and the other chapters' panels. */
function planPictures(m: MorphMeasure, c: Curves): Step[] {
  const A = TRAVEL_END
  const steps: Step[] = [
    // The compact layout's ground, under the panels, so their gaps are not the card's.
    { key: 'backdrop', kf: [{ opacity: 0 }, { opacity: 1 }], t: [0.34, A], ease: 'linear' },
    {
      key: 'clip',
      kf: [
        { clipPath: inset(m.hero, m, `${m.heroRadius} ${m.heroRadius} 0px 0px`) },
        { clipPath: inset(m.panel, m, m.panelRadius) },
      ],
      t: [0, A],
      ease: c.travel,
    },
    {
      key: 'art',
      box: m.artCompact,
      kf: [{ transform: fromTo(m.artFull, m.artCompact, true) }, { transform: 'none' }],
      t: [0, A],
      ease: c.travel,
    },
    // The washes over the picture fill its frame all the way, the hero's turning into the
    // panel's. They move as layout, as the clip does: a transform runs on the compositor
    // (WebKit's Core Animation), the clip-path does not, and the two drift apart by a few
    // pixels. They also reach a little past the frame (style.css, `.morph-wash`), so the
    // clip alone cuts the picture's edge.
    {
      key: 'wash',
      box: m.panel,
      kf: [rect(m.hero), rect(m.panel)],
      t: [0, A],
      ease: c.travel,
    },
    // One wash comes in before the other goes, so the picture is never lighter than at either end.
    { key: 'shade', kf: [{ opacity: 0 }, { opacity: 1 }], t: [0, 0.4], ease: 'linear' },
    { key: 'scrim', kf: [{ opacity: 1 }, { opacity: 0 }], t: [0.25, A], ease: 'linear' },
  ]
  // The other chapters come in from the side they sit on: their far edge starts at the window's edge.
  m.panels.forEach((r, i) => {
    if (i === m.active) return
    const dx = i < m.active ? -(r.x + r.w) : m.W - r.x
    steps.push({
      key: `panel${i}`,
      box: r,
      kf: [{ transform: `translateX(${px(dx)})` }, { transform: 'none' }],
      t: [0, A],
      ease: c.travel,
    })
  })
  return steps
}

/** The title, its label and the tools, Play and the server line, and what leaves first. */
function planWords(m: MorphMeasure, c: Curves): Step[] {
  const A = TRAVEL_END
  // Only the title changes size; the rest only moves.
  const travelThenRise = (key: 'label' | 'title', [a, b]: [Box, Box]): Step => ({
    key,
    box: b,
    kf: [
      { transform: fromTo(a, b, key === 'title'), offset: 0, easing: c.travel },
      { transform: `translate(0px, ${px(m.rise)}) scale(1)`, offset: A, easing: c.settle },
      { transform: HOME, offset: 1 },
    ],
    t: [0, 1],
    ease: 'linear',
    // Leaving, it travels from where it stands, with the picture.
    away: {
      t: [0, A],
      kf: [{ transform: HOME }, { transform: fromTo(a, b, key === 'title') }],
      ease: c.travel,
    },
  })
  const fadeUp = (key: 'play' | 'status', box: Box, from: number, gone: number): Step => ({
    key,
    box,
    // A soft fade with a small lift, on an even curve: an ease-out would pop it in.
    kf: [
      { opacity: 0, transform: 'translateY(8px)' },
      { opacity: 1, transform: 'none' },
    ],
    t: [from, 1],
    ease: c.settle,
    // Leaving, it fades as the travel starts, while the picture is still near its panel.
    away: { t: [0, gone], ease: 'linear' },
  })
  const leave = (key: 'actions' | 'blurb', drop: number, until: number, box?: Box): Step => ({
    key,
    ...(box && { box }),
    kf: [
      { opacity: 1, transform: 'none' },
      { opacity: 0, transform: `translateY(${drop}px)` },
    ],
    t: [0, until],
    ease: 'linear',
  })
  const [toolsFrom, toolsTo] = m.tools
  return [
    travelThenRise('label', m.label),
    travelThenRise('title', m.title),
    {
      key: 'tools',
      box: toolsTo,
      kf: [{ transform: fromTo(toolsFrom, toolsTo, false) }, { transform: HOME }],
      t: [0, A],
      ease: c.travel,
    },
    fadeUp('play', m.play, 0.64, 0.2),
    fadeUp('status', m.status, 0.7, 0.16),
    // The card's own Play row and the blurb leave before the picture covers them.
    leave('actions', 8, 0.14),
    leave('blurb', 6, 0.16, m.blurb),
    {
      key: 'side',
      kf: [{ transform: 'none' }, { transform: 'translateX(-100%)' }],
      t: [0, 0.34],
      ease: c.leave,
    },
  ]
}

/**
 * The whole morph, as keyframes from the full layout to the compact one. Leaving,
 * the travel keeps its own curve rather than its mirror, which would start slowly:
 * the picture moves off the moment the click lands, and settles into the hero.
 */
export function planMorph(m: MorphMeasure, c: Curves): Step[] {
  return [...planPictures(m, c), ...planWords(m, c)].map((s) =>
    s.ease === c.travel ? { ...s, away: { ...s.away, ease: c.travel } } : s,
  )
}

/** A step's keyframes played in either direction. */
export function keyframesFor(s: Step, toCompact: boolean): Keyframe[] {
  if (toCompact) return s.kf
  return s.away?.kf ?? reverseKeyframes(s.kf)
}

/**
 * A step's timing played in either direction. Going out, its window and its curve
 * mirror within the travel, which the way out lasts, or it takes its own (`away`).
 */
export function timing(s: Step, toCompact: boolean, duration: number): KeyframeAnimationOptions {
  const [a, b] = toCompact ? s.t : (s.away?.t ?? [TRAVEL_END - s.t[1], TRAVEL_END - s.t[0]])
  const out = s.away?.ease ?? mirror(s.ease)
  return {
    duration: duration * (b - a),
    delay: duration * a,
    easing: toCompact ? s.ease : out,
    fill: 'both',
  }
}
