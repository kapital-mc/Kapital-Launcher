/**
 * Wording and bounds for a chapter's settings panel (#36), kept pure so the
 * panel only renders them.
 */

/** The launcher's fixed heap minimum, Prism's own default (services.minMemMiB). */
export const MIN_MEMORY_MB = 512

/** The slider's step: a quarter gigabyte reads as a round number at every stop. */
export const MEMORY_STEP_MB = 256

/** The slider's end when the machine's memory is unknown. */
const FALLBACK_MAX_MB = 32768

/** A memory amount for a readout: gigabytes with one decimal from a gigabyte up. */
export function memoryLabel(mb: number): string {
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`
  return `${mb} MB`
}

/** The slider's end: the machine's memory, rounded down to a step. */
export function sliderMaxMb(machineMemoryMb: number): number {
  const ceiling = machineMemoryMb > 0 ? machineMemoryMb : FALLBACK_MAX_MB
  return Math.max(MIN_MEMORY_MB, Math.floor(ceiling / MEMORY_STEP_MB) * MEMORY_STEP_MB)
}

/** What a preset does, for its radio label; an unknown name is shown as is. */
export function presetLabel(name: string): string {
  switch (name) {
    case '':
      return 'Default'
    case 'zgc':
      return 'ZGC garbage collector'
    default:
      return name
  }
}

/** What a preset needs, under its label, when it needs something (issue 189). */
export function presetNote(name: string): string | undefined {
  return name === 'zgc' ? 'Java 21 and later' : undefined
}

const GB = 1024

/**
 * How far a memory setting is from comfortable, from 0 (fine) through 1
 * (worth a look) to 2 (likely trouble), for the readout's colour. Within a
 * gigabyte of the pack's recommendation is fine, and four away is trouble,
 * either side: too little runs out mid-game, too much buys nothing the pack
 * uses. Whatever the pack says, the machine keeps some for itself and for the
 * game's memory outside Java's heap: six gigabytes left is fine, three is
 * trouble. The two measures are linear between their ends, so the colour
 * moves with the slider rather than jumping at a line.
 */
export function memoryTone(mb: number, packMb: number, machineMb: number): number {
  const span = (from: number, to: number, v: number) => (2 * (v - from)) / (to - from)
  const below = packMb > 0 ? span(packMb - GB, packMb - 4 * GB, mb) : 0
  const above = packMb > 0 ? span(packMb + GB, packMb + 4 * GB, mb) : 0
  const left = machineMb > 0 ? span(6 * GB, 3 * GB, machineMb - mb) : 0
  return Math.min(2, Math.max(0, below, above, left))
}
