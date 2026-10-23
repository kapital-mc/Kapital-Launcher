import { Suspense, lazy, useEffect, useRef, useState } from 'react'
import { HeaderBar } from './components/shell/HeaderBar'
import { Toasts } from './components/shell/Toasts'
import { LaunchIntro } from './components/shell/LaunchIntro'
import { LauncherLayout } from './components/shell/LauncherLayout'
import { selectCanRotate, selectChapter, useChapterStore } from './stores/useChapterStore'
import { useEngineStore } from './stores/useEngineStore'
import { useSettingsStore } from './stores/useSettingsStore'
import { useServerStore } from './stores/useServerStore'
import { isActive, useGameStore } from './stores/useGameStore'
import { Environment } from '../wailsjs/runtime/runtime'
import { errMsg, hasWailsBridge, readOr } from './lib/ipc'
import { guessPlatform } from './lib/platform'
import { SLIDE_INTERVAL_MS } from './lib/slides'
import { usePages } from './lib/usePages'
import { heroArtOf } from './lib/heroArt'
import { useCompactLayout } from './lib/compactMode'

// What's new, shown once on the first start after an update, loads in a chunk of
// its own after the first paint, which keeps it out of the bundle budget.
const WhatsNew = lazy(() =>
  import('./components/shell/WhatsNew').then((m) => ({ default: m.WhatsNew })),
)
// So does the popup for a Play Prism ignored over the profile name.
const ProfileNotice = lazy(() =>
  import('./components/shell/ProfileNotice').then((m) => ({ default: m.ProfileNotice })),
)

