import { describe, expect, it } from 'vitest'
import { downloadValue, installStage } from './installStage'

const step = (phase: string, received = 0, total = 0, error = '') => ({
  phase,
  received,
  total,
  error,
})

describe('installStage', () => {
  it('offers before Download, then follows the install to its end', () => {
    const line = (p: { phase: string }) => `step ${p.phase}`
    expect(installStage(null, line)).toEqual({ kind: 'offer' })
    expect(installStage(step('downloading', 5, 20), line)).toEqual({
      kind: 'working',
      value: 0.25,
      line: 'step downloading',
    })
    expect(installStage(step('done'), line)).toEqual({ kind: 'done' })
    expect(installStage(step('failed', 0, 0, 'no disk'), line)).toEqual({
      kind: 'failed',
      error: 'no disk',
    })
  })

  it('counts the steps after the download as full, and an unknown size as none', () => {
    expect(downloadValue(step('verifying'))).toBe(1)
    expect(downloadValue(step('downloading', 10, 0))).toBe(0)
  })
})
