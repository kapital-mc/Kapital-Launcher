# ADR-0015: The compact launcher

**Status:** accepted, 2026-10-23. Built.

## Context

The download site's redesign shows the launcher as a window of three chapter
panels side by side, the open one wider (mockup J, site PR #299). The author
asked (2026-10-22) for the launcher itself to collapse into that layout, from
two places: a right-click on the header's sidebar button, and shrinking the
window past a size. The smallest window allowed until then (1024 × 640) cut
the hero's label and title off at the top.

The motion the author described: with a chapter open, its picture becomes the
picture of its panel, cropped down at the sides as the other panels come in
beside it; the sidebar leaves to the left; the panels cover the pack, the
changelog and the wiki. Play and the server line must not travel with the
title: the title rises into place while they fade and slide in beneath it,
and the way out is the same in reverse. It was settled in a prototype over
three rounds (artifact "Compact launcher motion", v3).

Wails v2's `WindowSetSize` has no animated form (wails.io, Runtime, Window,
read 2026-10-22), so a menu choice that resized the window would jump.

## Decision

- **Two layouts, one setting.** `launcherCompact` in the settings is the
  choice from the menu. The window shows the compact launcher when it is
  chosen, or when the window is smaller than the size the full layout fits
  (`layout.compact` in the tokens: 880 × 700, measured where the hero's label
  reaches its top padding). Below that size compact is held on and the menu's
  Full launcher is greyed out; it lets go once the window is the hysteresis
  (40 px) past both sides. Shrinking takes compact at once, mid-drag, so the
  full layout is never seen cut; growing waits for the drag to pause.
- **The window keeps its size.** Choosing compact changes the layout, never
  the window. The window minimum drops to 720 × 240. A panel shorter than
  `layout.compact.short` drops its number and kind and takes a smaller title,
  so Play and the title still fit.
- **A left-click in compact goes back to full.** The sidebar button's usual
  click (narrowing the sidebar) has nothing to act on there.
- **Both layouts are laid out; actors carry one into the other.** The full
  layout is always in the page, hidden and inert while compact shows; the
  compact one is in the page while it shows or a morph runs, and its code loads
  on demand. The morph (`lib/compactMorph`, `lib/compactMorphRun`) measures
  both, lays copies of the moving pieces over them and animates those with the
  Web Animations API: the picture's frame as a clip-path from the hero's
  rectangle to the panel's while the picture itself scales at its own aspect;
  the other panels from the window's edge on their side; the title and label to
  the panel's foot with the picture, then up into place; Play and the server
  line fading up after. The way out is the same keyframes reversed with each
  curve mirrored. It waits for every picture to decode before the first frame,
  retargets if the window is resized mid-way, and with reduced motion or in a
  hidden window the layouts simply swap.
- **One timing for the travel.** 900 ms; the picture, the panels and the title
  share one even curve (`ease-travel`) to 62% of it; the rise starts from rest
  (`ease-settle`). These are motion tokens.
- **Pages open over whichever layout shows.** The settings, the account and a
  chapter's pages slide over the panels as they do over the card.
- **The panels' pictures change together.** Every panel shows its chapter's
  slide, so opening another chapter in compact changes no picture; once a
  minute they all move on at once (the author, 2026-10-23).
- **A switch between panels is soft.** Widths, the title and Play's row move
  over `duration.switch` (600 ms) on `ease-travel`. A pointer resting on
  another chapter's panel for `duration.lean` (700 ms) leans it toward the
  open one by `effect.panel.lean`, taking the room from the open panel alone;
  a pointer passing over starts nothing. The other chapters' pictures are
  dimmed and blurred by `effect.panel.blur`.

## Consequences

- The cut-off smallest window is gone: below the floor the panels show.
- Every piece the morph moves is marked `data-morph` in the hero, the action
  bar and the panels. A change to those components keeps the marks, or the
  morph finds a piece missing and swaps without motion.
- The full layout stays mounted under compact, so its stores and timers run as
  before; nothing new polls.
