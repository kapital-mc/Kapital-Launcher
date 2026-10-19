# ADR-0010: The app draws its own window header

**Status:** accepted, 2026-09-29.

## Context

The OS title bar is a grey strip that matches nothing else in the window and
has no room for the app's own controls; settings (#5) needed an entry point
that is not tied to one chapter. The author asked for a header like a
browser's tab strip: the brand at the left, the gear at the right, and no
system bar.

Electron can hide the title bar and keep Windows' real caption buttons
(Window Controls Overlay). Wails v2 cannot: a frameless window on Windows
loses the caption buttons with the bar. macOS can: a hidden title bar keeps
the traffic lights.

## Decision

- **A full-width header bar, `layout.titlebar` tall**
  (`components/shell/HeaderBar.tsx`). Left: the Kapital Launcher logo (the
  author's artwork, kept full size in `../Art`), which replaces the Kapitel
  Kapital wordmark the sidebar used to carry. The download site (ADR-9)
  switched to the launcher's own logo too, so the wordmark file is gone.
  Right: the settings gear. The bar's empty area drags the window through
  Wails' `--wails-draggable` property, and a double-click maximises; its
  controls opt out of both.
- **Windows (and Linux): frameless, with our own buttons.** Minimise,
  maximise or restore, and close, 46px wide at the bar's height, close red
  on hover, glyphs from `lib/icons.ts`. They call the Wails runtime.
- **macOS: a hidden title bar, native traffic lights.** No window buttons are
  drawn; the brand starts after the lights. Full-screen, zoom and the
  window menu stay the system's.

## Consequences

- Windows 11's Snap Layouts flyout on hovering maximise is gone; Win+Z,
  Win+arrow and dragging to an edge still snap, since the drag is a native
  caption drag.
- Resizing from the edges is Wails' frameless resize handling on Windows.
- `[verify]` on macOS: that `TitleBarHidden` places the traffic lights inside
  the 40px bar and that `pl-20` clears them.
- Reopen when moving to Wails v3 (ADR-1) if it offers the real caption
  buttons without the bar.

## Amendment, 2026-10-09: the macOS menu bar (#225)

"The window menu stays the system's" assumed a menu bar macOS draws by itself.
It does not: Wails v2 installs one only when `options.App.Menu` is set, and the
launcher set none, so Cmd+C, Cmd+V, Cmd+A, Cmd+Z, Cmd+Q and Cmd+M, which are key
equivalents of menu items, reached nothing. On darwin the window now carries
Wails' App, Edit and Window menus (`appMenu` in `menu.go`); Windows and Linux
keep no menu bar. The macOS workflow starts the built bundle and reads its menu
bar where the runner allows it.

The header no longer starts as the Windows bar until Wails names the platform:
`lib/platform.ts` guesses it from the WebView's user agent, so a Mac never shows
the Windows buttons, nor the brand under the traffic lights, at start.

## Amendment, 2026-10-18: no wordmark, and a macOS bar as tall as its title bar

The author asked for the Kapital Launcher logo to go from the header bar: the
bar now holds only the gear, and the window buttons off macOS. The logo stays
on the download site.

On macOS the traffic lights did not sit in the middle of the 40px bar. A
hidden title bar leaves them where a standard title bar puts them: in the
author's screenshot (macOS, 2x) they were 7.5pt from the window's top and left
edges and 12pt tall, centred about 13pt down, so 21pt of bar showed under them
against 7pt above. Wails v2 has no option to move them (Electron's
`trafficLightPosition` has no equivalent in its `mac.Options`). So on macOS the bar is
`layout.titlebarMac` (28px) tall, the height of a standard title bar, and the
gear is the smaller button, inset from the right as the lights are from the
left. Windows and Linux keep 40px. The screenshot settles the `[verify]` above:
the lights sit inside the bar; `pl-20` went with the logo. `[verify]` on the
author's Mac: the lights centred in the 28px bar.

## Amendment, 2026-10-19: the sidebar button on the left of the bar

The author asked for a compact sidebar that shows only the chapter icons and a
square account tile. Its button opens the bar on the left, where the wordmark
was: on macOS 64px in, past the traffic lights and inside the compact
sidebar's width, and elsewhere 16px in, the sidebar's own padding, so the
buttons' hover boxes line up with the chapter tiles' left edge. Compacting moves only the sidebar's right edge (to
`layout.sidebarCompact`, an icon row's width between the sidebar's padding),
so the icons and the initials never move sideways; the text fades out first and
back in once there is room. Compact, the Chapters heading folds away and the
sidebar's padding matches the chapter card's, so the first chapter tile and the
account tile line up with the card's top and bottom edges (the author asked,
2026-10-19). The choice is saved as `sidebarCompact` in the
settings and taken without animation on launch. `[verify]` on the author's Mac:
the button clear of the traffic lights, and the fold animating in WKWebView
(it moves grid tracks between 1fr and 0fr; a WebView that cannot interpolate
them snaps instead).

