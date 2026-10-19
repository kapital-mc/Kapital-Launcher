/**
 * The opacity half of a fold (ui/Fold) and of the sidebar's text as it
 * narrows: out quickly as something closes, and back in only once it has
 * opened most of the way, so text is never seen squeezed or cut off. Its own
 * module so the sidebar's rows can share it without importing a component.
 */
export function fade(shown: boolean): string {
  return `transition-opacity ease-standard motion-reduce:transition-none ${
    shown ? 'opacity-100 duration-reveal delay-(--duration-reveal)' : 'opacity-0 duration-fast'
  }`
}
