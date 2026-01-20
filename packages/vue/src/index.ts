import { createVolarPlugin } from '@golar/volar'

createVolarPlugin({
	filename: import.meta.filename,
	async getLanguagePlugins() {
		const ts = await import('typescript')
		const { createVueLanguagePlugin, createParsedCommandLineByJson } = await import('@vue/language-core')
		return [
			createVueLanguagePlugin<string>(
				await import('typescript'),
				{},
				createParsedCommandLineByJson(ts, ts.sys, ts.sys.getCurrentDirectory(), {}).vueOptions,
				scriptId => scriptId
			)
		]
	},
})
