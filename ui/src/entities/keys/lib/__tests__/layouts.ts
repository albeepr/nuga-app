import { describe, expect, test } from 'vitest'

import type { KeyboardLayout } from '../../model/types'
import { Air96V2, Halo96 } from '../layouts'

function rowWidths (layout: KeyboardLayout): number[] {
  return layout.map((row) => row.reduce((sum, k) => sum + (k.width ?? 1), 0))
}

describe('Air96V2 layout', () => {
  test('has one key per VIA matrix position', () => {
    expect(Air96V2.flat().filter((k) => k.code).length).toBe(100)
  })

  test('matches 96% row geometry', () => {
    expect(rowWidths(Air96V2)).toStrictEqual(rowWidths(Halo96))
  })
})
