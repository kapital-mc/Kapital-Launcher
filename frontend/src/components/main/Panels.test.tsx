import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { Panels } from './Panels'
import { BUNDLED_MANIFEST } from '../../lib/manifest'

vi.mock('../../../wailsjs/go/main/App')

const chapter = BUNDLED_MANIFEST.chapters[2]!
const noop = () => undefined

function panels(over: Partial<ComponentProps<typeof Panels>> = {}) {
  return render(
    <Panels
      chapter={chapter}
      installed={true}
      sizeBytes={undefined}
      packState={undefined}
      changelog={undefined}
      wikiPage={undefined}
      onOpenWiki={noop}
      {...over}
    />,
  )
}

const row = (label: string) => screen.getByText(label).nextElementSibling as HTMLElement

describe('Panels', () => {
  afterEach(cleanup)

  it('says Not installed for the version and the size of a chapter that is not installed', () => {
    panels({ installed: false })
    expect(row('Version')).toHaveTextContent('Not installed')
    expect(row('Size')).toHaveTextContent('Not installed')
    expect(row('Version')).toHaveClass('text-fg-faint')
    expect(screen.queryByText(/PLACEHOLDER/)).toBeNull()
  })

  it('says Unknown, faint, for a fact that nobody has settled', () => {
    const open = { ...chapter, pack: { ...chapter.pack, mods: null, minecraft: '[PLACEHOLDER]' } }
    panels({ chapter: open, installed: true })
    expect(row('Mods')).toHaveTextContent('Unknown')
    expect(row('Minecraft')).toHaveTextContent('Unknown')
    expect(row('Size')).toHaveTextContent('Unknown')
    expect(row('Mods')).not.toHaveClass('font-mono')
  })

  it('keeps numbers and versions in the data face and words in the UI face', () => {
    panels({ sizeBytes: 1_500_000_000 })
    expect(row('Size')).toHaveTextContent('1.5 GB')
    expect(row('Size')).toHaveClass('font-mono')
    cleanup()
    const visuals = { ...chapter, pack: { ...chapter.pack, type: 'client-visuals' as const } }
    panels({ chapter: visuals })
    expect(row('Loader')).toHaveTextContent('Client visuals')
    expect(row('Loader')).not.toHaveClass('font-mono')
  })

  it('gives the wiki post one title line and four reserved text lines, with the link pinned below', () => {
    const onOpenWiki = vi.fn()
    panels({ onOpenWiki })
    const title = screen.getByText(chapter.wiki.title)
    const line = screen.getByText(chapter.wiki.line)
    expect(title).toHaveClass('line-clamp-1')
    expect(line).toHaveClass('line-clamp-4', 'min-h-[4lh]')
    const link = screen.getByRole('button', { name: 'Read the history →' })
    expect(link).toHaveClass('mt-auto')
    fireEvent.click(link)
    expect(onOpenWiki).toHaveBeenCalledOnce()
  })

  it('shows only the latest changelog entries, each clamped', () => {
    const changelog = ['1.0.5', '1.0.4', '1.0.3', '1.0.2', '1.0.1'].map((version) => ({
      version,
      summary: `Summary of ${version}`,
    }))
    panels({ chapter: { ...chapter, changelog } })
    expect(screen.getByText('1.0.5')).toBeInTheDocument()
    expect(screen.getByText('1.0.3')).toBeInTheDocument()
    expect(screen.queryByText('1.0.2')).toBeNull()
    expect(screen.getByText('Summary of 1.0.5')).toHaveClass('line-clamp-2')
  })

  it('shows the entries of the pack source by their summary, the details in the tooltip', () => {
    const entries = ['1.0.5', '1.0.4', '1.0.3', '1.0.2'].map((version) => ({
      version,
      date: '2026-10-03',
      summary: `First of ${version}.`,
      details: `First of ${version}.\nSecond of ${version}.`,
    }))
    const manifested = { ...chapter, changelog: [{ version: '0.1', summary: 'Manifest entry.' }] }
    panels({ chapter: manifested, changelog: { chapterId: chapter.id, checked: true, entries } })
    expect(screen.getByText('1.0.5')).toHaveClass('text-accent', 'font-mono')
    expect(screen.getByText('First of 1.0.5.')).toHaveClass('line-clamp-2')
    expect(screen.getByText('First of 1.0.5.')).toHaveAttribute(
      'title',
      'First of 1.0.5.\nSecond of 1.0.5.',
    )
    expect(screen.queryByText('Second of 1.0.5.')).toBeNull()
    expect(screen.queryByText('1.0.2')).toBeNull()
    expect(screen.queryByText('Manifest entry.')).toBeNull()
  })

  it('falls back to the manifest changelog when the source has none or cannot be read', () => {
    const manifested = { ...chapter, changelog: [{ version: '0.1', summary: 'Manifest entry.' }] }
    const none = { chapterId: chapter.id, checked: true, entries: [] }
    panels({ chapter: manifested, changelog: none })
    expect(screen.getByText('Manifest entry.')).toBeInTheDocument()
    cleanup()
    panels({ chapter: manifested, changelog: { ...none, checked: false } })
    expect(screen.getByText('Manifest entry.')).toBeInTheDocument()
  })

  it('has one sentence for an empty changelog', () => {
    panels()
    expect(screen.getByText('No entries yet.')).toHaveClass('text-fg-faint', 'text-sm')
  })

  it('opens the full changelog from its expand button, every entry with every line', async () => {
    const entries = ['1.0.5', '1.0.4', '1.0.3', '1.0.2'].map((version) => ({
      version,
      date: '2026-10-03',
      summary: `Fixed Map ${version}; Added Xaeros Minimap`,
      details: `Fixed Map ${version}\nAdded Xaeros Minimap`,
    }))
    panels({ changelog: { chapterId: chapter.id, checked: true, entries } })
    fireEvent.click(screen.getByRole('button', { name: 'Full changelog' }))
    const dialog = await screen.findByRole('dialog', { name: `${chapter.name} changelog` })
    expect(within(dialog).getByText('1.0.2')).toBeInTheDocument()
    expect(within(dialog).getByText('Fixed Map 1.0.2')).toBeInTheDocument()
    expect(within(dialog).getAllByText('2026-10-03')).toHaveLength(4)
    fireEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('splits a manifest entry at its semicolons in the popup', async () => {
    const changelog = [{ version: '0.1', summary: 'Added Create; Fixed Teleport Roles' }]
    panels({ chapter: { ...chapter, changelog } })
    fireEvent.click(screen.getByRole('button', { name: 'Full changelog' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Added Create')).toBeInTheDocument()
    expect(within(dialog).getByText('Fixed Teleport Roles')).toBeInTheDocument()
  })

  it('opens the pack details on a click anywhere on the column', async () => {
    const toggles = [{ name: 'Distant Horizons', jarPrefix: 'DistantHorizons' }]
    panels({ chapter: { ...chapter, pack: { ...chapter.pack, memoryGb: 8, toggles } } })
    fireEvent.click(screen.getByText('Loader'))
    const dialog = await screen.findByRole('dialog', { name: `${chapter.name} pack` })
    expect(within(dialog).getByText(chapter.blurb)).toBeInTheDocument()
    expect(within(dialog).getByText('Recommended memory').nextElementSibling).toHaveTextContent(
      '8 GB',
    )
    expect(within(dialog).getByText('Optional mods').nextElementSibling).toHaveTextContent(
      'Distant Horizons',
    )
  })

  it('shows the wiki page whole with its facts, and opens it from the popup', async () => {
    const onOpenWiki = vi.fn()
    const wikiPage = {
      title: 'The Obelisk',
      line: 'A tower on the water, with a long line the column cuts.',
      url: '/wiki/locations/the-obelisk',
      eras: ['Frangfurd'],
      id: 'locations/The Obelisk.md',
      related: [],
      type: 'location',
      category: 'landmark',
      facts: [
        { label: 'Within', value: 'Frangfurder Sea' },
        { label: 'Built', value: '2566' },
      ],
    }
    panels({ wikiPage, onOpenWiki })
    fireEvent.click(screen.getByRole('button', { name: 'Wiki details' }))
    const dialog = await screen.findByRole('dialog', { name: 'The Obelisk' })
    expect(within(dialog).getByText('Landmark · Frangfurd')).toBeInTheDocument()
    expect(within(dialog).getByText('Within').nextElementSibling).toHaveTextContent(
      'Frangfurder Sea',
    )
    fireEvent.click(within(dialog).getByRole('button', { name: 'Read on the wiki' }))
    expect(onOpenWiki).toHaveBeenCalledOnce()
  })

  it('does not open a popup from the column link, which opens the wiki itself', async () => {
    const onOpenWiki = vi.fn()
    panels({ onOpenWiki })
    fireEvent.click(screen.getByRole('button', { name: 'Read the history →' }))
    expect(onOpenWiki).toHaveBeenCalledOnce()
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})
