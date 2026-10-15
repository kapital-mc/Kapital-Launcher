import { useEffect } from 'react'
import { SplashCard } from '../components/splash/SplashCard'
import { SplashCover } from '../components/splash/SplashCover'
import { send, useCardStore } from './bridge'

/** Whether this page is a macOS cover window's (`splash.html?view=cover`). */
const isCover = () => new URLSearchParams(window.location.search).get('view') === 'cover'

/**
 * The loading card's whole page (#97): the card for the state Go last pushed,
 * and nothing until the first one. The card scopes the chapter's accent itself;
 * the theme is an attribute on the root, as in the launcher's own window. A
 * macOS cover window loads the same page with `?view=cover` and is pushed the
 * same state, and shows the chapter's backdrop for it (SplashCover) instead.
 */
export function SplashApp() {
  const state = useCardStore((s) => s.state)
  const theme = state?.theme

  useEffect(() => {
    if (theme) document.documentElement.dataset.theme = theme
    else delete document.documentElement.dataset.theme
  }, [theme])

  if (!state) return null
  if (isCover()) return <SplashCover chapter={state.chapter} />
  return (
    <SplashCard
      chapter={state.chapter}
      state={state.game}
      copyLog={state.copyLog}
      error={state.error}
      report={state.report}
      onLeave={() => send('leave')}
      onOpenFolder={() => send('openFolder')}
      onCopyLog={() => send('copyLog')}
      onShowConsole={() => send('showConsole')}
    />
  )
}
