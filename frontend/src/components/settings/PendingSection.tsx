import type { LucideIcon } from '../../lib/icons'
import { SettingsCard, SettingsSection } from '../ui/SettingsLayout'

/**
 * What a chapter's Mods and Storage hold in their place until their data is read and the page
 * has slid in (lib/pageSettled): the heading, and an empty card of one token height
 * (`layout.pending`). The page keeps its length, and the card grows from this to its rows
 * instead of the section appearing. The card is a `SettingsCard` in the place the section's own
 * is, so it is the same box that follows its content smoothly when the rows replace this row.
 */
export function PendingRow() {
  return <div aria-hidden className="h-(--layout-pending)" />
}

/** The Storage section's bar, empty, in the bar's place. */
export function PendingBar() {
  return <div aria-hidden className="bg-raised-2 rounded-pill h-2" />
}

interface Props {
  title: string
  icon: LucideIcon
  /** Storage's bar over its card. */
  bar?: boolean
}

/**
 * The section's pending look on its own, for the page to show while the section's code is on its
 * way: the same markup the section draws until its data and the page are ready.
 */
export function PendingSection({ title, icon, bar = false }: Props) {
  return (
    <SettingsSection title={title} icon={icon}>
      {bar && <PendingBar />}
      <SettingsCard>
        <PendingRow />
      </SettingsCard>
    </SettingsSection>
  )
}
