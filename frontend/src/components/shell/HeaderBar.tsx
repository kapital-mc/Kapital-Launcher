import { useEffect, useState } from 'react'
import type { CSSProperties } from 'react'
import { GetWindowChrome } from '../../../wailsjs/go/main/App'
import { hasWailsBridge, readOr } from '../../lib/ipc'
import { Copy, Minus, Square, X } from '../../lib/icons'
import type { LucideIcon } from '../../lib/icons'
import {
  Quit,
  WindowIsMaximised,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../../wailsjs/runtime/runtime'
import type { WindowChrome } from '../../types'
import { Icon } from '../ui/Icon'
import { HeaderTools } from './HeaderTools'

interface Props {
  /** Wails' `Environment().platform`: "windows", "darwin" or "linux". */
  platform: string
  onOpenSettings?: () => void
  /** The sidebar is compact, for the button's glyph and name. */
  sidebarCompact?: boolean
  /** Narrows the sidebar to its icons, or widens it again. */
  onToggleSidebar?: () => void
}

/**
 * The window's own header bar, replacing the OS title bar
 * (docs/adr/0010-window-chrome.md). Its empty area drags the window and a
 * double-click maximises; anything interactive opts out of the drag.
 *
 * macOS keeps its native traffic lights at the top left, in the unified
 * toolbar's taller bar, so no window buttons are drawn there and the bar takes
 * the height and the lights' edge Go measured from AppKit, which differ
 * between macOS versions (ADR-10 amendment, 2026-10-19). Everywhere else the
 * bar ends in our own minimise, maximise and close. The bar carries no
 * wordmark (ADR-10 amendment, 2026-10-18).
 *
 * Its tools (HeaderTools) open the bar on the left, the sidebar's button
 * first: past the traffic lights on macOS, and elsewhere at the sidebar's own
 * padding, so the buttons line up with the chapter tiles' left edge.
 */
export function HeaderBar({
  platform,
  onOpenSettings,
  sidebarCompact = false,
  onToggleSidebar,
}: Props) {
  const mac = platform === 'darwin'
  const chrome = useWindowChrome(mac)
  return (
    <header
      onDoubleClick={() => bridged(WindowToggleMaximise)}
      // The bar's height and the lights' edge are measured by AppKit at start:
      // the computed-value exception the style rule allows. Until then, and if
      // nothing was measured, the layout tokens stand.
      // eslint-disable-next-line no-restricted-syntax
      style={chrome ? chromeVars(chrome) : undefined}
      className={`bg-sunken border-line flex shrink-0 items-center justify-between border-b [--wails-draggable:drag] ${mac ? 'h-(--layout-titlebar-mac)' : 'h-(--layout-titlebar)'}`}
    >
      <div
        onDoubleClick={(e) => e.stopPropagation()}
        className={`flex h-full items-center [--wails-draggable:no-drag] ${mac ? 'ml-(--layout-controls-mac) pl-3' : 'pl-4'}`}
      >
        <HeaderTools
          size={mac ? 'xs' : 'sm'}
          sidebarCompact={sidebarCompact}
          onToggleSidebar={onToggleSidebar}
          onOpenSettings={onOpenSettings}
        />
      </div>
      {!mac && (
        <div
          onDoubleClick={(e) => e.stopPropagation()}
          className="flex h-full items-center [--wails-draggable:no-drag]"
        >
          <WindowControls />
        </div>
      )}
    </header>
  )
}

/**
 * Where macOS drew its title bar, read once from Go. Null off macOS, without a
 * bridge, and when nothing was measured.
 */
function useWindowChrome(mac: boolean): WindowChrome | null {
  const [chrome, setChrome] = useState<WindowChrome | null>(null)
  useEffect(() => {
    if (!mac) return
    let live = true
    void readOr(GetWindowChrome, null).then((c) => {
      if (live && c && c.titlebarHeight > 0 && c.controlsEnd > 0) setChrome(c)
    })
    return () => {
      live = false
    }
  }, [mac])
  return chrome
}

/** The measured values, as the layout tokens they stand in for. */
function chromeVars(c: WindowChrome): CSSProperties {
  return {
    '--layout-titlebar-mac': `${c.titlebarHeight}px`,
    '--layout-controls-mac': `${c.controlsEnd}px`,
  } as CSSProperties
}

/** Minimise, maximise or restore, and close, sized and ordered as Windows draws them. */
function WindowControls() {
  const [maximised, setMaximised] = useState(false)

  // The restore glyph follows the window, however it got maximised: our
  // button, a double-click, Win+Up or a drag to the top edge. Each of those
  // resizes the page, so resize is the moment to ask.
  useEffect(() => {
    let live = true
    const sync = async () => {
      const m = await readOr(WindowIsMaximised, false)
      if (live) setMaximised(m)
    }
    void sync()
    window.addEventListener('resize', sync)
    return () => {
      live = false
      window.removeEventListener('resize', sync)
    }
  }, [])

  return (
    <div className="ml-1 flex h-full">
      <WindowButton icon={Minus} title="Minimise" onClick={() => bridged(WindowMinimise)} />
      <WindowButton
        icon={maximised ? Copy : Square}
        title={maximised ? 'Restore' : 'Maximise'}
        onClick={() => bridged(WindowToggleMaximise)}
      />
      <WindowButton icon={X} title="Close" onClick={() => bridged(Quit)} danger />
    </div>
  )
}

function WindowButton({
  icon,
  title,
  onClick,
  danger,
}: {
  icon: LucideIcon
  title: string
  onClick: () => void
  danger?: boolean
}) {
  const hover = danger ? 'hover:bg-danger hover:text-canvas' : 'hover:bg-hover hover:text-fg'
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={title}
      className={`text-fg-muted duration-fast ease-standard flex h-full w-11.5 items-center justify-center transition-colors ${hover}`}
    >
      <Icon icon={icon} size="sm" />
    </button>
  )
}

/**
 * Window calls are fire-and-forget and throw synchronously without a Wails
 * bridge. In the browser-only preview there is no window to move, so they do
 * nothing there.
 */
function bridged(call: () => void) {
  if (hasWailsBridge()) call()
}
