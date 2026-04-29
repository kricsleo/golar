import { createVolarPlugin } from '@golar/volar'
import { Debug, ms } from '@golar/util'
import { enableCompileCache } from 'node:module'
enableCompileCache()

const debug = Debug.create('plugin:vue')

export type VuePluginConfigureOptions = {
	vitePressExtensions?: string[] | undefined
}

export function configure(opts: VuePluginConfigureOptions) {
	createVolarPlugin({
		filename: import.meta.filename,
		extensions: ['.vue', ...(opts.vitePressExtensions ?? [])].map(
			(extension) => ({
				extension,
				stripFromDeclarationFileName: false,
				allowExtensionlessImports: false,
			}),
		),
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
}

configure({})
