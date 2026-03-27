import path from 'node:path'
import { defineConfig, defineNativeRule, rules } from 'golar/unstable'
import '@golar/vue'
import { jsRule } from './js-rule.ts'

const addonFilename =
	process.platform === 'win32'
		? 'e2e_lint_typecheck_rust_addon.dll'
		: process.platform === 'darwin'
			? 'libe2e_lint_typecheck_rust_addon.dylib'
			: 'libe2e_lint_typecheck_rust_addon.so'
const addonPath = path.join(
	import.meta.dirname,
	'..',
	'..',
	'..',
	'target',
	'debug',
	addonFilename,
)

export default defineConfig({
	lint: {
		use: [
			{
				files: ['index.ts', 'index.vue'],
				rules: [
					jsRule,
					defineNativeRule({
						addonPath,
						name: 'rust/unsafe-calls',
					}),
					...rules({
						'explicit-anys': true,
					}),
				],
			},
		],
	},
	typecheck: {
		include: ['index.ts', 'index.vue'],
	},
})
