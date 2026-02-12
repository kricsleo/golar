import { describe, expect, it } from 'vitest'

import { ms } from './ms.ts'

describe('ms', () => {
	it('formats values up to 1ms as microseconds', () => {
		expect(ms(0)).toBe('0µs')
		expect(ms(0.123)).toBe('123µs')
		expect(ms(1)).toBe('1000µs')
	})

	it('formats values greater than 1ms as ms+µs', () => {
		expect(ms(1.234)).toBe('1ms23µs')
		expect(ms(9.999)).toBe('10ms0µs')
		expect(ms(10)).toBe('10ms0µs')
	})

	it('formats values greater than 10ms as 3-significant-digit ms', () => {
		expect(ms(10.001)).toBe('10ms')
		expect(ms(12.345)).toBe('12.3ms')
		expect(ms(999.9)).toBe('1000ms')
	})

	it('formats values greater than 1s as s+ms', () => {
		expect(ms(1000)).toBe('1000ms')
		expect(ms(1000.1)).toBe('1s0ms')
		expect(ms(1234)).toBe('1s23ms')
		expect(ms(1999)).toBe('2s0ms')
	})

	it('formats values greater than 1m as m+s', () => {
		expect(ms(60_000)).toBe('60s0ms')
		expect(ms(60_001)).toBe('1m0s')
		expect(ms(61_234)).toBe('1m1s')
		expect(ms(61_600)).toBe('1m2s')
		expect(ms(119_950)).toBe('2m0s')
	})

	it('always formats absolute value without sign', () => {
		expect(ms(-0.5)).toBe('500µs')
		expect(ms(-1234)).toBe('1s23ms')
		expect(ms(-61_234)).toBe('1m1s')
	})
})
