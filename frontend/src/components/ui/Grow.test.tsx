import { afterEach, describe, expect, it } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { Grow } from './Grow'

const line = (shown: boolean) => <Grow>{shown && <p>Online servers refuse it.</p>}</Grow>

describe('Grow', () => {
  afterEach(cleanup)

  it('is open with what it holds, and folded with nothing', () => {
    const { container, rerender } = render(line(true))
    const fold = container.firstElementChild!
    expect(fold).toHaveClass('grid-rows-[1fr]')
    expect(fold).not.toHaveAttribute('inert')
    rerender(line(false))
    expect(fold).toHaveClass('grid-rows-[0fr]')
    expect(fold).toHaveAttribute('inert')
    expect(fold).toHaveAttribute('aria-hidden', 'true')
  })

  it('keeps the last content while it folds, and lets it go once folded', () => {
    const { container, rerender } = render(line(true))
    rerender(line(false))
    expect(screen.getByText('Online servers refuse it.')).toBeInTheDocument()
    fireEvent.transitionEnd(container.firstElementChild!)
    expect(screen.queryByText('Online servers refuse it.')).toBeNull()
  })

  it('takes an empty string for nothing', () => {
    const { container } = render(<Grow>{''}</Grow>)
    expect(container.firstElementChild).toHaveClass('grid-rows-[0fr]')
  })
})
