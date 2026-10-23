import type { Chapter } from '../../types'
import { selectPanorama, selectSlide, useChapterStore } from '../../stores/useChapterStore'
import {
  selectInstalled,
  selectInstanceSize,
  selectPackState,
  useEngineStore,
} from '../../stores/useEngineStore'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { OpenChapterWiki, OpenWikiPage } from '../../../wailsjs/go/main/App'
import { errMsg } from '../../lib/ipc'
import { packVersion } from '../../lib/packState'
import { heroArtOf } from '../../lib/heroArt'
import { useOpenMap } from '../../lib/useOpenMap'
import { Hero } from './Hero'
import { ChapterActions } from './ChapterActions'
import { Panels } from './Panels'

/**
 * One chapter's view: the hero, the action bar and the panels, reading the
 * chapter's own slice of each store. It takes the chapter rather than the
 * selection so the card that is leaving can keep showing the chapter it
 * showed while the new one slides in (#59).
 */
interface Props {
  chapter: Chapter
  onOpenSettings: (chapterId: string) => void
  onOpenLogs: (chapterId: string) => void
  onOpenMap: (chapterId: string) => void
}

export function ChapterView({ chapter, onOpenSettings, onOpenLogs, onOpenMap }: Props) {
  const slide = useChapterStore(selectSlide(chapter.id))
  const wikiPick = slide?.page
  // Only the slideshow shows the slide's picture, and only the panorama shows the
  // cube; the other choices keep the chapter's own picture, and a chapter with no
  // panorama read yet (or none at all) falls back to it. The post still follows the slide.
  const heroArt = useSettingsStore((s) => heroArtOf(s.settings))
  const panorama = useChapterStore(selectPanorama(chapter.id))

  const installed = useEngineStore(selectInstalled(chapter.id))
  const instanceSize = useEngineStore(selectInstanceSize(chapter.id))
  const packState = useEngineStore(selectPackState(chapter.id))
  const changelog = useEngineStore((s) => s.changelogs[chapter.id])

  // The page the panel shows is the one that opens: the pick, or the
  // manifest's teaser when there is none.
  const openWiki = () =>
    (wikiPick ? OpenWikiPage(wikiPick.url) : OpenChapterWiki(chapter.id)).catch((e) =>
      console.warn('open wiki', errMsg(e)),
    )
  const openMap = useOpenMap(chapter, onOpenMap)

  return (
    <>
      <Hero
        chapter={chapter}
        art={heroArt === 'slideshow' ? slide?.art : undefined}
        panorama={heroArt === 'panorama' ? panorama : undefined}
        onOpenSettings={() => onOpenSettings(chapter.id)}
        onOpenLogs={() => onOpenLogs(chapter.id)}
        onOpenMap={openMap}
        version={packVersion(packState, chapter.pack.version)}
      />
      <ChapterActions chapter={chapter} />
      <Panels
        chapter={chapter}
        installed={installed}
        sizeBytes={instanceSize}
        packState={packState}
        changelog={changelog}
        wikiPage={wikiPick}
        onOpenWiki={openWiki}
      />
    </>
  )
}
