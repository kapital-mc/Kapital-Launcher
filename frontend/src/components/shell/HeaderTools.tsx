import { OpenExternal } from '../../../wailsjs/go/main/App'
import { errMsg } from '../../lib/ipc'
import { Suspense, lazy } from 'react'
import { BookOpen, Settings, Triangle } from '../../lib/icons'
import { useChapterStore } from '../../stores/useChapterStore'
import { useEngineStore } from '../../stores/useEngineStore'
import { IconButton } from '../ui/IconButton'
import { SidebarButton } from './SidebarButton'
import type { LauncherChoice } from './SidebarButton'

// The check for a newer launcher, its button and its panel load in a chunk of their own after the
// first paint, which keeps them out of the bundle budget.
const UpdateNotice = lazy(() => import('./UpdateNotice').then((m) => ({ default: m.UpdateNotice })))

interface Props {
  /** The header's button size: `xs` in macOS's shorter bar. */
  size: 'xs' | 'sm'
  sidebarCompact: boolean
  onToggleSidebar?: () => void
  onOpenSettings?: () => void
  launcher?: LauncherChoice
}

/**
 * The tools at the left of the header bar: the sidebar's button (with the
 * compact launcher in its right-click menu), then the
 * settings, Prism itself (for accounts and mods, greyed out while there is no
 * Prism), the wiki's home page, and, only while a newer launcher is
 * out, a button in the accent with its panel under it (`UpdateNotice`).
 */
export function HeaderTools({
  size,
  sidebarCompact,
  onToggleSidebar,
  onOpenSettings,
  launcher,
}: Props) {
  const prismFound = useEngineStore((s) => s.engine?.found === true)
  const openPrism = useEngineStore((s) => s.openPrism)
  const wikiHome = useChapterStore((s) => s.manifest.wiki.baseUrl)

  return (
    <div className="flex items-center gap-1">
      <SidebarButton
        size={size}
        sidebarCompact={sidebarCompact}
        onToggleSidebar={onToggleSidebar}
        launcher={launcher}
      />
      <div aria-hidden className="bg-line mx-1.5 h-4 w-px" />
      <IconButton
        icon={Settings}
        title="Settings"
        size={size}
        tooltip
        onClick={() => onOpenSettings?.()}
        disabled={!onOpenSettings}
      />
      <IconButton
        icon={Triangle}
        title="Open Prism"
        size={size}
        tooltip
        disabled={!prismFound}
        onClick={() => void openPrism().catch((e) => console.warn('open prism', errMsg(e)))}
      />
      <IconButton
        icon={BookOpen}
        title="Open Wiki"
        size={size}
        tooltip
        onClick={() =>
          void OpenExternal(wikiHome).catch((e) => console.warn('open wiki', errMsg(e)))
        }
      />
      <Suspense fallback={null}>
        <UpdateNotice size={size} />
      </Suspense>
    </div>
  )
}
