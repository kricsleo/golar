import { createVolarPlugin } from '@golar/volar'
import { forEachEmbeddedCode } from '@vue/language-core'

createVolarPlugin({
	filename: import.meta.filename,
	async getLanguagePlugins() {
		const ts = await import('./typescript-lite.js')
		const compilerDom = await import('@vue/compiler-dom')
		const { createParsedCommandLineByJson } = await import('@vue/language-core')
		const { vueOptions } = createParsedCommandLineByJson(ts, ts.sys, ts.sys.getCurrentDirectory(), {})
		const plugins = (await Promise.all([
			import('@vue/language-core/lib/plugins/vue-tsx.js'),
			import('@vue/language-core/lib/plugins/file-vue.js'),
			import('@vue/language-core/lib/plugins/vue-script-js.js'),
			import('@vue/language-core/lib/plugins/vue-template-html.js'),
		])).flatMap(({default:{default: ctor}}) => ctor({
			modules: {
				typescript: ts,
				"@vue/compiler-dom": compilerDom
			},
			compilerOptions: {},
			vueCompilerOptions: vueOptions,
		}))
		const { VueVirtualCode } = await import('@vue/language-core/lib/virtualCode/index.js')
		return [
			{
				getLanguageId(scriptId) {
				  return scriptId.endsWith('.vue') ? 'vue' : undefined
				},
				createVirtualCode(scriptId, languageId, snapshot) {
					return new VueVirtualCode(
						scriptId,
						languageId,
						snapshot,
						vueOptions,
						plugins,
						ts,
					);
				},
				typescript: {
					extraFileExtensions: [{
						extension: 'vue',
						isMixedContent: true,
						scriptKind: 7 satisfies import('typescript').ScriptKind.Deferred,
					}],
					getServiceScript(root) {
						for (const code of forEachEmbeddedCode(root)) {
							if (/script_(js|jsx|ts|tsx)/.test(code.id)) {
								const lang = code.id.slice('script_'.length);
								return {
									code,
									extension: '.' + lang,
									scriptKind: lang === 'js'
										? ts.ScriptKind.JS
										: lang === 'jsx'
										? ts.ScriptKind.JSX
										: lang === 'tsx'
										? ts.ScriptKind.TSX
										: ts.ScriptKind.TS,
								};
							}
						}
						return undefined
					}
				}
			},
		]
	},
})
