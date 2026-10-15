import { chapterArt } from '../../lib/art'
import type { CardState } from '../../splash/bridge'

interface Props {
  chapter: CardState['chapter']
}

/**
 * What the macOS cover shows behind the loading card (ADR-0012's 2026-10-15
 * amendment): black, the game's own colour, with the chapter's art seen only
 * through a few soft spots away from the card. The art has the card's
 * treatment, blurred and tinted toward the chapter's accent, more blurred and
 * much fainter here, so the screen reads as black with a hint of the chapter.
 * The spots fade in slowly and then breathe (`cover-spots` in style.css). It is
 * a page of its own in each cover window and takes no input.
 */
export function SplashCover({ chapter }: Props) {
  const art = chapterArt(chapter.id)
  return (
    <div
      data-chapter={chapter.id}
      aria-hidden
      className="bg-cover-ground pointer-events-none relative isolate h-full w-full overflow-hidden"
    >
      {art && (
        // The art's strength is on this layer and the motion on the one inside,
        // so the animated opacity never replaces the token.
        <div className="absolute inset-0 opacity-(--cover-art)">
          <div className="cover-spots absolute inset-0">
            <img
              src={art}
              alt=""
              className="size-full scale-110 object-cover blur-(--cover-blur)"
              draggable={false}
            />
            <div className="bg-accent absolute inset-0 opacity-(--effect-tile-tint) mix-blend-color" />
          </div>
        </div>
      )}
    </div>
  )
}
