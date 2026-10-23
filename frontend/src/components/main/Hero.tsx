import { Suspense, lazy } from 'react'
import type { Chapter } from '../../types'
import { chapterArt, chapterTitleArt } from '../../lib/art'
import { isPlaceholder, knownFacts } from '../../lib/manifest'
import { panoramaStart, usePanoramaReady } from '../../lib/panorama'
import { Drift } from '../ui/Drift'
import { HeroTools } from './HeroTools'
import { Spinner } from '../ui/Spinner'
import { Pill } from '../ui/Pill'
import { useLoadedImage } from '../../lib/useLoadedImage'

// The turning cube is drawn only when the panorama is the chosen picture, so
// its code loads then and stays out of the launcher's first paint and its bundle
// budget (scripts/check-bundle-size.mjs).
const PanoramaCube = lazy(() => import('./PanoramaCube').then((m) => ({ default: m.PanoramaCube })))

interface Props {
  chapter: Chapter
  /** The slide's picture (issue 142); undefined shows the bundled art. */
  art?: string
  /**
   * The chapter's title-screen panorama (issue 195), six faces in the game's order, drawn as a
   * turning cube over the picture; undefined, as before it has loaded or where the chapter has
   * none, leaves the picture alone.
   */
  panorama?: readonly string[]
  /** Opens the chapter's own settings: memory and JVM preset (#36). */
  onOpenSettings: () => void
  /** Opens the chapter's logs and crash reports (issue 155). */
  onOpenLogs: () => void
  /** Opens the chapter's map, in the launcher or the browser as settings say (issue 161). */
  onOpenMap: () => void
  /** The pack's version, as the pack line names it (issue 162); null when nobody knows it yet. */
  version: string | null
}

/**
 * The chapter's picture, name and blurb. Art is the slide's picture from the
 * wiki, which drifts to the next one in place (issue 142), else the bundled
 * screenshot, else the accent grid. With a panorama (issue 195) a turning cube
 * covers the picture once its faces have loaded, under the same scrim. The hero takes whatever height the card has left above the
 * action bar and the panels, up to the layout token, so the card fits the
 * window without scrolling (#68).
 */
export function Hero({
  chapter,
  art: slideArt,
  panorama,
  onOpenSettings,
  onOpenLogs,
  onOpenMap,
  version,
}: Props) {
  // The slide's picture once it has loaded, the chapter's own until then: the
  // drift to a new picture starts only when that picture can be drawn.
  const art = useLoadedImage(slideArt ?? chapterArt(chapter.id), chapterArt(chapter.id))
  const title = chapterTitleArt(chapter.id)
  // The faces are decoded in the background first (lib/panorama), so a chapter switch never
  // waits on megabytes of PNG; the picture under it stays, with a word that the cube is coming.
  const panoramaReady = usePanoramaReady(panorama?.length === 6 ? panorama : undefined)
  // A fact nobody has settled has no chip: a placeholder in brackets is for the
  // author, not the player.
  const spec = knownFacts(chapter.pack.loader, chapter.pack.minecraft)
  const mods = knownFacts(chapter.pack.mods)

  // Its own layer, so the card's rounded corner clips the picture and the scrim
  // over it once, together: clipped one by one, the picture shows past the
  // scrim along the corner's curve as a light line.
  return (
    <section
      data-morph="hero"
      className="border-line relative flex max-h-(--layout-hero) min-h-0 grow flex-col justify-end overflow-hidden border-b px-7 py-11 will-change-transform"
    >
      {art ? (
        <>
          <Drift id={art} className="absolute inset-0" layer="absolute inset-0">
            <img
              src={art}
              alt=""
              className="size-full object-cover object-[center_40%]"
              draggable={false}
            />
          </Drift>
          {panorama && panorama.length === 6 && panoramaReady && (
            <Suspense fallback={null}>
              <PanoramaCube key={panorama[0]} faces={panorama} start={panoramaStart(chapter.id)} />
            </Suspense>
          )}
          <div data-morph="hero-scrim" className="scrim-hero absolute inset-0" aria-hidden />
          {panorama && panorama.length === 6 && !panoramaReady && (
            <span
              role="status"
              className="text-fg-muted text-2xs absolute right-7 bottom-4 flex items-center gap-2"
            >
              <Spinner />
              Loading panorama…
            </span>
          )}
        </>
      ) : (
        <>
          <div className="art-pending absolute inset-0" aria-hidden />
          <div className="text-accent text-2xs tracking-label absolute top-6 left-7 font-mono">
            [ {chapter.name.toUpperCase()} SCREENSHOT ]
          </div>
        </>
      )}

      <HeroTools
        name={chapter.name}
        onOpenSettings={onOpenSettings}
        onOpenLogs={onOpenLogs}
        onOpenMap={onOpenMap}
        className="absolute top-7 right-7"
        morph="hero-tools"
      />

      <div className="relative flex max-w-140 flex-col gap-3.5">
        <div data-morph="hero-label" className="text-fg-muted flex items-baseline gap-2.5 text-sm">
          <span className="text-accent font-mono">{chapter.number}</span>
          <span>{chapter.kind}</span>
        </div>
        {/* The heading is the artwork when there is one, at one height for
            every chapter (#77); its alt keeps the name for readers. */}
        {/* The morph measures the title itself, not the heading's full width. */}
        <h1 className="font-display text-display m-0 font-semibold">
          {title ? (
            <img
              data-morph="hero-title"
              src={title}
              alt={chapter.name}
              // Decoded at startup (preloadImages): drawn with the card, not after it.
              decoding="sync"
              className="block h-(--layout-title) w-auto max-w-full"
              draggable={false}
            />
          ) : (
            <span data-morph="hero-title">{chapter.name}</span>
          )}
        </h1>
        {/* The words that leave first when the launcher goes compact (lib/compactMorph). */}
        <div data-morph="hero-blurb" className="flex flex-col gap-3.5">
          <p className="text-fg-soft m-0 text-lg leading-normal">{chapter.blurb}</p>
          <div className="flex flex-wrap gap-2">
            {spec && <Pill>{spec}</Pill>}
            {mods && <Pill>{mods} mods</Pill>}
            {version && !isPlaceholder(version) && <Pill>Version {version}</Pill>}
          </div>
        </div>
      </div>
    </section>
  )
}
