import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import * as Bindings from '../../../wailsjs/go/main/App'
import { models } from '../../../wailsjs/go/models'
import { useEngineStore } from '../../stores/useEngineStore'
import { useContraptionStore } from '../../stores/useContraptionStore'
import { BUNDLED_MANIFEST } from '../../lib/manifest'
import type { Contraption } from '../../types'
import { ContraptionsPanel } from './ContraptionsPanel'

vi.mock('../../../wailsjs/go/main/App')

const chapter = BUNDLED_MANIFEST.chapters.find((c) => c.id === 'frangfurd')!

const report = (present: boolean) =>
  models.InstanceReport.createFrom({
    root: 'C:/Prism',
    dir: 'C:/Prism/instances',
    present: { frangfurd: present },
    sizeBytes: {},
  })

const bus: Contraption = {
  kind: 'toolgun',
  file: 'Bus.excraft',
  name: 'Bus',
  bytes: 10_400,
  modified: Date.UTC(2026, 10, 25),
  blocks: 609,
  blockTypes: 2,
  subLevels: 1,
  materials: [
    { id: 'minecraft:oak_planks', count: 600 },
    { id: 'create:shaft', count: 9 },
  ],
  items: [{ id: 'minecraft:coal', count: 64 }],
  unread: false,
}
const bridge: Contraption = {
  ...bus,
  kind: 'schematic',
  file: 'bridge.nbt',
  name: 'bridge',
  size: [3, 2, 1],
  materials: [],
  items: [],
}

const panel = () => render(<ContraptionsPanel chapter={chapter} onClose={() => undefined} />)

