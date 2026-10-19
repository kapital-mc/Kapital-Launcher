import { afterEach, describe, expect, it } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { Fact, FactList } from './Fact'
import { PLACEHOLDER } from '../../lib/manifest'

describe('Fact', () => {
  afterEach(cleanup)

  it('sets a value in the data face', () => {
    render(
      <FactList>
        <Fact label="Minecraft" value="1.21.1" />
      </FactList>,
    )
    expect(screen.getByText('1.21.1')).toHaveClass('font-mono')
  })

  it('sets the values at the right edge when spread', () => {
    const { container } = render(<FactList facts={[{ label: 'Mods', value: '98' }]} spread />)
    expect(container.querySelector('dl')).toHaveClass('[&_dd]:text-right')
  })

  it('sets a word in the UI face when it is plain', () => {
    render(
      <FactList>
        <Fact label="Loader" value="Client visuals" plain />
      </FactList>,
    )
    expect(screen.getByText('Client visuals')).not.toHaveClass('font-mono')
  })

  it('says Unknown, faint, in place of the marker', () => {
    render(
      <FactList>
        <Fact label="Mods" value={PLACEHOLDER} />
      </FactList>,
    )
    const value = screen.getByText('Unknown')
    expect(value).toHaveClass('text-fg-faint')
    expect(value).not.toHaveClass('font-mono')
    expect(screen.queryByText(PLACEHOLDER)).toBeNull()
  })

  it('says what it is told in place of the marker', () => {
    render(
      <FactList>
        <Fact label="Size" value={PLACEHOLDER} unset="Not installed" />
      </FactList>,
    )
    expect(screen.getByText('Not installed')).toHaveClass('text-fg-faint')
  })
})
