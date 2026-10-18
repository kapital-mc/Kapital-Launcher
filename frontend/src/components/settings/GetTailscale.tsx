import type {
  TailscaleInfo,
  TailscaleInstallProgress,
  TailscaleRelease,
} from '../../types/tailscale'
import { megabytes } from '../../lib/prismInstall'
import { tailscaleStepLine } from '../../lib/tailscale'
import { Button } from '../ui/Button'
import { installStage } from '../../lib/installStage'
import { InstallPopup } from '../shell/InstallPopup'

interface Props {
  release: TailscaleRelease
  install: TailscaleInstallProgress | null
  info: TailscaleInfo | null
  /** "darwin" or "windows": the two OSes ask for different things. */
  platform: string
  onConfirm: () => void
  onClose: () => void
  onSignIn: () => void
  onOpen: () => void
}

/**
 * Getting Tailscale, in the same popup as getting Prism: the official
 * installer, what the OS will ask, then the next step. Tailscale has windows
 * of its own the launcher cannot hide (an installer, and on macOS the system
 * extension and VPN prompts), so the popup says what to expect instead.
 */
export function GetTailscale({
  release,
  install,
  info,
  platform,
  onConfirm,
  onClose,
  onSignIn,
  onOpen,
}: Props) {
  const mac = platform === 'darwin'
  return (
    <InstallPopup
      title="Get Tailscale"
      facts={`Version ${release.version} · ${megabytes(release.size)} · pkgs.tailscale.com`}
      stage={installStage(install, tailscaleStepLine)}
      intro={
        <>
          <p className="m-0">
            Tailscale is a private network the server can be reached through. The official installer
            comes from Tailscale&apos;s own package server and is checked before it runs.
          </p>
          <p className="m-0">
            {mac
              ? 'macOS asks for your password to install it. Tailscale then asks to add a system extension and a VPN configuration: allow both.'
              : 'Windows asks for administrator approval to install it.'}{' '}
            You sign in once in your browser.
          </p>
        </>
      }
      done={<NextStep info={info} onSignIn={onSignIn} onOpen={onOpen} />}
      onConfirm={onConfirm}
      onClose={onClose}
    />
  )
}

/** What the player does after the installer: start Tailscale, sign in, or nothing. */
function NextStep({
  info,
  onSignIn,
  onOpen,
}: {
  info: TailscaleInfo | null
  onSignIn: () => void
  onOpen: () => void
}) {
  if (info?.signingIn) return <p className="m-0">Finish signing in in your browser.</p>
  switch (info?.state) {
    case 'running':
      return <p className="m-0">Tailscale is connected.</p>
    case 'needsLogin':
      return (
        <>
          <p className="m-0">Tailscale is installed. Sign in to join its network.</p>
          <div>
            <Button onClick={onSignIn}>Sign in</Button>
          </div>
        </>
      )
    case 'missing':
      return <p className="m-0">Tailscale is not installed yet. Finish its installer first.</p>
    default:
      return (
        <>
          <p className="m-0">
            Tailscale is installed. Open it once and allow what it asks for, then sign in.
          </p>
          <div>
            <Button onClick={onOpen}>Open Tailscale</Button>
          </div>
        </>
      )
  }
}
