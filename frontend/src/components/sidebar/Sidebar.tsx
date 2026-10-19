import { Suspense, lazy } from 'react'
import { ChapterNav } from './ChapterNav'
import { EngineCard } from './EngineCard'

// Only a download the player starts fills the section, from a page or popup
// that loads later still, so the section and the stores it listens through
// are fetched after the first paint rather than in it.
const Downloads = lazy(() => import('./Downloads').then((m) => ({ default: m.Downloads })))

interface Props {
  /** Opens or closes the account page (issue 192). */
  onOpenAccount?: () => void
  /** Whether the account page is the one open, for the tile's pressed look. */
  accountOpen?: boolean
}

export function Sidebar({ onOpenAccount, accountOpen = false }: Props) {
  return (
    <aside className="bg-sunken border-line flex w-(--layout-sidebar) shrink-0 flex-col gap-7 border-r px-4 pt-5.5 pb-4">
      <ChapterNav />
      <div className="grow" />
      <Suspense fallback={null}>
        <Downloads />
      </Suspense>
      <EngineCard onOpenAccount={onOpenAccount} accountOpen={accountOpen} />
    </aside>
  )
}
