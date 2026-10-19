import { OpenExternal } from '../../../wailsjs/go/main/App'
import { errMsg } from '../../lib/ipc'
import {
  BookOpen,
  Download,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  Triangle,
} from '../../lib/icons'
import { useChapterStore } from '../../stores/useChapterStore'
import { useEngineStore } from '../../stores/useEngineStore'
import { useUpdateStore } from '../../stores/useUpdateStore'
import { IconButton } from '../ui/IconButton'

interface Props {
  /** The header's button size: `xs` in macOS's shorter bar. */
  size: 'xs' | 'sm'
  sidebarCompact: boolean
  onToggleSidebar?: () => void
  onOpenSettings?: () => void
}

/**
 * The tools at the left of the header bar: the sidebar's button, then the
 * settings, Prism itself (for accounts and mods, greyed out while there is no
 * Prism), the wiki's home page, and, only while a newer launcher is
 * out, a button in the accent that brings its popup back.
 */
export function HeaderTools({ size, sidebarCompact, onToggleSidebar, onOpenSettings }: Props) {
  const prismFound = useEngineStore((s) => s.engine?.found === true)
  const openPrism = useEngineStore((s) => s.openPrism)
  const wikiHome = useChapterStore((s) => s.manifest.wiki.baseUrl)
  const updateOut = useUpdateStore((s) => s.update?.available === true)
  const reopenUpdate = useUpdateStore((s) => s.reopen)

  return (
    <div className="flex items-center gap-1">
      <IconButton
        icon={sidebarCompact ? PanelLeftOpen : PanelLeftClose}
        title={sidebarCompact ? 'Expand sidebar' : 'Compact sidebar'}
        size={size}
        tooltip
        onClick={() => onToggleSidebar?.()}
        disabled={!onToggleSidebar}
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
      {updateOut && (
        <IconButton
          icon={Download}
          title="Update Launcher"
          size={size}
          tooltip
          tone="accent"
          onClick={reopenUpdate}
        />
      )}
    </div>
  )
}
