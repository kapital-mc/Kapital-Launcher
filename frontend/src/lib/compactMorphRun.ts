import { boxOf, cover, planMorph, reverseKeyframes, timing } from './compactMorph'
import type { ActorKey, Box, Curves, MorphMeasure, Step } from './compactMorph'
import { ms, token } from './smoothResize'

/**
 * Runs the morph of lib/compactMorph in the page: reads both layouts, builds
 * the actors, plays them, and hands back a handle to end it. The layouts mark
 * their pieces with `data-morph`; the full one is `data-layout="full"`, the
 * compact one `data-layout="compact"`, the open chapter's card `data-morph-card`
 * and its panel `data-active`.
 */

const sel = (name: string) => `[data-morph="${name}"]`

interface Pieces {
  full: HTMLElement
  hero: HTMLElement
  side: HTMLElement
  actions: HTMLElement
  panels: HTMLElement[]
  active: number
  on: HTMLElement
  art: HTMLElement
}

/** The marked pieces of both layouts, or null when one is missing (nothing to morph). */
function find(area: HTMLElement): Pieces | null {
  const full = area.querySelector<HTMLElement>('[data-layout="full"]')
  const compact = area.querySelector<HTMLElement>('[data-layout="compact"]')
  const card = full?.querySelector<HTMLElement>('[data-morph-card]')
  const hero = card?.querySelector<HTMLElement>(sel('hero'))
  const actions = card?.querySelector<HTMLElement>(sel('actions'))
  const side = full?.querySelector<HTMLElement>(sel('side'))
  const panels = [...(compact?.querySelectorAll<HTMLElement>(sel('panel')) ?? [])]
  const active = panels.findIndex((p) => p.dataset.active !== undefined)
  const on = panels[active]
  const art = on?.querySelector<HTMLElement>(sel('panel-art'))
  if (!full || !hero || !actions || !side || !on || !art) return null
  return { full, hero, side, actions, panels, active, on, art }
}

/** The picture's natural size; a box with none (the pending grid) keeps the panel's aspect. */
function naturalOf(art: HTMLElement, panel: Box): [number, number] {
  if (art instanceof HTMLImageElement && art.naturalWidth > 0)
    return [art.naturalWidth, art.naturalHeight]
  return [panel.w, panel.h]
}

function measure(area: HTMLElement, p: Pieces): MorphMeasure | null {
  const box = (el: Element | null | undefined) => (el ? boxOf(el, area) : null)
  const q = (root: HTMLElement, name: string) => box(root.querySelector(sel(name)))
  const pairs = ['label', 'title', 'tools'].map((k) => [
    q(p.hero, `hero-${k}`),
    q(p.on, `panel-${k}`),
  ])
  const play = q(p.on, 'panel-play')
  const status = q(p.on, 'panel-status')
  const body = q(p.on, 'panel-body')
  const blurb = q(p.hero, 'hero-blurb')
  if (!play || !status || !body || !blurb || pairs.some(([a, b]) => !a || !b)) return null
  const [label, title, tools] = pairs as [Box, Box][]
  const hero = boxOf(p.hero, area)
  const panel = boxOf(p.on, area)
  const nat = naturalOf(p.art, panel)
  return {
    W: area.clientWidth,
    H: area.clientHeight,
    hero,
    panel,
    artFull: cover(hero, nat),
    artCompact: cover(panel, nat),
    panels: p.panels.map((el) => boxOf(el, area)),
    active: p.active,
    label: label!,
    title: title!,
    tools: tools!,
    play,
    status,
    blurb,
    rise: body.y + body.h - (title![1].y + title![1].h),
    heroRadius: token('--radius-lg'),
    panelRadius: token('--radius-md'),
  }
}

const place = (el: HTMLElement, r: Box) =>
  Object.assign(el.style, {
    left: `${r.x}px`,
    top: `${r.y}px`,
    width: `${r.w}px`,
    height: `${r.h}px`,
  })

/**
 * A copy of `el` for the layer, without ids or marks that would be found again.
 * A piece of the open panel also drops its `compact-*` look, which depends on
 * sitting in the open panel; a whole panel keeps it.
 */
