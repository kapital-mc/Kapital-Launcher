import { afterEach, describe, expect, it, vi } from 'vitest'
import * as Runtime from '../../wailsjs/runtime/runtime'
import { EVENT_LAUNCHER_UPDATE, useUpdateStore } from './useUpdateStore'

vi.mock('../../wailsjs/go/main/App')
vi.mock('../../wailsjs/runtime/runtime')

describe('useUpdateStore', () => {
  afterEach(() => {
    Reflect.deleteProperty(window, 'go')
    vi.mocked(Runtime.EventsOn).mockReset()
    vi.mocked(Runtime.EventsOff).mockReset()
  })

  it('listens to nothing and returns a function without a bridge', () => {
    const off = useUpdateStore.getState().listen()
    expect(typeof off).toBe('function')
    expect(() => off()).not.toThrow()
    expect(Runtime.EventsOn).not.toHaveBeenCalled()
    expect(Runtime.EventsOff).not.toHaveBeenCalled()
  })

  it('follows the launcher:update event with a bridge', () => {
    Object.assign(window, { go: {} })
    const off = useUpdateStore.getState().listen()
    expect(Runtime.EventsOn).toHaveBeenCalledWith(EVENT_LAUNCHER_UPDATE, expect.any(Function))
    off()
    expect(Runtime.EventsOff).toHaveBeenCalledWith(EVENT_LAUNCHER_UPDATE)
  })
})