describe('ContraptionsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useContraptionStore.getState().clear()
    useEngineStore.setState({ instances: report(true) })
    vi.mocked(Bindings.ListContraptions).mockResolvedValue([bus, bridge] as never)
  })
  afterEach(() => {
    cleanup()
    Reflect.deleteProperty(window, 'go')
  })

  it('lists each kind in its own section with its count', async () => {
    panel()
    const toolgun = await screen.findByRole('region', { name: 'Tool Gun' })
    expect(within(toolgun).getByText('Bus')).toBeInTheDocument()
    expect(within(toolgun).getByText('1')).toBeInTheDocument()
    const schematics = screen.getByRole('region', { name: 'Schematics' })
    expect(within(schematics).getByText('bridge')).toBeInTheDocument()
    expect(Bindings.ListContraptions).toHaveBeenCalledExactlyOnceWith('frangfurd')
  })

  it('opens a row to its materials and renames it on commit', async () => {
    vi.mocked(Bindings.RenameContraption).mockResolvedValue([
      { ...bus, file: 'Bus Mk2.excraft', name: 'Bus Mk2' },
      bridge,
    ] as never)
    panel()
    fireEvent.click(await screen.findByRole('button', { name: /^Bus/ }))
    const materials = screen.getByRole('button', { name: /^Materials/ })
    expect(materials).toHaveAttribute('aria-expanded', 'false')
    expect(materials).toHaveTextContent('2 block types · 1 item')
    fireEvent.click(materials)
    expect(materials).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Oak planks')).toBeInTheDocument()
    expect(screen.getByText('create')).toBeInTheDocument()
    expect(screen.getByText('Coal')).toBeInTheDocument()
    const name = screen.getByLabelText('Name')
    fireEvent.change(name, { target: { value: 'Bus Mk2' } })
    fireEvent.keyDown(name, { key: 'Enter' })
    await waitFor(() =>
      expect(Bindings.RenameContraption).toHaveBeenCalledExactlyOnceWith(
        'frangfurd',
        'toolgun',
        'Bus.excraft',
        'Bus Mk2',
      ),
    )
    // The renamed row stays open.
    expect(await screen.findByLabelText('Name')).toHaveValue('Bus Mk2')
  })

  it('shows a refused rename under the field', async () => {
    vi.mocked(Bindings.RenameContraption).mockRejectedValue(new Error('Taken already exists'))
    panel()
    fireEvent.click(await screen.findByRole('button', { name: /^Bus/ }))
    const name = screen.getByLabelText('Name')
    fireEvent.change(name, { target: { value: 'Taken' } })
    fireEvent.keyDown(name, { key: 'Enter' })
    expect(await screen.findByText('Taken already exists')).toBeInTheDocument()
  })

  it('asks before a delete, every time', async () => {
    vi.mocked(Bindings.DeleteContraption).mockResolvedValue([bridge] as never)
    panel()
    fireEvent.click(await screen.findByRole('button', { name: /^Bus/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = screen.getByRole('dialog')
    expect(within(dialog).queryByRole('checkbox')).toBeNull()
    expect(Bindings.DeleteContraption).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(screen.queryByRole('region', { name: 'Tool Gun' })).toBeNull())
    expect(Bindings.DeleteContraption).toHaveBeenCalledExactlyOnceWith(
      'frangfurd',
      'toolgun',
      'Bus.excraft',
    )
  })

  it('imports, lists again and shows a refusal', async () => {
    vi.mocked(Bindings.ImportContraptions).mockRejectedValue(new Error('Bad.nbt: not a schematic'))
    panel()
    await screen.findByRole('region', { name: 'Tool Gun' })
    fireEvent.click(screen.getByRole('button', { name: 'Import' }))
    expect(await screen.findByText('Bad.nbt: not a schematic')).toBeInTheDocument()
    await waitFor(() => expect(Bindings.ListContraptions).toHaveBeenCalledTimes(2))
  })

  it('exports and opens a kind’s folder', async () => {
    vi.mocked(Bindings.ExportContraption).mockResolvedValue(true)
    vi.mocked(Bindings.OpenContraptionFolder).mockResolvedValue()
    panel()
    fireEvent.click(await screen.findByRole('button', { name: /^Bus/ }))
    const toolgun = screen.getByRole('region', { name: 'Tool Gun' })
    fireEvent.click(within(toolgun).getByRole('button', { name: 'Export' }))
    await waitFor(() =>
      expect(Bindings.ExportContraption).toHaveBeenCalledExactlyOnceWith(
        'frangfurd',
        'toolgun',
        'Bus.excraft',
      ),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Open the Schematics folder' }))
    await waitFor(() =>
      expect(Bindings.OpenContraptionFolder).toHaveBeenCalledExactlyOnceWith(
        'frangfurd',
        'schematic',
      ),
    )
  })

  it('exports the ticked files into one zip', async () => {
    vi.mocked(Bindings.ExportContraptionsZip).mockResolvedValue(1)
    panel()
    await screen.findByRole('region', { name: 'Tool Gun' })
    fireEvent.click(screen.getByRole('button', { name: 'Export' }))
    const dialog = screen.getByRole('dialog', { name: 'Export contraptions' })
    expect(within(dialog).getByText('2 of 2 files')).toBeInTheDocument()
    // A kind's files fold open, each ticked; clearing one leaves the kind's box mixed.
    fireEvent.click(within(dialog).getByRole('button', { name: 'Tool Gun files' }))
    fireEvent.click(within(dialog).getByRole('checkbox', { name: 'Bus' }))
    expect(within(dialog).getByRole('checkbox', { name: 'Tool Gun' })).not.toBeChecked()
    // Clearing the schematics leaves nothing to export.
    fireEvent.click(within(dialog).getByRole('checkbox', { name: 'Schematics' }))
    expect(within(dialog).getByRole('button', { name: 'Export' })).toBeDisabled()
    fireEvent.click(within(dialog).getByRole('checkbox', { name: 'Schematics' }))
    fireEvent.click(within(dialog).getByRole('button', { name: 'Export' }))
    await waitFor(() =>
      expect(Bindings.ExportContraptionsZip).toHaveBeenCalledExactlyOnceWith('frangfurd', [
        { kind: 'schematic', file: 'bridge.nbt' },
      ]),
    )
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('says so when there are none, and when the file cannot be read', async () => {
    vi.mocked(Bindings.ListContraptions).mockResolvedValue([] as never)
    panel()
    expect(await screen.findByText('No saved contraptions.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export' })).toBeDisabled()
    cleanup()
    useContraptionStore.getState().clear()
    vi.mocked(Bindings.ListContraptions).mockResolvedValue([{ ...bus, unread: true }] as never)
    panel()
    fireEvent.click(await screen.findByRole('button', { name: /^Bus/ }))
    expect(screen.getByText('The file could not be read.')).toBeInTheDocument()
  })

  it('reads nothing for a chapter that is not installed', () => {
    useEngineStore.setState({ instances: report(false) })
    panel()
    expect(screen.getByText(`${chapter.name} is not installed yet.`)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled()
    expect(Bindings.ListContraptions).not.toHaveBeenCalled()
  })

  it('says it needs the app window without a bridge', async () => {
    Reflect.deleteProperty(window, 'go')
    panel()
    expect(
      await screen.findByText('Contraptions can only be read in the app window.'),
    ).toBeInTheDocument()
  })
})
