import { useEngineStore } from '../../stores/useEngineStore'
import { installLine } from '../../lib/prismInstall'
import { accountName, profileInitials } from '../../lib/profileName'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { IconButton } from '../ui/IconButton'
import { ErrorLine } from '../ui/Notes'
import { TextLink } from '../ui/TextLink'
import { ChevronRight, RefreshCw } from '../../lib/icons'
import { Icon } from '../ui/Icon'
import { Fold } from '../ui/Fold'

interface Props {
  /** Opens or closes the account page (issue 192); without it the account half is plain text. */
  onOpenAccount?: () => void
  /** The account page is open: the card takes the accent, as a chosen chapter does. */
  accountOpen?: boolean
  /**
   * The sidebar is compact: the card folds to a square around the initials, with even sides,
   * and the initials stay a button for the account page.
   */
  compact?: boolean
}

/**
 * The account and engine card at the foot of the sidebar. The account line
 * shows the profile *name* from settings, or the name the game reported for
 * Prism's default account, and nothing more: the Microsoft
 * account itself lives inside Prism and this app never reads it. The account
 * half is a button that opens the account page (issue 192); Re-detect stays a
 * button of its own beside it.
 *
 * Compact, everything but the initials folds away: the name and Re-detect
 * sideways, the engine lines downwards, and the padding grows by the 2px that
 * makes the card as wide as a chapter row and exactly as tall.
 */
export function EngineCard({ onOpenAccount, accountOpen = false, compact = false }: Props) {
  const engine = useEngineStore((s) => s.engine)
  const refresh = useEngineStore((s) => s.refresh)
  const release = useEngineStore((s) => s.release)
  const install = useEngineStore((s) => s.install)
  const installPrism = useEngineStore((s) => s.installPrism)
  const profile = useSettingsStore((s) =>
    accountName(s.settings.profileName, s.settings.playerName),
  )
  const offline = useSettingsStore((s) => s.settings.offline)

  const name = profile || 'Default profile'
  const initials = profileInitials(name)
  const accountLine = offline ? 'Offline' : 'Microsoft account · via Prism'

  // An update runs while Prism is found; the first install is the action bar's.
  const updateLine =
    engine?.found && install && ['downloading', 'unpacking', 'verifying'].includes(install.phase)
      ? installLine(install, 'Updating')[0].replace(/^◐ /, '')
      : null
  const updateFailed = engine?.found && install?.phase === 'failed' ? install.error : null
  const how = engine?.source === 'managed' ? 'managed' : 'linked'
  const engineLine =
    engine === null
      ? 'not checked'
      : !engine.found
        ? 'not found'
        : (updateLine ?? `Prism ${engine.version || ''}${engine.version ? ' · ' : ''}${how}`)
  // GetPrismRelease only offers an update for the managed Prism, in use.
  const offer = engine?.found && release?.updateAvailable && !updateLine ? release : null

  return (
    <div
      className={`duration-fast ease-standard rounded-lg border transition-colors ${
        accountOpen ? 'bg-accent-wash border-accent-edge' : 'bg-raised border-line'
      }`}
    >
      {/* The padding moves with the folds, on their curve: faster, it would shrink the tile for a
          moment as it widens again, before the folds have opened enough to outgrow it. */}
      <div
        className={`duration-slow ease-glide flex flex-col transition-[padding] motion-reduce:transition-none ${
          compact ? 'p-3.5' : 'p-3'
        }`}
      >
        <div className="flex items-center">
          <button
            type="button"
            aria-label={`Account, ${name}`}
            aria-pressed={accountOpen}
            disabled={!onOpenAccount}
            onClick={() => onOpenAccount?.()}
            className="hover:bg-hover duration-fast ease-standard -m-1.5 flex min-w-0 grow cursor-pointer items-center rounded-md p-1.5 text-left transition-colors disabled:cursor-default disabled:hover:bg-transparent"
          >
            <span className="bg-hover flex size-8 shrink-0 items-center justify-center rounded-sm font-mono text-xs">
              {initials || 'KK'}
            </span>
            <Fold open={!compact} axis="x" className="min-w-0 grow">
              <span className="flex items-center gap-2.5 pl-2.5">
                <span className="flex min-w-0 grow flex-col gap-0.5">
                  <span className="truncate text-sm font-semibold">{name}</span>
                  <span className="text-fg-faint text-2xs truncate" title={accountLine}>
                    {accountLine}
                  </span>
                </span>
                {onOpenAccount && <Icon icon={ChevronRight} size="sm" className="text-fg-faint" />}
              </span>
            </Fold>
          </button>
          <Fold open={!compact} axis="x" className="shrink-0">
            <span className="flex pl-1.5">
              <IconButton
                icon={RefreshCw}
                title="Re-detect Prism"
                size="sm"
                onClick={() => void refresh()}
              />
            </span>
          </Fold>
        </div>
        <Fold open={!compact} axis="y">
          <div className="flex flex-col gap-2.5 pt-2.5">
            <div className="bg-line h-px" />
            <div className="text-fg-faint text-2xs flex justify-between gap-2 whitespace-nowrap">
              <span>engine</span>
              <span
                className={`min-w-0 truncate font-mono ${
                  engine !== null && !engine.found ? 'text-danger' : 'text-fg-muted'
                }`}
                title={engineLine}
              >
                {engineLine}
              </span>
            </div>
            {offer && (
              <TextLink size="xs" className="self-start" onClick={() => void installPrism()}>
                Update Prism to {offer.version}
              </TextLink>
            )}
            {updateFailed && <ErrorLine>Update failed: {updateFailed}</ErrorLine>}
          </div>
        </Fold>
      </div>
    </div>
  )
}