function copy(el: Element, whole = false): HTMLElement {
  const c = el.cloneNode(true) as HTMLElement
  for (const e of [c, ...c.querySelectorAll('[data-morph],[id]')]) {
    e.removeAttribute('data-morph')
    e.removeAttribute('id')
  }
  if (!whole) c.classList.remove(...[...c.classList].filter((n) => n.startsWith('compact-')))
  return c
}

/** The actors, each keyed as the plan names it; the sidebar and the card's Play row are the real ones. */
function buildActors(layer: HTMLElement, p: Pieces): Map<ActorKey, HTMLElement> {
  const els = new Map<ActorKey, HTMLElement>()
  layer.replaceChildren()
  const div = (className: string) => Object.assign(document.createElement('div'), { className })
  const backdrop = div('morph-backdrop')
  const clip = div('morph-clip')
  const art = copy(p.art)
  const scrim = div('scrim-hero')
  const shade = copy(p.on.querySelector(sel('panel-shade')) ?? div('scrim-foot'))
  clip.append(art, scrim, shade)
  layer.append(backdrop, clip)
  els.set('backdrop', backdrop).set('clip', clip).set('art', art).set('scrim', scrim)
  els.set('shade', shade)
  p.panels.forEach((panel, i) => {
    if (i === p.active) return
    const k = copy(panel, true)
    layer.append(k)
    els.set(`panel${i}`, k)
  })
  for (const k of ['label', 'title', 'tools', 'play', 'status'] as const) {
    const e = copy(p.on.querySelector(sel(`panel-${k}`))!)
    layer.append(e)
    els.set(k, e)
  }
  const blurb = copy(p.hero.querySelector(sel('hero-blurb'))!)
  layer.append(blurb)
  els.set('blurb', blurb).set('side', p.side).set('actions', p.actions)
  return els
}

/** The morph's curves, from the motion tokens. */
const curves = (): Curves => ({
  travel: token('--ease-travel'),
  settle: token('--ease-settle'),
  leave: token('--ease-leave'),
  out: token('--ease-out'),
})

export interface Morph {
  /** Settles when every piece has arrived. */
  finished: Promise<void>
  /** Takes the actors down and lets the real elements go: call once the layout has swapped. */
  end: () => void
}

/**
 * Plays the morph into compact or out of it, over `layer`. Null when a piece is
 * missing, so the caller swaps the layouts at once instead.
 */
export function runMorph(area: HTMLElement, layer: HTMLElement, toCompact: boolean): Morph | null {
  const pieces = find(area)
  const m = pieces && measure(area, pieces)
  if (!pieces || !m) return null
  const els = buildActors(layer, pieces)
  const duration = ms('--duration-morph')
  const c = curves()
  const keyframes = (s: Step) => (toCompact ? s.kf : reverseKeyframes(s.kf))
  const steps = planMorph(m, c)
  const anims = steps.map((s) => {
    const el = els.get(s.key)!
    if (s.box) place(el, s.box)
    const a = el.animate(keyframes(s), timing(s, toCompact, duration))
    // Held at its first frame until every picture is decoded, so no frame is dropped.
    a.pause()
    return a
  })
  pieces.hero.style.visibility = 'hidden'

  // The window changed size mid-way: measure again and move every piece's end, keeping its progress.
  const resize = new ResizeObserver(() => {
    const again = measure(area, pieces)
    if (!again) return
    planMorph(again, c).forEach((s, i) => {
      if (s.box) place(els.get(s.key)!, s.box)
      ;(anims[i]?.effect as KeyframeEffect | null)?.setKeyframes(keyframes(s))
    })
  })
  resize.observe(area)

  const pictures = [...layer.querySelectorAll('img')].map((i) => i.decode().catch(() => undefined))
  const finished = Promise.all(pictures)
    .then(() => new Promise((r) => requestAnimationFrame(r)))
    .then(() => {
      for (const a of anims) a.play()
      return Promise.all(anims.map((a) => a.finished))
    })
    .then(() => undefined)
  const end = () => {
    resize.disconnect()
    for (const a of anims) a.cancel()
    pieces.hero.style.removeProperty('visibility')
    layer.replaceChildren()
  }
  return { finished, end }
}
