import { describe, expect, it } from 'vitest'
import { canReset, settingsDetail, settingsFiles } from './gameSettings'

const item = (id: string, files: string[], keys = 0) => ({ id, name: id, files, keys })

describe('gameSettings', () => {
  it('names one file, counts more, and says Default when nothing is left', () => {
    expect(settingsDetail(item('distantHorizons', ['config/DistantHorizons.toml']))).toBe(
      'DistantHorizons.toml',
    )
    expect(settingsDetail(item('video', ['config/a.json', 'config/b.json']))).toBe('2 files')
    expect(settingsDetail(item('video', []))).toBe('Default')
    expect(canReset(item('video', []))).toBe(false)
  })

  it('counts key bindings, not the file they are in', () => {
    expect(settingsDetail(item('keys', ['options.txt'], 1))).toBe('1 binding')
    expect(settingsDetail(item('keys', ['options.txt'], 102))).toBe('102 bindings')
    expect(settingsDetail(item('keys', [], 0))).toBe('Default')
    expect(canReset(item('keys', ['options.txt'], 0))).toBe(false)
  })

  it('lists the files for the tooltip', () => {
    expect(settingsFiles(item('video', ['config/a.json', 'config/b.json']))).toBe(
      'config/a.json\nconfig/b.json',
    )
    expect(settingsFiles(item('video', []))).toBe('')
  })
})
