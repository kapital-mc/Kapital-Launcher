import { Suspense, lazy, useEffect, useState } from 'react'
import { tailscaleStep } from '../../lib/tailscale'
import { tailscaleWorking, useTailscaleStore } from '../../stores/useTailscaleStore'
import { ErrorLine } from '../ui/Notes'
import { Grow } from '../ui/Grow'
import { TextLink } from '../ui/TextLink'
import { Toggle } from '../ui/Toggle'

// The popup loads only when the player asks for Tailscale.
const GetTailscale = lazy(() => import('./GetTailscale').then((m) => ({ default: m.GetTailscale })))

interface Props {
  /** The chapter whose server the row is about. */
  chapterId: string
  /** Whether the chapter's Tailscale switch is on. */
  on: boolean
  /** The settings store's refusal, under the switch. */
  error?: string
  /** Saves the switch; the section owns the settings write. */
  onChange: (on: boolean) => void
}

/**
 * The Tailscale row of a chapter whose server has a Tailscale address, between
 * Address and Join. It is the next step until there is none: Install while
 * Tailscale is missing (the popup that gets it), Connect while it does not
 * reach this server (Go opens Tailscale, signs in or opens the server's share
 * link, whichever is due), and the switch once it does. On, the address
 * becomes a choice under Address and the server is added to the in-game
 * server list; off, both go.
 */
export function TailscaleRow({ chapterId, on, error, onChange }: Props) {
  const info = useTailscaleStore((s) => s.info)
  const release = useTailscaleStore((s) => s.release)
  const install = useTailscaleStore((s) => s.install)
  const storeError = useTailscaleStore((s) => s.error)
  const load = useTailscaleStore((s) => s.load)
  const loadRelease = useTailscaleStore((s) => s.loadRelease)
  const installTailscale = useTailscaleStore((s) => s.installTailscale)
  const connect = useTailscaleStore((s) => s.connect)
  const open = useTailscaleStore((s) => s.open)
  const [offering, setOffering] = useState(false)
  const offerPending = useTailscaleStore((s) => s.offerPending)

  useEffect(() => {
    void load()
  }, [load])
  // A developer preview's request for the popup, taken once.
  useEffect(() => {
    if (!offerPending) return
    useTailscaleStore.setState({ offerPending: false })
    setOffering(true)
  }, [offerPending])
  const step = tailscaleStep(info, chapterId)
  const canInstall = info?.canInstall === true
  useEffect(() => {
    if (step === 'install' && canInstall) void loadRelease()
  }, [step, canInstall, loadRelease])

  // Without a release to show (offline, or an OS it cannot install on),
  // Install opens the download page through Go.
  const get = () => (canInstall && release ? setOffering(true) : void open())
  const busy = tailscaleWorking(install) || info?.signingIn === true
  const failure = error ?? storeError

  return (
    <div className="flex flex-col gap-1">
      {step === 'ready' ? (
        <Toggle label="Tailscale" checked={on} error={error} onChange={onChange} />
      ) : (
        <div className="flex min-h-6 items-center justify-between gap-6">
          <span className="text-fg text-sm font-medium">Tailscale</span>
          {step && (
            <TextLink
              strong
              disabled={busy}
              onClick={step === 'install' ? get : () => void connect(chapterId)}
              className="disabled:text-fg-faint disabled:no-underline"
            >
              {step === 'install' ? 'Install' : 'Connect'}
            </TextLink>
          )}
        </div>
      )}
      <Grow className="-mt-1" inner="pt-1">
        {step === 'ready'
          ? storeError && <ErrorLine>{storeError}</ErrorLine>
          : failure && <ErrorLine>{failure}</ErrorLine>}
      </Grow>
      {offering && release && (
        <Suspense fallback={null}>
          <GetTailscale
            release={release}
            install={install}
            info={info}
            onConfirm={() => void installTailscale()}
            onClose={() => setOffering(false)}
            onConnect={() => void connect(chapterId)}
          />
        </Suspense>
      )}
    </div>
  )
}
