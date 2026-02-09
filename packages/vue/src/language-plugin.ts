import { createParsedCommandLine, forEachEmbeddedCode } from '@vue/language-core'
import * as ts from './typescript-lite.js'
import compilerDom from '@vue/compiler-dom'
import { createParsedCommandLineByJson } from '@vue/language-core'
import { VueVirtualCode } from '@vue/language-core/lib/virtualCode/index.js'
import PluginVueTsx from '@vue/language-core/lib/plugins/vue-tsx.js'
import PluginFileVue from '@vue/language-core/lib/plugins/file-vue.js'
import PluginVueScriptJs from '@vue/language-core/lib/plugins/vue-script-js.js'
import PluginVueTemplateHtml from '@vue/language-core/lib/plugins/vue-template-html.js'
import type { VolarLanguagePlugin } from '@golar/volar'

export async function vueLanguagePlugin(cwd: string, configFileName: string | null): Promise<VolarLanguagePlugin> {
	const { options: compilerOptions, vueOptions: vueCompilerOptions } = configFileName == null
		? createParsedCommandLineByJson(ts, ts.sys, cwd, {})
		: createParsedCommandLine(ts, ts.sys, configFileName)
	const plugins = (await Promise.all([
		PluginVueTsx,
		PluginFileVue,
		PluginVueScriptJs,
		PluginVueTemplateHtml,
	])).flatMap(({ default: ctor }) => ctor({
		modules: {
			typescript: ts,
			"@vue/compiler-dom": compilerDom
		},
		compilerOptions,
		vueCompilerOptions,
	}))
	return {
		getLanguageId(scriptId) {
		  return scriptId.endsWith('.vue') ? 'vue' : undefined
		},
		createVirtualCode(scriptId, languageId, snapshot) {
			return new VueVirtualCode(
				scriptId,
				languageId,
				snapshot,
				vueCompilerOptions,
				plugins,
				ts,
			);
		},
		getVirtualCodeErrors(root) {
			return (root as VueVirtualCode).vueSfc!.errors
				.filter(e => 'code' in e)
				.map(e => ({
					start: e.loc?.start.offset ?? 0,
					end: e.loc?.end.offset ?? 0,
					message: e.message,
				}))
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
			},
		}
	}
}
