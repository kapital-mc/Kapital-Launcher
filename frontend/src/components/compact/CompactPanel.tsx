import type { Chapter } from '../../types'
import { selectSlide, useChapterStore } from '../../stores/useChapterStore'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { chapterArt, chapterTitleArt } from '../../lib/art'
import { heroArtOf } from '../../lib/heroArt'
import { useLoadedImage } from '../../lib/useLoadedImage'
import { useOpenMap } from '../../lib/useOpenMap'
import { Drift } from '../ui/Drift'
import { ChapterActions } from '../main/ChapterActions'
import { HeroTools } from '../main/HeroTools'

interface Props {
  chapter: Chapter
  /** Whether this is the open chapter: the wide panel, with its tools and Play. */
  active: boolean
  /** Whether it leans toward the open panel, the pointer having rested on it (lib/usePanelLean). */
  leaning: boolean
  onPointerEnter: () => void
  onPointerLeave: () => void
  onSelect: () => void
  onOpenSettings: (chapterId: string) => void
  onOpenLogs: (chapterId: string) => void
  onOpenMap: (chapterId: string) => void
}

/**
 * One chapter in the compact launcher: its picture, the hero's, under a wash
 * rising from the foot; its number and kind, its title, and for the open
 * chapter its tools and Play. Another chapter's panel is one button that opens
 * it. Every piece the morph moves is marked `data-morph` (lib/compactMorph).
 */
export function CompactPanel({
  chapter,
  active,
  leaning,
  onPointerEnter,
  onPointerLeave,
  onSelect,
  onOpenSettings,
  onOpenLogs,
  onOpenMap,
}: Props) {
  const slide = useChapterStore(selectSlide(chapter.id))
  const heroArt = useSettingsStore((s) => heroArtOf(s.settings))
  const own = chapterArt(chapter.id)
  const art = useLoadedImage((heroArt === 'slideshow' ? slide?.art : undefined) ?? own, own)
  const title = chapterTitleArt(chapter.id)
  const openMap = useOpenMap(chapter, onOpenMap)

  return (
    <div
      role="group"
      aria-label={chapter.name}
      data-morph="panel"
      data-active={active || undefined}
      data-lean={leaning || undefined}
      onPointerEnter={onPointerEnter}
      onPointerLeave={onPointerLeave}
      data-chapter={chapter.id}
      className="compact-panel bg-sunken relative min-w-0 overflow-hidden rounded-md"
    >
      {art ? (
        // A new slide drifts in as it does in the hero (issue 142).
        <Drift id={art} className="absolute inset-0" layer="absolute inset-0">
          <img
            data-morph="panel-art"
            src={art}
            alt=""
            draggable={false}
            className="compact-art size-full object-cover object-[center_40%]"
          />
        </Drift>
      ) : (
        <div data-morph="panel-art" aria-hidden className="art-pending absolute inset-0" />
      )}
      <div data-morph="panel-shade" aria-hidden className="scrim-foot absolute inset-0" />
      {!active && (
        <button
          type="button"
          aria-label={`Open ${chapter.name}`}
          onClick={onSelect}
          className="absolute inset-0 z-10 rounded-md"
        />
      )}
      <div inert={!active} className="contents">
        <HeroTools
          name={chapter.name}
          onOpenSettings={() => onOpenSettings(chapter.id)}
          onOpenLogs={() => onOpenLogs(chapter.id)}
          onOpenMap={openMap}
          className="compact-tools absolute top-6 right-6"
          morph="panel-tools"
        />
      </div>
      <div data-morph="panel-body" className="absolute inset-x-6 bottom-6 flex flex-col gap-3.5">
        <div
          data-morph="panel-label"
          className="text-fg-muted panel-short:invisible panel-short:absolute flex items-baseline gap-2.5 overflow-hidden text-sm whitespace-nowrap"
        >
          <span className="text-accent font-mono">{chapter.number}</span>
          <span className="truncate">{chapter.kind}</span>
        </div>
        <h2 className="font-display text-display m-0 font-semibold">
          {title ? (
            <img
              data-morph="panel-title"
              src={title}
              alt={chapter.name}
              decoding="sync"
              draggable={false}
              className="compact-title block h-auto"
            />
          ) : (
            <span data-morph="panel-title" className="block truncate">
              {chapter.name}
            </span>
          )}
        </h2>
        {/* Play's row: open under the open chapter's title, closed under the others.
            It is not clipped while it opens: Play fades in once there is room. */}
        <div inert={!active} className="compact-foot -mt-3.5">
          <div className="min-h-0">
            <div className="compact-foot-row pt-3.5">
              <ChapterActions chapter={chapter} layout="panel" />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