The downloads section above the account tile (Tailscale, 2026-10-19) keeps each
bar and its percentage when compact, with a spinner beside the percentage (the
author found the bare number odd, 2026-10-19); its heading, the names and the steps
fold away with the rest of the sidebar's text. There is no paused state:
every download runs at once, none waits behind another.

The author then asked for the bar's left side to carry the launcher's tools
after the sidebar's button: Settings (moved from the right, which now holds
only the window buttons off macOS), Open Prism (greyed out while there is no
Prism), the wiki's home page, and, only while a newer launcher is out,
an update button in the accent that brings the update popup back after Not
now (`components/shell/HeaderTools`).

Each of those buttons names itself in a tooltip after two seconds of hover
(`motion.duration.tooltip`), drawn by `IconButton`'s `tooltip` prop rather than
the browser's own: the mono face, square corners and a hairline, under the
button's left edge. Keyboard focus shows it at once. The button keeps its
`aria-label`, so the tooltip is `aria-hidden`.

A settings page's column grows with the card up to the other pages' measure
(`max-w-200`), and the column and its nav are centred only past it, so
compacting or widening the sidebar resizes the sections along with the card
rather than sliding a fixed block (the author asked, 2026-10-19).

## Amendment, 2026-10-19: a macOS bar measured from AppKit

The author compared the 28px bar with the Claude app, whose buttons have room
above and below and share the traffic lights' centre line, and asked whether
the launcher could match it. The 2026-10-18 amendment was wrong that Wails v2
cannot: it cannot move the lights, but it can make the title bar taller, which
moves them down with it. `mac.TitleBarHiddenInset()` gives the window an empty
`NSToolbar`, and a window with a toolbar has a taller title bar.

How tall depends on the toolbar style, which Wails leaves on automatic, and
automatic resolves differently by macOS version: the macOS CI runners measured
a 38pt bar on macOS 15 and a 52pt bar on macOS 26. So `backend/windowchrome`
sets the style to unified at start (one Objective-C call, on the main thread)
and measures the lights: twice their centre's distance from the window's top is
the bar's height, and the zoom button's right edge is where the header's
buttons may start. Unified measured 52pt on both, with the lights ending at
73pt on macOS 15 and 79pt on macOS 26. The header reads that through
`GetWindowChrome` and sets its height and left inset from it (two computed
custom properties, the one inline style). `layout.titlebarMac` (52px) and
`layout.controlsMac` (80px) are the fallback when nothing was measured.

The Mac CI job now checks this on both runners: the measurement must match
what System Events reports for the lights, a real click on the header's first
button must compact the sidebar (the toolbar does not swallow clicks over the
WebView), and pictures of the bar before and after are uploaded.

The sidebar button now starts about 12px past the lights, around 85 to 91px
in, rather than at 64px, so on macOS it no longer sits over the compact
sidebar's icon column. `[verify]` on the author's Mac: the bar's height and the
buttons centred on the lights, every header button clickable, the window
dragged by the bar's empty space, and full screen entered and left.
