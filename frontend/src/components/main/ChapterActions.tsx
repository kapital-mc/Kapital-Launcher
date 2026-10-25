import type { Chapter } from '../../types'
import {
  selectInstalled,
  selectInstancePack,
  selectPackState,
  useEngineStore,
} from '../../stores/useEngineStore'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { selectStatus, useServerStore } from '../../stores/useServerStore'
import { selectGame, useGameStore } from '../../stores/useGameStore'
import { errMsg } from '../../lib/ipc'
import { openLink } from '../../lib/useOpenMap'
import { ActionBar } from './ActionBar'
import type { ActionLayout } from './ActionBar'

const PRISM_SITE = 'https://prismlauncher.org'

interface Props {
  chapter: Chapter
  /** The chapter card's bar, or a compact panel's foot. */
  layout?: ActionLayout
}

/**
 * Play and its state line for one chapter, reading the chapter's own slice of
 * each store: the chapter card's action bar, and the foot of the chapter's
 * panel in the compact launcher.
 */
export function ChapterActions({ chapter, layout = 'bar' }: Props) {
  const engine = useEngineStore((s) => s.engine)
  const launching = useEngineStore((s) => s.launching)
  const release = useEngineStore((s) => s.release)
  const install = useEngineStore((s) => s.install)
  const installPrism = useEngineStore((s) => s.installPrism)
  const installed = useEngineStore(selectInstalled(chapter.id))
  const instancePack = useEngineStore(selectInstancePack(chapter.id))
  const packState = useEngineStore(selectPackState(chapter.id))
  const launch = useEngineStore((s) => s.launch)
  const installing = useEngineStore((s) => s.installing)
  const installedNow = useEngineStore((s) => s.installedNow)
  const installChapter = useEngineStore((s) => s.installChapter)
  const reinstallChapter = useEngineStore((s) => s.reinstallChapter)

  const status = useServerStore(selectStatus(chapter.id))
  const game = useGameStore(selectGame(chapter.id))
  const stopGame = useGameStore((s) => s.stop)
  const checking = useServerStore((s) => s.checking)
  const checkServer = useServerStore((s) => s.check)

  const devPack = useSettingsStore((s) => s.settings.packOverrides?.[chapter.id])
  const serverChoice = useSettingsStore((s) => s.settings.serverChoices?.[chapter.id])
  const joinServer = useSettingsStore((s) => s.settings.joinServers?.includes(chapter.id) ?? false)

  return (
    <ActionBar
      layout={layout}
      chapter={chapter}
      engine={engine}
      status={status}
      installed={installed}
      devPack={devPack}
      serverChoice={serverChoice}
      joinServer={joinServer}
      instancePack={instancePack}
      packState={packState}
      game={game}
      launching={launching === chapter.id}
      installing={installing === chapter.id}
      installedNow={installedNow === chapter.id}
      checking={checking === chapter.id}
      onPlay={() => void launch(chapter.id)}
      // The store has recorded a refusal for the corner's notice.
      onStop={() => void stopGame(chapter.id).catch((e) => console.warn('stop game', errMsg(e)))}
      onInstall={() => void installChapter(chapter.id)}
      onReinstall={() => void reinstallChapter(chapter.id)}
      release={release}
      install={install}
      onGetPrism={() => void installPrism()}
      onOpenPrismSite={() => openLink(PRISM_SITE, 'open prism site')}
      onOpenReleasePage={() => openLink(release?.page || PRISM_SITE, 'open prism release')}
      onCheckServer={() => void checkServer(chapter.id)}
    />
  )
}
