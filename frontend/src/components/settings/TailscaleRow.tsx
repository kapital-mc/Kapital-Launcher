import { Suspense, lazy, useEffect, useState } from 'react'
import { hasWailsBridge } from '../../lib/ipc'
import { guessPlatform } from '../../lib/platform'
import { tailscaleAction, tailscaleHint } from '../../lib/tailscale'
import { tailscaleWorking, useTailscaleStore } from '../../stores/useTailscaleStore'
import { Button } from '../ui/Button'
import { ErrorLine } from '../ui/Notes'
import { Toggle } from '../ui/Toggle'

// The popup loads only when the player asks for Tailscale.
const GetTailscale = lazy(() => import('./GetTailscale').then((m) => ({ default: m.GetTailscale })))

interface Props {
  /** Whether the chapter's Tailscale switch is on. */
  on: boolean
  /** The settings store's refusal, under the switch. */
  error?: string
  /** Saves the switch; the section owns the settings write. */
  onChange: (on: boolean) => void
}

const ACTION_LABELS = { get: 'Get Tailscale', open: 'Open Tailscale', signIn: 'Sign in' } as const

/**
 * The Tailscale switch of a chapter whose server has a Tailscale address,
 * between Address and Join. On, the address becomes a choice under Address
 * and the server is added to the in-game server list; off, both go. Without
 * Tailscale, turning it on opens the popup that gets it, and the switch turns
 * on once the installer is done. The button beside it is the next step:
 * Get Tailscale, Open Tailscale or Sign in.
 */
export function TailscaleRow({ on, error, onChange }: Props) {
  const info = useTailscaleStore((s) => s.info)
  const release = useTailscaleStore((s) => s.release)
  const install = useTailscaleStore((s) => s.install)
  const storeError = useTailscaleStore((s) => s.error)
  const load = useTailscaleStore((s) => s.load)
  const loadRelease = useTailscaleStore((s) => s.loadRelease)
  const installTailscale = useTailscaleStore((s) => s.installTailscale)
  const signIn = useTailscaleStore((s) => s.signIn)
  const open = useTailscaleStore((s) => s.open)
  const [offering, setOffering] = useState(false)
  // The switch the player turned on without Tailscale, saved once it is installed.
  const [pending, setPending] = useState(false)

  useEffect(() => {
    void load()
  }, [load])
  const missing = info?.state === 'missing'
  const canInstall = info?.canInstall === true
  useEffect(() => {
    if (missing && canInstall) void loadRelease()
  }, [missing, canInstall, loadRelease])
  useEffect(() => {
    if (!pending || install?.phase !== 'done') return
    setPending(false)
    onChange(true)
  }, [pending, install?.phase, onChange])

  // Without a release to show (offline, or an OS it cannot install on), Get
  // Tailscale opens the download page through Go.
  const get = () => (canInstall && release ? setOffering(true) : void open())
  const action = tailscaleAction(info)
  const run = { get, open: () => void open(), signIn: () => void signIn() }
  const toggle = (next: boolean) => {
    if (next && missing) {
      setPending(true)
      get()
      return
    }
    onChange(next)
  }

  return (
    <div className="flex flex-col gap-3">
      <Toggle
        label="Tailscale"
        checked={on}
        hint={tailscaleHint(info, on)}
        error={error}
        disabled={tailscaleWorking(install)}
        onChange={toggle}
      />
      {action && (
        <div>
          <Button onClick={run[action]} disabled={tailscaleWorking(install)}>
            {ACTION_LABELS[action]}
          </Button>
        </div>
      )}
      {storeError && <ErrorLine>{storeError}</ErrorLine>}
      {offering && release && (
        <Suspense fallback={null}>
          <GetTailscale
            release={release}
            install={install}
            info={info}
            platform={guessPlatform(navigator.userAgent, hasWailsBridge())}
            onConfirm={() => void installTailscale()}
            onClose={() => {
              setOffering(false)
              if (install?.phase !== 'done' && !tailscaleWorking(install)) setPending(false)
            }}
            onSignIn={() => void signIn()}
            onOpen={() => void open()}
          />
        </Suspense>
      )}
    </div>
  )
}
