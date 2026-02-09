import { createVolarPlugin } from '@golar/volar'
await import('./patch-language-tools.ts')
const { vueLanguagePlugin } = await import('./language-plugin.ts')

createVolarPlugin({
	filename: import.meta.filename,
	extraFileExtensions: ['.vue'],
	languagePlugins: async (cwd, configFileName) => [
		await vueLanguagePlugin(cwd, configFileName)
	]
})
