import { describe, expect, it } from 'vitest'
import { fillOf, isDownloading, publishedOn, stepLabel } from './terrain'
import type { DHDownloadProgress } from '../types'

const p = (over: Partial<DHDownloadProgress>): DHDownloadProgress => ({
  chapterId: 'frangfurd',
  phase: 'downloading',
  received: 0,
  total: 0,
  ...over,
})

describe('terrain helpers', () => {
  it('fills by bytes, full while checking and placing, and never past the ends', () => {
    expect(fillOf(p({ received: 1, total: 4 }))).toBe(0.25)
    expect(fillOf(p({ received: 9, total: 4 }))).toBe(1)
    expect(fillOf(p({ received: 1, total: 0 }))).toBe(0)
    expect(fillOf(p({ phase: 'verifying' }))).toBe(1)
    expect(fillOf(p({ phase: 'placing' }))).toBe(1)
  })

  it('tells a running download by its progress, or by Go before any', () => {
    expect(isDownloading(undefined, p({}))).toBe(true)
    expect(isDownloading(undefined, p({ phase: 'failed' }))).toBe(false)
    expect(isDownloading(undefined, undefined)).toBe(false)
    expect(
      isDownloading(
        {
          chapterId: 'frangfurd',
          available: false,
          reason: '',
          createdAt: '',
          sizeBytes: 0,
          dimensions: 0,
          serverDhVersion: '',
          clientDhVersion: '',
          existingBytes: 0,
          partialBytes: 0,
          running: false,
          downloading: true,
        },
        undefined,
      ),
    ).toBe(true)
  })

  it('names each step, and dates a snapshot', () => {
    expect(stepLabel('verifying')).toBe('Checking the files')
    expect(stepLabel('anything')).toBe('Downloading')
    expect(publishedOn('2026-10-13T12:03:41Z', 'en-GB')).toBe('13 Oct 2026')
    expect(publishedOn('soon')).toBe('soon')
  })
})
