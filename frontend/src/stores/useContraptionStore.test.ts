import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as Bindings from '../../wailsjs/go/main/App'
import { useContraptionStore } from './useContraptionStore'
import type { Contraption } from '../types'

vi.mock('../../wailsjs/go/main/App')

const bus = (name = 'Bus'): Contraption => ({
  kind: 'toolgun',
  file: `${name}.excraft`,
  name,
  bytes: 10,
  modified: 0,
  blocks: 609,
  blockTypes: 31,
  subLevels: 1,
  materials: [],
  items: [],
  unread: false,
})

describe('useContraptionStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useContraptionStore.getState().clear()
  })
  afterEach(() => Reflect.deleteProperty(window, 'go'))

  it('reads a chapter, and says so when there is no bridge', async () => {
    vi.mocked(Bindings.ListContraptions).mockResolvedValue([bus()] as never)
    await useContraptionStore.getState().load('frangfurd')
    expect(useContraptionStore.getState().list).toEqual([bus()])
    Reflect.deleteProperty(window, 'go')
    await useContraptionStore.getState().load('frangfurd')
    expect(useContraptionStore.getState().unavailable).toBe(true)
    expect(useContraptionStore.getState().list).toBeNull()
  })

  it('keeps the reason a read was refused', async () => {
    vi.mocked(Bindings.ListContraptions).mockRejectedValue('Frangfurd is not installed')
    await useContraptionStore.getState().load('frangfurd')
    expect(useContraptionStore.getState().listError).toBe('Frangfurd is not installed')
  })

  it('takes Go’s list after a rename, and rethrows a refusal', async () => {
    vi.mocked(Bindings.ListContraptions).mockResolvedValue([bus()] as never)
    await useContraptionStore.getState().load('frangfurd')
    vi.mocked(Bindings.RenameContraption).mockResolvedValue([bus('Mk2')] as never)
    const done = useContraptionStore.getState().rename('frangfurd', bus(), 'Mk2')
    expect(useContraptionStore.getState().busy).toBe('toolgun/Bus.excraft')
    await done
    expect(Bindings.RenameContraption).toHaveBeenCalledWith(
      'frangfurd',
      'toolgun',
      'Bus.excraft',
      'Mk2',
    )
    expect(useContraptionStore.getState().list).toEqual([bus('Mk2')])
    expect(useContraptionStore.getState().busy).toBeNull()

    vi.mocked(Bindings.DeleteContraption).mockRejectedValue('no such file')
    await expect(useContraptionStore.getState().remove('frangfurd', bus('Mk2'))).rejects.toBe(
      'no such file',
    )
    expect(useContraptionStore.getState().list).toEqual([bus('Mk2')])
    expect(useContraptionStore.getState().busy).toBeNull()
  })

  it('lists again after an import, even one Go refused in part', async () => {
    vi.mocked(Bindings.ListContraptions).mockResolvedValue([bus(), bus('New')] as never)
    vi.mocked(Bindings.ImportContraptions).mockRejectedValue('Bad.nbt: not a gzipped NBT file')
    await useContraptionStore.getState().load('frangfurd')
    await expect(useContraptionStore.getState().importFiles('frangfurd')).rejects.toBe(
      'Bad.nbt: not a gzipped NBT file',
    )
    expect(Bindings.ListContraptions).toHaveBeenCalledTimes(2)
    expect(useContraptionStore.getState().busy).toBeNull()
  })

  it('exports and opens folders through Go, and does nothing without a bridge', async () => {
    vi.mocked(Bindings.ExportContraption).mockResolvedValue(true)
    await expect(useContraptionStore.getState().exportFile('frangfurd', bus())).resolves.toBe(true)
    await useContraptionStore.getState().openFolder('frangfurd', 'schematic')
    expect(Bindings.OpenContraptionFolder).toHaveBeenCalledWith('frangfurd', 'schematic')
    Reflect.deleteProperty(window, 'go')
    await expect(useContraptionStore.getState().exportFile('frangfurd', bus())).resolves.toBe(false)
    await expect(useContraptionStore.getState().importFiles('frangfurd')).resolves.toBe(0)
    await useContraptionStore.getState().rename('frangfurd', bus(), 'x')
    expect(Bindings.RenameContraption).not.toHaveBeenCalled()
  })
})