export default function App() {
  const chapter = useChapterStore(selectChapter)
  const chapters = useChapterStore((s) => s.manifest.chapters)
  const selectedId = useChapterStore((s) => s.selectedId)
  const select = useChapterStore((s) => s.select)
  const loadChapters = useChapterStore((s) => s.load)
  const loadWikiPages = useChapterStore((s) => s.loadWikiPages)
  const loadWikiShots = useChapterStore((s) => s.loadWikiShots)
  const advanceSlide = useChapterStore((s) => s.advance)
  const setSlidesTogether = useChapterStore((s) => s.setTogether)
  const canRotate = useChapterStore(selectCanRotate)
  const loadPanoramas = useChapterStore((s) => s.loadPanoramas)
  const heroArt = useSettingsStore((s) => heroArtOf(s.settings))

  const engine = useEngineStore((s) => s.engine)
  const loadEngine = useEngineStore((s) => s.load)
  const loadInstances = useEngineStore((s) => s.loadInstances)
  const loadRelease = useEngineStore((s) => s.loadRelease)
  const installedNow = useEngineStore((s) => s.installedNow)
  const listenInstall = useEngineStore((s) => s.listenInstall)

  const listenServers = useServerStore((s) => s.listen)
  const checkServer = useServerStore((s) => s.check)

  const listenGame = useGameStore((s) => s.listen)
  const loadGames = useGameStore((s) => s.load)
  const gameActive = useGameStore((s) => Object.values(s.states).some((g) => isActive(g.phase)))
  const gameRunning = useGameStore((s) =>
    Object.values(s.states).some((g) => g.phase === 'running'),
  )

  const theme = useSettingsStore((s) => s.settings.theme)
  const loadSettings = useSettingsStore((s) => s.load)
  const sidebarCompact = useSettingsStore((s) => s.settings.sidebarCompact === true)
  const updateSettings = useSettingsStore((s) => s.update)
  const toggleSidebar = () => {
    updateSettings({ sidebarCompact: !sidebarCompact }).catch((e) =>
      console.warn('save sidebar', errMsg(e)),
    )
  }
  // The compact launcher: chosen from the sidebar button's menu, or held on by a
  // window too small for the full layout (lib/compactMode).
  const launcherCompact = useSettingsStore((s) => s.settings.launcherCompact === true)
  const layout = useCompactLayout(launcherCompact)
  const chooseLayout = (compact: boolean) => {
    updateSettings({ launcherCompact: compact }).catch((e) =>
      console.warn('save layout', errMsg(e)),
    )
  }

  // Which OS draws the window, for the header's controls: guessed from the
  // WebView at once, then Wails' answer (lib/platform).
  const [platform, setPlatform] = useState(() =>
    guessPlatform(navigator.userAgent, hasWailsBridge()),
  )
  useEffect(() => {
    void readOr(Environment, null).then((env) => env && setPlatform(env.platform))
  }, [])

  // The pages over the chapter card: the settings from the gear, a chapter's
  // own from its hero and from the corner's notices (lib/usePages).
  const pages = usePages(selectedId, select)

  // One read per store on mount. These are reads of state Go holds, not
  // events, so an effect is the right tool.
  useEffect(() => {
    void loadChapters()
    void loadEngine()
    void loadSettings()
    void loadWikiPages()
    void loadWikiShots()
  }, [loadChapters, loadEngine, loadSettings, loadWikiPages, loadWikiShots])

  // A run on Prism's default account tells Go the player's name (settings' playerName), which
  // it has by the time the game runs: read the settings again then, so the account tile shows it.
  useEffect(() => {
    if (gameRunning) void loadSettings()
  }, [gameRunning, loadSettings])

  // The open chapter's slide moves on every minute (issue 142): a timer for
  // what is on screen, not a poll of anything Go holds. It runs only while the
  // window is in view, starts over on every switch, and not at all unless the
  // slideshow is what the hero shows, or with nothing to move on to. The
  // compact launcher shows every chapter's slide, and they move on together:
  // a switch there changes no picture and does not start the minute over.
  const slideshow = heroArt === 'slideshow'
  const together = layout.compact
  useEffect(() => setSlidesTogether(together), [together, setSlidesTogether])
  const rotation = together ? '' : selectedId
  useEffect(() => {
    if (!slideshow || !canRotate) return
    let timer: number | undefined
    const stop = () => {
      window.clearInterval(timer)
      timer = undefined
    }
    const start = () => {
      stop()
      if (document.visibilityState === 'visible')
        timer = window.setInterval(() => void advanceSlide(), SLIDE_INTERVAL_MS)
    }
    start()
    document.addEventListener('visibilitychange', start)
    return () => {
      stop()
      document.removeEventListener('visibilitychange', start)
    }
  }, [slideshow, canRotate, rotation, advanceSlide])

  // An instance appears when the user imports one in Prism, which happens in
  // another window. Coming back is the moment to look again; a focus event,
  // not a timer, so nothing polls.
  useEffect(() => {
    const onFocus = () => void loadInstances()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [loadInstances])

  // The panoramas are the resources packs in the instances, which change in
  // other windows (Prism, the game) and when an install finishes. They are read
  // only while the panorama is the hero's picture: when it is chosen, when the
  // window is focused again, and when an install or a game run has ended (a game
  // running leaves them alone). A focus event or one of those, never a timer.
  const panoramaChosen = heroArt === 'panorama'
  useEffect(() => {
    if (!panoramaChosen || gameActive) return
    void loadPanoramas()
    const onFocus = () => void loadPanoramas()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [panoramaChosen, gameActive, installedNow, loadPanoramas])

  // Prism install progress arrives as events too; one listener for the app.
  useEffect(() => listenInstall(), [listenInstall])

  // Prism's latest release matters in two cases: there is no Prism, so the
  // launcher can offer to get it; or the launcher's own copy is the one in
  // use, so it can offer an update (ADR-11). Checked when the engine is
  // known, never on a timer.
  const engineFound = engine?.found
  const engineSource = engine?.source
  useEffect(() => {
    if (engineFound === false || engineSource === 'managed') void loadRelease()
  }, [engineFound, engineSource, loadRelease])

  // Server status arrives as events from Go's ticker (.claude/rules/ipc.md);
  // this is the one listener, for the app's lifetime.
  useEffect(() => listenServers(), [listenServers])

  // A game's phases arrive as events too (#44). The read covers a window that
  // opens after a game began, and runs after the listener so no phase is lost
  // between them.
  useEffect(() => {
    const off = listenGame()
    void loadGames()
    return off
  }, [listenGame, loadGames])

  // Opening a chapter with a server asks for a fresh ping rather than waiting
  // for the next tick, so the line is current when it is looked at.
  useEffect(() => {
    if (chapter?.server) void checkServer(chapter.id)
  }, [chapter?.id, chapter?.server, checkServer])

  // The accent and the theme are attributes on the root: tokens.css maps
  // data-chapter to --accent and data-theme to the palette.
  useEffect(() => {
    document.documentElement.dataset.chapter = selectedId
  }, [selectedId])
  useEffect(() => {
    if (theme === 'system') delete document.documentElement.dataset.theme
    else document.documentElement.dataset.theme = theme
  }, [theme])

  if (!chapter) return null

  return (
    <div className="bg-canvas flex h-full flex-col">
      <ChapterSelectionSync />
      <HeaderBar
        platform={platform}
        onOpenSettings={pages.toggleSettings}
        sidebarCompact={sidebarCompact}
        onToggleSidebar={toggleSidebar}
        launcher={{ compact: layout.compact, held: layout.held, choose: chooseLayout }}
      />
      <LauncherLayout
        chapter={chapter}
        chapters={chapters}
        select={select}
        pages={pages}
        compact={layout.compact}
      />
      <Toasts onOpenReport={pages.openReport} />
      <Suspense fallback={null}>
        <WhatsNew />
        <ProfileNotice onOpenAccount={pages.openAccount} />
      </Suspense>
      <LaunchIntro />
    </div>
  )
}

/**
 * Keeps the open chapter and the saved `lastChapter` in step, in both
 * directions, so the app reopens where it was left. Kept out of the nav so
 * ChapterNav stays a pure view over the chapter store; the settings write is
 * an app-level concern.
 *
 * One component owns both directions on purpose. Split across two effects
 * (restore in App, save here) they each saw the other's stale value and
 * swapped forever, and every save spawned Prism (#6).
 */
function ChapterSelectionSync() {
  const selectedId = useChapterStore((s) => s.selectedId)
  const select = useChapterStore((s) => s.select)
  const loaded = useSettingsStore((s) => s.loaded)
  const last = useSettingsStore((s) => s.settings.lastChapter)
  const update = useSettingsStore((s) => s.update)
  // Whether the saved chapter has been applied yet. A ref, not state: flipping
  // it must not cause a render of its own.
  const restored = useRef(false)

  useEffect(() => {
    // Until settings arrive, `last` is the default and says nothing.
    if (!loaded) return
    // First pass: reopen the saved chapter and save nothing. If it no longer
    // exists, select is a no-op and the next pass saves the default once.
    if (!restored.current) {
      restored.current = true
      if (last) select(last)
      return
    }
    if (selectedId && selectedId !== last) {
      update({ lastChapter: selectedId }).catch((e) => console.warn('save chapter', errMsg(e)))
    }
  }, [loaded, selectedId, last, select, update])
  return null
}
