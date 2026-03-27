import { it, describe } from 'vitest'
import { Tester } from '../rule-tester.ts'

export function createRuleTesterTSConfig(
	defaultCompilerOptions?: Record<string, unknown>,
) {
	return {
		'/tsconfig.base.json': JSON.stringify(
			{
				compilerOptions: {
					lib: ['esnext'],
					moduleResolution: 'bundler',
					strict: true,
					target: 'esnext',
					types: [],
					...defaultCompilerOptions,
				},
			},
			null,
			2,
		),
		'/tsconfig.json': JSON.stringify(
			{ extends: './tsconfig.base.json' },
			null,
			2,
		),
	}
}

export const ruleTester = new Tester({
	describe,
	it,
	defaults: {
		filename: '/file.ts',
		files: createRuleTesterTSConfig(),
	},
})

export const domLibRuleTester = new Tester({
	describe,
	it,
	defaults: {
		filename: '/file.ts',
		files: createRuleTesterTSConfig({
			lib: ['esnext', 'dom'],
		}),
	},
})
