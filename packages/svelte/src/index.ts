import { svelte2tsx } from 'svelte2tsx'
import { createPlugin } from '@golar/plugin'
import util from 'node:util'

createPlugin({
	filename: import.meta.filename,
	createServiceCode(fileName, sourceText) {
		// TODO: handle parsing errors
		// TODO: .d.ts references
		const tsx = svelte2tsx(sourceText, {
			isTsFile: true,
			mode: "ts",
		});

		return {
			serviceText: tsx.code,
			sourceMap: tsx.map.mappings,
		}
	},
})
