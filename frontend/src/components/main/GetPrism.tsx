import type { PrismInstallProgress, PrismRelease } from '../../types'
import { installLine, megabytes } from '../../lib/prismInstall'
import { TextLink } from '../ui/TextLink'
import { installStage } from '../../lib/installStage'
import { InstallPopup } from '../shell/InstallPopup'

interface Props {
  release: PrismRelease
  /** The install this popup started, from prism:install; null until Download. */
  install: PrismInstallProgress | null
  onConfirm: () => void
  onCancel: () => void
  onOpenReleasePage: () => void
  onOpenPrismSite: () => void
}

/**
 * Getting Prism (ADR-11), as a popup over the window: what is downloaded, how
 * big it is, where from and under which licence, before anything is fetched.
 * Nothing happens until the player confirms. It floats over the launcher
 * rather than opening in the action bar, so the chapter card keeps its layout
 * (on a short window the inline card pushed the hero's title off the top), and
 * it follows the install to its end. Escape, Not now and a click on the cover
 * all close it.
 */
export function GetPrism({
  release,
  install,
  onConfirm,
  onCancel,
  onOpenReleasePage,
  onOpenPrismSite,
}: Props) {
  return (
    <InstallPopup
      title="Get Prism Launcher"
      facts={`Version ${release.version} · ${megabytes(release.size)} · GPL-3.0`}
      stage={installStage(install, (p) => installLine(p, 'Getting')[1])}
      intro={
        <p className="m-0">
          Kapital Launcher plays through Prism Launcher. It is downloaded from Prism&apos;s releases
          on GitHub, apart from any Prism you install yourself. Prism asks you to sign in with
          Microsoft on your first Play; Kapital Launcher never sees your account.
        </p>
      }
      done={
        <p className="m-0">
          Prism is ready. The first time you play, Prism asks you to sign in with Microsoft.
        </p>
      }
      links={
        <>
          <TextLink size="xs" onClick={onOpenReleasePage}>
            What&apos;s in {release.version}
          </TextLink>
          <TextLink size="xs" onClick={onOpenPrismSite}>
            Prism&apos;s website
          </TextLink>
        </>
      }
      onConfirm={onConfirm}
      onClose={onCancel}
    />
  )
}
