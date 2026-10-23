import { describe, expect, it } from 'vitest'
import { TRAVEL_END, cover, mirror, planMorph, reverseKeyframes, timing } from './compactMorph'
import type { Box, Curves, MorphMeasure } from './compactMorph'

const box = (x: number, y: number, w: number, h: number): Box => ({ x, y, w, h })
const curves: Curves = {
  travel: 'cubic-bezier(0.4, 0, 0.2, 1)',
  settle: 'cubic-bezier(0.33, 0, 0.2, 1)',
  leave: 'cubic-bezier(0.55, 0, 0.75, 0.2)',
  out: 'cubic-bezier(0.16, 1, 0.3, 1)',
}

/** A 1280 × 760 area with the middle chapter open. */
function measure(): MorphMeasure {
  const panels = [box(4, 4, 274, 752), box(282, 4, 712, 752), box(998, 4, 278, 752)]
  return {
    W: 1280,
    H: 760,
    hero: box(280, 16, 984, 398),
    panel: panels[1]!,
    artFull: box(280, -40, 984, 525),
    artCompact: box(-50, 4, 1410, 752),
    panels,
    active: 1,
    label: [box(308, 210, 300, 20), box(306, 600, 300, 20)],
    title: [box(308, 236, 400, 68), box(306, 630, 400, 68)],
    tools: [box(1104, 44, 132, 44), box(946, 28, 132, 44)],
    play: box(306, 712, 240, 52),
    status: box(560, 720, 120, 30),
    blurb: box(308, 318, 520, 80),
    rise: 64,
    heroRadius: '10px',
    panelRadius: '8px',
  }
}

describe('cover', () => {
  it('fills the box at the picture own aspect, centred across and 40% down', () => {
    const r = cover(box(0, 0, 100, 100), [200, 100])
    expect(r).toEqual({ x: -50, y: 0, w: 200, h: 100 })
    const tall = cover(box(0, 0, 100, 100), [100, 200])
    expect(tall.y).toBeCloseTo(-40)
  })
})

describe('mirror', () => {
  it('turns an ease-out into the matching ease-in, and keeps anything else', () => {
    expect(mirror('cubic-bezier(0.16, 1, 0.3, 1)')).toBe('cubic-bezier(0.7, 0, 0.84, 0)')
    expect(mirror(mirror(curves.travel))).toBe('cubic-bezier(0.4, 0, 0.2, 1)')
    expect(mirror('linear')).toBe('linear')
  })
})

describe('reverseKeyframes', () => {
  it('plays from the other end, offsets and segment curves mirrored', () => {
    const kf: Keyframe[] = [
      { transform: 'a', offset: 0, easing: curves.travel },
      { transform: 'b', offset: 0.62, easing: curves.settle },
      { transform: 'c', offset: 1 },
    ]
    const back = reverseKeyframes(kf)
    expect(back.map((k) => k.transform)).toEqual(['c', 'b', 'a'])
    expect(back.map((k) => k.offset)).toEqual([0, 0.38, 1])
    expect(back[0]!.easing).toBe(mirror(curves.settle))
    expect(back[1]!.easing).toBe(mirror(curves.travel))
    expect(back[2]!.easing).toBeUndefined()
  })
})

describe('planMorph', () => {
  const steps = planMorph(measure(), curves)
  const step = (key: string) => steps.find((s) => s.key === key)!

  it('brings each other chapter in from the side it sits on', () => {
    expect(step('panel0').kf[0]!.transform).toBe('translateX(-278px)')
    expect(step('panel2').kf[0]!.transform).toBe('translateX(282px)')
    expect(steps.some((s) => s.key === 'panel1')).toBe(false)
  })

  it('crops the frame from the hero to the panel while the picture travels with it', () => {
    expect(step('clip').kf[0]!.clipPath).toBe(
      'inset(16px 16px 346px 280px round 10px 10px 0px 0px)',
    )
    expect(step('clip').kf[1]!.clipPath).toBe('inset(4px 286px 4px 282px round 8px)')
    for (const k of ['clip', 'art', 'panel0', 'panel2', 'tools']) {
      expect(step(k).t).toEqual([0, TRAVEL_END])
      expect(step(k).ease).toBe(curves.travel)
    }
  })

  it('carries the washes over the picture with its frame, stretched from the hero to the panel', () => {
    const wash = step('wash')
    expect(wash.box).toEqual(measure().panel)
    expect(wash.kf[0]!.transform).toBe(`translate(-2px, 12px) scale(${984 / 712}, ${398 / 752})`)
    expect(wash.kf[1]!.transform).toBe('none')
    expect(wash.t).toEqual([0, TRAVEL_END])
    // The washes themselves only fade, inside it.
    expect(step('scrim').box).toBeUndefined()
    expect(step('shade').box).toBeUndefined()
    // The panel's wash is in before the hero's starts to go: the picture never shows lighter.
    expect(step('shade').t[1]).toBeGreaterThan(step('scrim').t[0])
    expect(step('shade').t[0]).toBe(0)
    expect(step('scrim').t[1]).toBe(TRAVEL_END)
  })

  it('takes the title to the panel foot with the picture, then lets it rise as Play fades in', () => {
    const title = step('title')
    expect(title.kf[0]!.transform).toBe('translate(2px, -394px) scale(1)')
    expect(title.kf[1]).toMatchObject({
      offset: TRAVEL_END,
      transform: 'translate(0px, 64px) scale(1)',
    })
    expect(title.kf[2]!.transform).toBe('translate(0px, 0px) scale(1)')
    // The label only moves; it never scales.
    expect(step('label').kf[0]!.transform).toMatch(/scale\(1\)$/)
    expect(step('play').t[0]).toBeGreaterThan(TRAVEL_END)
    expect(step('status').t[0]).toBeGreaterThan(step('play').t[0])
  })

  it('lets the card own Play row, the blurb and the sidebar go first', () => {
    expect(step('actions').t[1]).toBeLessThan(0.2)
    expect(step('blurb').t[1]).toBeLessThan(0.2)
    expect(step('side').kf[1]!.transform).toBe('translateX(-100%)')
    expect(step('side').box).toBeUndefined()
  })
})

describe('timing', () => {
  const s = planMorph(measure(), curves).find((x) => x.key === 'play')!

  it('places a step in its window of the whole', () => {
    const t = timing(s, true, 1000)
    expect(t).toMatchObject({ delay: 700, easing: curves.out, fill: 'both' })
    expect(t.duration).toBeCloseTo(300)
  })

  it('going out, mirrors the window and the curve', () => {
    const t = timing(s, false, 1000)
    expect(t.delay).toBe(0)
    expect(t.duration).toBeCloseTo(300)
    expect(t.easing).toBe(mirror(curves.out))
  })
})
