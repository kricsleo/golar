import assert from 'node:assert'
import os from 'node:os'
import process from 'node:process'
import { golarAddonPath } from '../../packages/golar/src/addon.ts'

const addon = {
	exports: {} as {
		linter_RuleTesterLint(
			files: string,
			fileName: string,
			ruleName: string,
			options: string,
		): string
	},
}
process.dlopen(
	addon,
	golarAddonPath,
	os.constants.dlopen.RTLD_NOW,
)

export function ruleTesterLint(
	files: Record<string, string>,
	fileName: string,
	ruleName: string,
	options: unknown,
) {
	assert.ok(
		files[fileName] != null,
		`${fileName} is not found in ${Object.keys(files).join(', ')}`,
	)

	const result = JSON.parse(
		addon.exports.linter_RuleTesterLint(
			JSON.stringify(files),
			fileName,
			ruleName,
			JSON.stringify(options),
		),
	) as {
		snapshot: string
		output?: string
		suggestions?: Array<{
			message: string
			output: string
		}>
	}

	return result
}
