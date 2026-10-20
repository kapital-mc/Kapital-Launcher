import { useEffect, useId, useRef, useState } from 'react'
import { selectWhatsNewShown, useWhatsNewStore } from '../../stores/useWhatsNewStore'
import type { Changelog, WhatsNewHighlight } from '../../types'
import { ChevronDown } from '../../lib/icons'
import { Dialog } from '../ui/Dialog'
import { Grow } from '../ui/Grow'
import { Icon } from '../ui/Icon'
import { Scrollable } from '../ui/Scrollable'
import { TextLink } from '../ui/TextLink'

const MONTHS = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
]

/** "2026-10-18" as "18 October 2026", the way the download site writes it. */
function longDate(iso: string) {
  const [year, month = 0, day] = iso.split('-').map(Number)
  const name = MONTHS[month - 1]
  return name ? `${day} ${name} ${year}` : iso
}

/**
 * What's new, on the first start after an update: the changes each release
 * since the last one seen puts forward, written by hand for it
 * (data/whatsnew.json), with small ones grouped under a heading such as Bug
 * fixes. The full changelog folds away under them and is read from the download
 * site only when opened. The X in its corner (or Escape) records the version as
 * seen, so it shows once; a click outside does not close it.
 *
 * App mounts it once for the app's lifetime, in a chunk of its own: it asks Go
 * on mount and renders nothing unless something is new.
 */
export function WhatsNew() {
  const load = useWhatsNewStore((s) => s.load)
  const shown = useWhatsNewStore(selectWhatsNewShown)
  const whatsNew = useWhatsNewStore((s) => s.whatsNew)
  const error = useWhatsNewStore((s) => s.error)
  const close = useWhatsNewStore((s) => s.close)
  useEffect(() => {
    void load()
  }, [load])
  if (!shown || !whatsNew?.releases) return null
  const releases = whatsNew.releases

  return (
    <Dialog
      title="What's new"
      facts={`Kapital Launcher ${whatsNew.version}`}
      onCancel={() => void close()}
      closable
      holdOnCover
    >
      {/* The highlights and the changelog scroll inside the box, which then fits the smallest
          window: its 640px less the layer's padding and the title. The thumb sits in
          the box's right padding. */}
      <div className="-mr-4 flex max-h-104 min-h-0">
        <Scrollable aria-label="Changes">
          <div className="flex flex-col gap-4 pr-4">
            {releases.map((r) => (
              <section key={r.version} aria-label={r.version} className="flex flex-col gap-3">
                {releases.length > 1 && (
                  <span className="text-fg-faint text-2xs tracking-eyebrow font-mono uppercase">
                    {r.version}
                  </span>
                )}
                <ul className="m-0 flex list-none flex-col gap-4 p-0">
                  {r.highlights.map((h) => (
                    <Highlight key={h.title} highlight={h} />
                  ))}
                </ul>
              </section>
            ))}
            <FullChangelog />
          </div>
        </Scrollable>
      </div>
      {error && <p className="text-danger m-0 text-xs">{error}</p>}
    </Dialog>
  )
}

/** One change: its name in front, with its line, or the small changes it groups, under it. */
function Highlight({ highlight: h }: { highlight: WhatsNewHighlight }) {
  return (
    <li className="flex gap-3">
      <span aria-hidden className="bg-accent rounded-pill mt-2 size-2 shrink-0" />
      <div className="flex min-w-0 flex-col gap-1">
        <span className="text-fg text-base font-semibold">{h.title}</span>
        {h.text && <p className="text-fg-soft m-0 leading-normal">{h.text}</p>}
        {h.items && h.items.length > 0 && (
          <ul className="text-fg-soft m-0 flex list-disc flex-col gap-0.5 pl-4 leading-normal">
            {h.items.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        )}
      </div>
    </li>
  )
}

/** Scrolls the list so the changelog's toggle is at its top, which shows as much of it as fits. */
function showFrom(el: HTMLElement | null) {
  const still = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? true
  el?.scrollIntoView?.({ block: 'start', behavior: still ? 'auto' : 'smooth' })
}

/**
 * The full changelog of the releases covered, folded until asked for. Opened,
 * the list scrolls to it as it unfolds, and again once it has, since its
 * height is only known then.
 */
function FullChangelog() {
  const [open, setOpen] = useState(false)
  const changelog = useWhatsNewStore((s) => s.changelog)
  const loadChangelog = useWhatsNewStore((s) => s.loadChangelog)
  const bodyId = useId()
  const head = useRef<HTMLButtonElement>(null)
  const toggle = () => {
    if (!open) {
      void loadChangelog()
      requestAnimationFrame(() => showFrom(head.current))
    }
    setOpen((o) => !o)
  }
  return (
    <div
      className="border-line flex flex-col border-t pt-3"
      onTransitionEnd={(e) => {
        if (open && e.propertyName === 'grid-template-rows') showFrom(head.current)
      }}
    >
      <button
        ref={head}
        type="button"
        aria-expanded={open}
        aria-controls={bodyId}
        onClick={toggle}
        className="group text-fg-muted hover:text-fg duration-fast ease-standard flex items-center justify-between text-xs transition-colors motion-reduce:transition-none"
      >
        <span>Full changelog</span>
        <Icon
          icon={ChevronDown}
          size="sm"
          className={`duration-fast ease-standard transition-transform motion-reduce:transition-none ${
            open ? 'rotate-180' : ''
          }`}
        />
      </button>
      <Grow inner="pt-3">
        {open && (
          <div id={bodyId}>
            <ChangelogBody changelog={changelog} />
          </div>
        )}
      </Grow>
    </div>
  )
}

function ChangelogBody({ changelog }: { changelog: Changelog | null }) {
  const openPage = useWhatsNewStore((s) => s.openPage)
  if (!changelog) return <p className="text-fg-faint m-0 text-xs">Reading the changelog…</p>
  const releases = changelog.releases ?? []
  if (releases.length === 0) {
    return (
      <p className="text-fg-muted m-0 text-xs leading-normal">
        {changelog.reason ? 'The changelog could not be read.' : 'No notes for this version yet.'}{' '}
        {changelog.page && (
          <TextLink size="xs" onClick={() => void openPage()}>
            Every release is on the download site.
          </TextLink>
        )}
      </p>
    )
  }
  return (
    <div className="flex flex-col gap-5">
      {releases.map((r) => (
        <section key={r.tag} aria-label={r.tag} className="flex flex-col gap-2">
          <div className="flex items-baseline justify-between gap-4">
            <span className="text-fg font-mono text-xs font-medium">{r.tag}</span>
            <time className="text-fg-faint font-mono text-xs" dateTime={r.date}>
              {longDate(r.date)}
            </time>
          </div>
          {(r.sections ?? []).length === 0 && (
            <p className="text-fg-faint m-0 text-xs">Maintenance only.</p>
          )}
          {(r.sections ?? []).map((s) => (
            <div key={s.title} className="flex flex-col gap-1">
              <span className="text-fg-faint text-2xs tracking-eyebrow uppercase">{s.title}</span>
              <ul className="text-fg-soft m-0 flex list-disc flex-col gap-0.5 pl-4 text-xs leading-normal">
                {s.items.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ))}
        </section>
      ))}
    </div>
  )
}
