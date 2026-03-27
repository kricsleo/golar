import { z } from 'zod'
import { ruleTesterLint } from './linter.ts'
import assert from 'node:assert/strict'

export type TesterSetupDescribe = (
	description: string,
	setup: () => void,
) => void

export type TesterSetupIt = ((
	description: string,
	setup: () => Promise<void> | void,
) => void) & {
	only: (description: string, setup: () => Promise<void> | void) => void
}

export interface TesterOptions {
	it: TesterSetupIt
	describe: TesterSetupDescribe
	defaults: {
		filename: string
		files?: Record<string, string>
	}
}

type TesterCase = {
	code: string
	filename?: string
	files?: Record<string, string>
	only?: boolean
	// TODO:
	options?: unknown
}

type TesterInvalidCase = TesterCase & {
	snapshot: string
	output?: string
	suggestions?: Array<{
		message: string
		output: string
	}>
}

export class Tester {
	readonly opts: TesterOptions

	constructor(opts: TesterOptions) {
		this.opts = opts
	}

	describe(
		rule: any,
		opts: {
			valid: (string | TesterCase)[]
			invalid: TesterInvalidCase[]
		},
	) {
		this.opts.describe(rule.name, () => {
			const hasOnly =
				opts.valid.some((c) => typeof c !== 'string' && c.only) ||
				opts.invalid.some(({ only }) => only)
			this.opts.describe('valid', () => {
				for (const valid of opts.valid) {
					const testCase = typeof valid === 'string' ? { code: valid } : valid
					const { code, options } = testCase
					const filename = testCase.filename ?? this.opts.defaults.filename
					const files = {
						...(this.opts.defaults.files ?? {}),
						...(testCase.files ?? {}),
						[filename]: code,
					}
					;(hasOnly && typeof valid !== 'string' && valid.only
						? this.opts.it.only
						: this.opts.it)(code, () => {
						const opts = z.object(rule.options).parse(options ?? {})
						const res = ruleTesterLint(files, filename, rule.name, opts)
						assert.equal(
							res.snapshot,
							code,
							'Expected snapshot to match input code',
						)
						assert.equal(res.output, undefined, 'Expected no autofix output')
						assert.equal(res.suggestions, undefined, 'Expected no suggestions')
					})
				}
			})

			this.opts.describe('invalid', () => {
				for (const invalid of opts.invalid) {
					;(hasOnly && invalid.only ? this.opts.it.only : this.opts.it)(
						invalid.code,
						() => {
							const filename = invalid.filename ?? this.opts.defaults.filename
							const files = {
								...(this.opts.defaults.files ?? {}),
								...(invalid.files ?? {}),
								[filename]: invalid.code,
							}
							const opts = z.object(rule.options).parse(invalid.options ?? {})
							const res = ruleTesterLint(files, filename, rule.name, opts)
							assert.equal(
								res.snapshot,
								invalid.snapshot,
								'Expected snapshot to match expected snapshot',
							)
							assert.equal(
								res.output,
								invalid.output,
								'Expected autofix output to match expected output',
							)
							assert.deepEqual(
								res.suggestions,
								invalid.suggestions,
								'Expected suggestions to match expected suggestions',
							)
						},
					)
				}
			})
		})
	}
}
