import { createContext, useContext } from 'react'

/**
 * Whether the page around a component has finished sliding in (components/main/PageLayer
 * provides it). A page whose parts read slowly starts the reads at once and holds back what
 * they would add to the page until this is true, so the page does not grow while it moves.
 * Outside a layer, and for any page that never slides, there is nothing to wait for: true.
 */
export const PageSettledContext = createContext(true)

export const usePageSettled = (): boolean => useContext(PageSettledContext)
