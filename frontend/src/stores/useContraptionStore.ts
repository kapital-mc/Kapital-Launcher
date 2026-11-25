import { create } from 'zustand'
import type { Contraption } from '../types'
import { errMsg, hasWailsBridge } from '../lib/ipc'
import { contraptionKey } from '../lib/contraptions'
import {
  DeleteContraption,
  ExportContraptionsZip,
  ExportContraption,
  ImportContraptions,
  ListContraptions,
  OpenContraptionFolder,
  RenameContraption,
} from '../../wailsjs/go/main/App'

/**
 * A chapter's saved contraptions, as the Contraptions page shows them: the Tool Gun's blueprints
 * and Create's schematics in its game folder. Go reads every file and does every change; this is
 * the last list it sent, read when the page opens and after each change. Only the lazy page
 * imports it, and the page empties it on the way out.
 *
 * A change is not drawn ahead of Go: the answer is the folder listed again, and a refusal is
 * rethrown for the page to show. With no bridge (the browser-only preview) nothing can be read.
 */
interface ContraptionStore {
  /** The chapter the list is of, and the list, newest first; null before the first read. */
  chapterId: string | null
  list: Contraption[] | null
  unavailable: boolean
  listError: string | null
  /** The row a change is in flight for (contraptionKey), or 'import' or 'export'. */
  busy: string | null
  load: (chapterId: string) => Promise<void>
  /** Renames a file; rejects with Go's reason. */
  rename: (chapterId: string, c: Contraption, name: string) => Promise<void>
  /** Deletes a file; rejects with Go's reason. */
  remove: (chapterId: string, c: Contraption) => Promise<void>
  /** Opens the file picker and copies the picks in; resolves to how many arrived. */
  importFiles: (chapterId: string) => Promise<number>
  /** Opens the save dialog and copies the file there; resolves to false when cancelled. */
  exportFile: (chapterId: string, c: Contraption) => Promise<boolean>
  /** Opens the save dialog and zips the picked files there; resolves to how many went in, 0 when cancelled. */
  exportZip: (chapterId: string, picks: readonly Contraption[]) => Promise<number>
  openFolder: (chapterId: string, kind: string) => Promise<void>
  clear: () => void
}

const EMPTY = { chapterId: null, list: null, unavailable: false, listError: null, busy: null }

export const useContraptionStore = create<ContraptionStore>((set, get) => {
  // Runs one change with its row marked busy, then takes Go's new list.
  const change = async (chapterId: string, busy: string, call: () => Promise<Contraption[]>) => {
    if (!hasWailsBridge()) return
    set({ busy })
    try {
      const list = await call()
      if (get().chapterId === chapterId) set({ list, listError: null })
    } finally {
      set({ busy: null })
    }
  }
  return {
    ...EMPTY,

    load: async (chapterId) => {
      if (!hasWailsBridge()) {
        set({ ...EMPTY, chapterId, unavailable: true })
        return
      }
      if (get().chapterId !== chapterId) set({ ...EMPTY, chapterId })
      try {
        const list = (await ListContraptions(chapterId)) as Contraption[]
        if (get().chapterId === chapterId) set({ list, listError: null })
      } catch (e) {
        if (get().chapterId === chapterId) set({ listError: errMsg(e) })
      }
    },

    rename: (chapterId, c, name) =>
      change(
        chapterId,
        contraptionKey(c),
        async () => (await RenameContraption(chapterId, c.kind, c.file, name)) as Contraption[],
      ),

    remove: (chapterId, c) =>
      change(
        chapterId,
        contraptionKey(c),
        async () => (await DeleteContraption(chapterId, c.kind, c.file)) as Contraption[],
      ),

    importFiles: async (chapterId) => {
      if (!hasWailsBridge()) return 0
      set({ busy: 'import' })
      try {
        return await ImportContraptions(chapterId)
      } finally {
        set({ busy: null })
        // Some files may have arrived before a refusal: list again either way.
        await get().load(chapterId)
      }
    },

    exportFile: async (chapterId, c) => {
      if (!hasWailsBridge()) return false
      return ExportContraption(chapterId, c.kind, c.file)
    },

    exportZip: async (chapterId, picks) => {
      if (!hasWailsBridge()) return 0
      set({ busy: 'export' })
      try {
        return await ExportContraptionsZip(
          chapterId,
          picks.map((c) => ({ kind: c.kind, file: c.file })),
        )
      } finally {
        set({ busy: null })
      }
    },

    openFolder: async (chapterId, kind) => {
      if (!hasWailsBridge()) return
      await OpenContraptionFolder(chapterId, kind)
    },

    clear: () => set(EMPTY),
  }
})
