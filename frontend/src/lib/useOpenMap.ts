import type { Chapter } from '../types'
import { useSettingsStore } from '../stores/useSettingsStore'
import { OpenExternal } from '../../wailsjs/go/main/App'
import { errMsg } from './ipc'

/** Hands an address to the system browser through Go's checks; a refusal is logged. */
export const openLink = (url: string, what: string) =>
  OpenExternal(url).catch((e) => console.warn(what, errMsg(e)))

/**
 * Opens a chapter's map where settings say (issue 161). In the browser it goes
 * through the same OpenExternal checks as every link; a chapter with no map
 * has nothing to hand the browser, so it opens the page, which says so.
 */
export function useOpenMap(chapter: Chapter, onOpenMap: (chapterId: string) => void) {
  const mapIn = useSettingsStore((s) => s.settings.mapIn)
  return () =>
    mapIn === 'browser' && chapter.map ? openLink(chapter.map, 'open map') : onOpenMap(chapter.id)
}
