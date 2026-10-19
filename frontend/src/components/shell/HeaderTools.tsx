import { OpenChapterWiki } from '../../../wailsjs/go/main/App'
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
 * Prism), the open chapter's wiki page, and, only while a newer launcher is
 * out, a button in the accent that brings its popup back.
 */
export function HeaderTools({ size, sidebarCompact, onToggleSidebar, onOpenSettings }: Props) {
  const prismFound = useEngineStore((s) => s.engine?.found === true)
  const openPrism = useEngineStore((s) => s.openPrism)
  const chapterId = useChapterStore((s) => s.selectedId)
  const update = useUpdateStore((s) => (s.update?.available ? s.update : null))
  const reopenUpdate = useUpdateStore((s) => s.reopen)

  return (
    <div className="flex items-center gap-1">
      <IconButton
        icon={sidebarCompact ? PanelLeftOpen : PanelLeftClose}
        title={sidebarCompact ? 'Expand sidebar' : 'Compact sidebar'}
        size={size}
        onClick={() => onToggleSidebar?.()}
        disabled={!onToggleSidebar}
      />
      <div aria-hidden className="bg-line mx-1.5 h-4 w-px" />
      <IconButton
        icon={Settings}
        title="Settings"
        size={size}
        onClick={() => onOpenSettings?.()}
        disabled={!onOpenSettings}
      />
      <IconButton
        icon={Triangle}
        title="Open Prism"
        size={size}
        disabled={!prismFound}
        onClick={() => void openPrism().catch((e) => console.warn('open prism', errMsg(e)))}
      />
      <IconButton
        icon={BookOpen}
        title="Open the wiki"
        size={size}
        onClick={() =>
          void OpenChapterWiki(chapterId).catch((e) => console.warn('open wiki', errMsg(e)))
        }
      />
      {update && (
        <IconButton
          icon={Download}
          title={`Update to ${update.latest}`}
          size={size}
          tone="accent"
          onClick={reopenUpdate}
        />
      )}
    </div>
  )
}
