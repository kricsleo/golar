#!/usr/bin/env node

import { createVolarPlugin } from '@golar/volar'

createVolarPlugin({
	filename: import.meta.filename,
	extraFileExtensions: ['.vue'],
	languagePlugins: async (cwd, configFileName) => {
		await import('./patch-language-tools.ts')
		const { vueLanguagePlugin } = await import('./language-plugin.ts')
		return [
			await vueLanguagePlugin(cwd, configFileName)
		]
	}
})
