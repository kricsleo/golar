import { createVolarPlugin } from '@golar/volar'
import { Debug, ms } from '@golar/util'
import { enableCompileCache } from 'node:module'
enableCompileCache()

const debug = Debug.create('plugin:ember')

createVolarPlugin({
	filename: import.meta.filename,
	extensions: [
		{
			extension: '.gts',
			// https://github.com/typed-ember/glint/issues/988
			// https://github.com/typed-ember/glint/blob/cecd7a10bf83cf71e759634bc0d88829c668fa0f/packages/core/src/cli/run-volar-tsc.ts#L14-L21
			stripFromDeclarationFileName: true,
			allowExtensionlessImports: true,
		},
		{
			extension: '.gjs',
			stripFromDeclarationFileName: true,
			allowExtensionlessImports: true,
		},
	],
	languagePlugins: async (cwd) => {
		const started = performance.now()
		const { createEmberLanguagePlugin, findConfig } =
			await import('@glint/ember-tsc')
		debug.printf('loaded language plugin +%s', ms(performance.now() - started))

		const config = findConfig(cwd)

		if (config == null) {
			return []
		}

		return [createEmberLanguagePlugin(config)]
	},
})
