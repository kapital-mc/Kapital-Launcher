import { useState } from 'react'
import type { AppSettings, Chapter } from '../../types'
import { chosenLabel } from '../../lib/manifest'
import { errMsg } from '../../lib/ipc'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { Server } from '../../lib/icons'
import { Segmented } from '../ui/Segmented'
import { SettingsCard, SettingsSection } from '../ui/SettingsLayout'
import { Toggle } from '../ui/Toggle'
import { hasDistantHorizons } from '../../lib/distantHorizons'
import { DistantHorizonsCard } from './DistantHorizonsCard'
import { TailscaleRow } from './TailscaleRow'

interface Props {
  chapter: Chapter
}

/** Which control a refusal is filed under. */
type Field = 'address' | 'join' | 'tailscale'

/**
 * The Server section of a chapter's settings page (issue 163), for a chapter
 * that has a server, installed or not: the address the launcher pings and, with
 * the switch on, joins, and the switch itself. A chapter with more than one
 * address offers them by label (issue 151); the choice is saved as that label
 * and Go checks it against the chapter's own list. A chapter with one has
 * nothing to choose, so its label is shown as text. "Join the server on Play"
 * is off for every chapter until the player turns it on; on, Play reads Join
 * and Prism is started with the chosen address. Both save the moment they are
 * changed, through the settings store as the Settings screen does, and a
 * refusal from Go shows under its control with the old value back. A chapter
 * whose pack has Distant Horizons gets the world data card under it (issue 136).
 *
 * A server with a Tailscale address offers it under Address too, greyed out
 * until the Tailscale switch below Address is on (TailscaleRow), which itself
 * shows only once Tailscale is installed and reaches the server. Turning the
 * switch off moves a chapter that had picked it back to its first address.
 */
export function ChapterServerSection({ chapter }: Props) {
  const settings = useSettingsStore((s) => s.settings)
  const update = useSettingsStore((s) => s.update)
  const [errors, setErrors] = useState<Partial<Record<Field, string>>>({})
  const server = chapter.server
  if (!server) return null

  const save = async (field: Field, patch: Partial<AppSettings>) => {
    try {
      await update(patch)
      setErrors((prev) => ({ ...prev, [field]: undefined }))
    } catch (e) {
      setErrors((prev) => ({ ...prev, [field]: errMsg(e) }))
    }
  }

  const tailscaleOn = settings.tailscaleServers?.includes(chapter.id) ?? false
  const hasTailscale = server.addresses.some((a) => a.tailscale)
  const label = chosenLabel(server, settings.serverChoices?.[chapter.id], tailscaleOn) ?? ''
  const othersTailscale = (settings.tailscaleServers ?? []).filter((id) => id !== chapter.id)
  const joining = settings.joinServers?.includes(chapter.id) ?? false
  const others = (settings.joinServers ?? []).filter((id) => id !== chapter.id)
  return (
    <SettingsSection title="Server" icon={Server}>
      <SettingsCard>
        {server.addresses.length > 1 ? (
          <Segmented
            inline
            label="Address"
            value={label}
            options={server.addresses.map((a) => ({
              value: a.label,
              label: a.label,
              disabled: a.tailscale === true && !tailscaleOn,
            }))}
            error={errors.address}
            onChange={(next) =>
              void save('address', {
                serverChoices: { ...settings.serverChoices, [chapter.id]: next },
              })
            }
          />
        ) : (
          <div className="flex items-center justify-between gap-6 text-sm">
            <span className="text-fg font-medium">Address</span>
            <span className="text-fg-muted">{label}</span>
          </div>
        )}
        {hasTailscale && (
          <TailscaleRow
            chapterId={chapter.id}
            on={tailscaleOn}
            error={errors.tailscale}
            onChange={(on) =>
              void save('tailscale', {
                tailscaleServers: on ? [...othersTailscale, chapter.id] : othersTailscale,
              })
            }
          />
        )}
        <Toggle
          label="Join the server on Play"
          checked={joining}
          error={errors.join}
          onChange={(on) =>
            void save('join', { joinServers: on ? [...others, chapter.id] : others })
          }
        />
      </SettingsCard>
      {hasDistantHorizons(chapter) && <DistantHorizonsCard chapter={chapter} />}
    </SettingsSection>
  )
}
