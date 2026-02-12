import { describe, expect, it, vi } from 'vitest'

import { createDebugFromString } from './debug.ts'

function captureStderr(fn: () => void): string {
	let stderr = ''
	const spy = vi.spyOn(process.stderr, 'write').mockImplementation(((
		chunk: unknown,
	) => {
		stderr += String(chunk)
		return true
	}) as never)

	try {
		fn()
		return stderr
	} finally {
		spy.mockRestore()
	}
}

describe('logger', () => {
	it('defaults to disabled for empty debug', () => {
		const output = captureStderr(() => {
			const l = createDebugFromString('', 'test:ns')
			l.print('hello')
			l.printf('value=%d', 7)
		})
		expect(output).toBe('')
	})

	it('matches and skips', () => {
		const output = captureStderr(() => {
			createDebugFromString(
				'golar:api:*,-golar:api:internal',
				'api:users',
			).print('m')
			createDebugFromString(
				'golar:api:*,-golar:api:internal',
				'api:internal',
			).print('s')
			createDebugFromString(
				'golar:api:*,-golar:api:internal',
				'worker:queue',
			).print('u')
		})
		expect(output).toBe('golar:api:users m\ngolar:worker:queue u\n')
	})

	it('leading skip then match', () => {
		const output = captureStderr(() => {
			createDebugFromString('-golar:api,golar:api*', 'api').print('exact')
			createDebugFromString('-golar:api,golar:api*', 'api:users').print('child')
		})
		expect(output).toBe('golar:api:users child\n')
	})

	it('skips empty parts', () => {
		const output = captureStderr(() => {
			createDebugFromString(',,golar:api,,', 'api').print('a')
			createDebugFromString(',,golar:api,,', 'worker').print('w')
		})
		expect(output).toBe('golar:api a\ngolar:worker w\n')
	})

	it('skips bare dash part', () => {
		const output = captureStderr(() => {
			createDebugFromString('-,golar:api', 'api').print('a')
			createDebugFromString('-,golar:api', 'worker').print('w')
		})
		expect(output).toBe('golar:api a\ngolar:worker w\n')
	})

	it('adds alternation for multiple parts', () => {
		const output = captureStderr(() => {
			createDebugFromString(
				'golar:api,golar:worker,-golar:internal,-golar:other',
				'api',
			).print('a')
			createDebugFromString(
				'golar:api,golar:worker,-golar:internal,-golar:other',
				'worker',
			).print('w')
			createDebugFromString(
				'golar:api,golar:worker,-golar:internal,-golar:other',
				'internal',
			).print('i')
			createDebugFromString(
				'golar:api,golar:worker,-golar:internal,-golar:other',
				'other',
			).print('o')
		})
		expect(output).toBe('golar:api a\ngolar:worker w\n')
	})

	it('skip only leaves logger disabled', () => {
		const output = captureStderr(() => {
			createDebugFromString('-api,-worker', 'api').print('a')
			createDebugFromString('-api,-worker', 'worker').print('w')
		})
		expect(output).toBe('')
	})

	it('wildcard match with wildcard skip', () => {
		const output = captureStderr(() => {
			createDebugFromString(
				'golar:service:*,-golar:service:db*',
				'service:web',
			).print('web')
			createDebugFromString(
				'golar:service:*,-golar:service:db*',
				'service:db:read',
			).print('read')
			createDebugFromString(
				'golar:service:*,-golar:service:db*',
				'service:db:write',
			).print('write')
		})
		expect(output).toBe('golar:service:web web\n')
	})

	it('print and printf', () => {
		const output = captureStderr(() => {
			const l = createDebugFromString('test:*', 'test:ns')
			l.print('hello', 'world')
			l.printf('value=%d', 7)
		})
		expect(output).toBe('golar:test:ns hello world\ngolar:test:ns value=7\n')
	})

	it('print and printf when disabled', () => {
		const output = captureStderr(() => {
			const l = createDebugFromString('', 'test:ns')
			l.print('hello')
			l.printf('value=%d', 7)
		})
		expect(output).toBe('')
	})
})
