import { useEffect, useState } from 'react'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { ChapterNav } from './ChapterNav'
import { EngineCard } from './EngineCard'

interface Props {
  /** Opens or closes the account page (issue 192). */
  onOpenAccount?: () => void
  /** Whether the account page is the one open, for the tile's pressed look. */
  accountOpen?: boolean
}

/**
 * The chapters and the account tile. Compact (the header's sidebar button,
 * saved as `sidebarCompact`) it narrows to the chapter icons and a square
 * account tile: only the right edge moves, the icons stay where they are, and
 * the text fades out before it could be seen cut and back in once there is
 * room for it. The heading folds away and the padding meets the chapter card's,
 * so the first row and the account tile line up with the card's top and bottom.
 *
 * The saved width arrives with the settings, after the first paint, and is
 * taken without a transition; only a change after that animates.
 */
export function Sidebar({ onOpenAccount, accountOpen = false }: Props) {
  const compact = useSettingsStore((s) => s.settings.sidebarCompact === true)
  const loaded = useSettingsStore((s) => s.loaded)
  const [settled, setSettled] = useState(false)

  useEffect(() => {
    if (!loaded) return
    // Two frames: the loaded width has to be painted once before a change can be a transition.
    let second = 0
    const first = requestAnimationFrame(() => {
      second = requestAnimationFrame(() => setSettled(true))
    })
    return () => {
      cancelAnimationFrame(first)
      cancelAnimationFrame(second)
    }
  }, [loaded])

  return (
    <aside
      className={`bg-sunken border-line flex shrink-0 flex-col gap-7 overflow-x-clip border-r px-4 ${
        compact ? 'w-(--layout-sidebar-compact) pt-5 pb-5' : 'w-(--layout-sidebar) pt-5.5 pb-4'
      } ${
        settled
          ? 'duration-slow ease-glide transition-[width,padding] motion-reduce:transition-none'
          : '[&_*]:transition-none'
      }`}
    >
      <ChapterNav compact={compact} />
      <div className="grow" />
      <EngineCard onOpenAccount={onOpenAccount} accountOpen={accountOpen} compact={compact} />
    </aside>
  )
}
