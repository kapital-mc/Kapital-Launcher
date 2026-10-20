import { useEffect, useRef } from 'react'
import type { HTMLAttributes, ReactNode, Ref } from 'react'
import { smoothResize } from '../../lib/smoothResize'

interface Props extends HTMLAttributes<HTMLDivElement> {
  /** The box's own look: background, border, radius, padding. */
  className?: string
  /** The content's layout inside it, such as `flex flex-col gap-4`. */
  inner?: string
  ref?: Ref<HTMLDivElement>
  children: ReactNode
}

/**
 * A card or a popup's box whose height follows its content smoothly, with
 * whatever is added to it later fading in (`lib/smoothResize`). It is what
 * `SettingsCard`, `Dialog` and the sidebar's tiles are drawn with, so a part
 * that grows or shrinks inside one animates without asking. A single line that
 * comes and goes still uses `Grow`, which also fades it out; the box follows it.
 */
export function Smooth({ className = '', inner = '', ref, children, ...rest }: Props) {
  const outer = useRef<HTMLDivElement | null>(null)
  const content = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!outer.current || !content.current) return
    return smoothResize(outer.current, content.current)
  }, [])
  const setOuter = (el: HTMLDivElement | null) => {
    outer.current = el
    if (typeof ref === 'function') ref(el)
    else if (ref) ref.current = el
  }
  return (
    <div ref={setOuter} className={className} {...rest}>
      <div ref={content} className={inner}>
        {children}
      </div>
    </div>
  )
}
