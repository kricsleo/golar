import { createVolarPlugin } from '@golar/volar'
import { Debug, ms } from '@golar/util'

const debug = Debug.create('plugin:vue')

createVolarPlugin({
	filename: import.meta.filename,
	extraFileExtensions: ['.vue'],
	languagePlugins: async (cwd, configFileName) => {
		const started = performance.now()
		await import('./patch-language-tools.ts')
		const patched = performance.now()
		const { vueLanguagePlugin } = await import('./language-plugin.ts')
		debug.printf(
			'loaded language plugin +%s (patching took +%s)',
			ms(performance.now() - started),
			ms(patched - started),
		)

		return [vueLanguagePlugin(cwd, configFileName)]
	},
})
