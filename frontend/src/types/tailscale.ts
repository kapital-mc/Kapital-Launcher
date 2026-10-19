// What Tailscale's bound methods and events carry (app_tailscale.go).

/** The player's Tailscale as its command line reports it, from GetTailscale and tailscale:state. */
export interface TailscaleInfo {
  state: 'missing' | 'stopped' | 'starting' | 'needsLogin' | 'needsApproval' | 'running' | string
  version: string
  /** Whether the launcher can install Tailscale here; elsewhere Get Tailscale opens its download page. */
  canInstall: boolean
  /** A sign-in the launcher started is waiting on the browser. */
  signingIn: boolean
  /** The chapters whose Tailscale address this machine reaches: the share link was accepted. */
  connected: string[]
}

/** The installer InstallTailscale would download, from Tailscale's package server. */
export interface TailscaleRelease {
  version: string
  file: string
  url: string
  size: number
  digest: string
}

/** One tailscale:install step: "installing" is Tailscale's own installer running. */
export interface TailscaleInstallProgress {
  phase: 'downloading' | 'verifying' | 'installing' | 'done' | 'failed' | string
  received: number
  total: number
  error: string
}
