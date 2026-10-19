import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as App from '../../wailsjs/go/main/App'
import * as Runtime from '../../wailsjs/runtime/runtime'
import {
  EVENT_TAILSCALE_INSTALL,
  EVENT_TAILSCALE_STATE,
  tailscaleWorking,
  useTailscaleStore,
} from './useTailscaleStore'

vi.mock('../../wailsjs/go/main/App')
vi.mock('../../wailsjs/runtime/runtime')

const running = {
  state: 'running',
  version: '1.104.1',
  canInstall: true,
  signingIn: false,
  connected: ['frangfurd'],
}

describe('useTailscaleStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useTailscaleStore.setState({ info: null, release: null, install: null, error: null })
  })
  afterEach(() => Reflect.deleteProperty(window, 'go'))

  it('reads the state, and nothing when Go cannot answer', async () => {
    vi.mocked(App.GetTailscale).mockResolvedValue(running)
    await useTailscaleStore.getState().load()
    expect(useTailscaleStore.getState().info).toEqual(running)
    vi.mocked(App.GetTailscale).mockRejectedValue(new Error('no'))
    await useTailscaleStore.getState().load()
    expect(useTailscaleStore.getState().info).toBeNull()
  })

  it('settles an install from its promise, whatever the events said', async () => {
    vi.mocked(App.InstallTailscale).mockResolvedValue()
    vi.mocked(App.GetTailscale).mockResolvedValue(running)
    await useTailscaleStore.getState().installTailscale()
    expect(useTailscaleStore.getState().install?.phase).toBe('done')
    vi.mocked(App.InstallTailscale).mockRejectedValue('cancelled')
    await useTailscaleStore.getState().installTailscale()
    expect(useTailscaleStore.getState().install).toMatchObject({
      phase: 'failed',
      error: 'cancelled',
    })
  })

  it('follows the events, and a late one does not reopen a finished install', () => {
    const heard: Record<string, (p: unknown) => void> = {}
    vi.mocked(Runtime.EventsOn).mockImplementation(((name: string, cb: (p: unknown) => void) => {
      heard[name] = cb
      return () => undefined
    }) as never)
    const off = useTailscaleStore.getState().listen()
    heard[EVENT_TAILSCALE_STATE]!(running)
    expect(useTailscaleStore.getState().info).toEqual(running)
    useTailscaleStore.setState({
      install: { phase: 'downloading', received: 0, total: 9, error: '' },
    })
    heard[EVENT_TAILSCALE_INSTALL]!({ phase: 'verifying', received: 9, total: 9, error: '' })
    expect(tailscaleWorking(useTailscaleStore.getState().install)).toBe(true)
    useTailscaleStore.setState({ install: { phase: 'done', received: 0, total: 0, error: '' } })
    heard[EVENT_TAILSCALE_INSTALL]!({ phase: 'installing', received: 9, total: 9, error: '' })
    expect(useTailscaleStore.getState().install?.phase).toBe('done')
    off()
    expect(Runtime.EventsOff).toHaveBeenCalledTimes(2)
  })

  it('records a refused Connect or open', async () => {
    vi.mocked(App.ConnectTailscale).mockRejectedValue(new Error('not installed'))
    await useTailscaleStore.getState().connect('frangfurd')
    expect(App.ConnectTailscale).toHaveBeenCalledWith('frangfurd')
    expect(useTailscaleStore.getState().error).toBe('not installed')
    vi.mocked(App.OpenTailscale).mockResolvedValue()
    await useTailscaleStore.getState().open()
    expect(useTailscaleStore.getState().error).toBeNull()
  })
})
