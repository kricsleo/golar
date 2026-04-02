import url from 'node:url'
import util from 'node:util'
import { globSync } from 'tinyglobby'
import { Debug } from '@golar/util'
import type { GolarBrand } from '../../../internal/linter/rule-creator.ts'
import { globalState } from './codegen-plugin.ts'

const debug = Debug.create('config')

export type LintConfiguredRule = {
	[GolarBrand]: {
		rule: true
	}
}
export type LintConfigUseDefinition = {
	// TODO: ignore: string[]
	files: string[]
	rules: LintConfiguredRule[]
}

export type Config = {
	lint?:
		| {
				use: LintConfigUseDefinition[]
		  }
		| undefined
	typecheck?: {
		include: string[]
		exclude?: string[] | undefined
	}
}

export function defineConfig(config: Config) {
	return config
}

export async function loadConfig(configPath: string) {
	const { default: config } = (await import(
		url.pathToFileURL(configPath).href
	)) as { default: Config }

	return config
}

export function resolveConfig(cwd: string, config: Config) {
	const allFiles = new Set<string>()
	const builtinEffectiveRulesByFile = new Map<
		string,
		Map<any, LintConfiguredRule>
	>()
	const jsEffectiveRulesByFile = new Map<string, Map<any, LintConfiguredRule>>()
	const nativeEffectiveRulesByFile = new Map<
		string,
		Map<any, LintConfiguredRule>
	>()

	for (const group of config.lint?.use ?? []) {
		// TODO: parallelize?
		for (const file of globSync(group.files, { cwd, absolute: true })) {
			allFiles.add(file)

			for (const rule of group.rules) {
				let rulesByFile: Map<string, Map<any, LintConfiguredRule>>
				// @ts-expect-error
				if (rule.rule.isBuiltin) {
					rulesByFile = builtinEffectiveRulesByFile
					// @ts-expect-error
				} else if (rule.rule.isCustomJs) {
					rulesByFile = jsEffectiveRulesByFile
					// @ts-expect-error
				} else if (rule.rule.isNative) {
					rulesByFile = nativeEffectiveRulesByFile
				} else {
					throw new Error(`Invalid rule: ${util.inspect(rule)}`)
				}
				let effectiveRules = rulesByFile.get(file)
				if (effectiveRules == null) {
					rulesByFile.set(file, (effectiveRules = new Map()))
				}

				// @ts-expect-error
				effectiveRules.set(rule.rule, rule)
			}
		}
	}

	// TODO: dedupe by options hash?

	const builtinRulesByFile = new Map<string, LintConfiguredRule[]>(
		builtinEffectiveRulesByFile
			.entries()
			.map(([file, rulesById]) => [file, Array.from(rulesById.values())]),
	)

	const jsFiles = Array.from(jsEffectiveRulesByFile.keys())
	const jsWorkersCount = Math.min(jsFiles.length, 4)
	const jsFilesPerThread = Math.ceil(jsFiles.length / jsWorkersCount)

	const jsEffectiveRulesIter = jsEffectiveRulesByFile.keys()
	const jsFilesByWorker: string[][] = Array(jsWorkersCount)
		.fill(null)
		.map(() => Array.from(jsEffectiveRulesIter.take(jsFilesPerThread)))

	const nativeRulesByFile = new Map<string, LintConfiguredRule[]>(
		nativeEffectiveRulesByFile
			.entries()
			.map(([file, rulesById]) => [file, Array.from(rulesById.values())]),
	)

	let typecheckFiles: string[] | null = null
	const typecheckInclude =
		config.typecheck == null
			? [
					'**/*.{ts,tsx}',
					...globalState.codegenPlugins
						.values()
						.flatMap((plugin) => plugin.extensions)
						.map(({ extension }) => `**/*${extension}`)
						.toArray(),
				]
			: config.typecheck.include
	// TODO: support .gitignore
	const typecheckExclude = [
		...(config.typecheck?.exclude ?? []),
		'**/node_modules',
		'**/.git',
		'**/.jj',
	]

	typecheckFiles = globSync(typecheckInclude, {
		cwd,
		ignore: typecheckExclude,
		absolute: true,
	})
	for (const file of typecheckFiles) {
		allFiles.add(file)
	}

	debug.print('typecheck files:\n' + typecheckFiles.join('\n'))

	return {
		files: Array.from(allFiles),
		builtinRulesByFile,
		jsFilesByWorker,
		jsRulesByFile: jsEffectiveRulesByFile,
		nativeRulesByFile,
		typecheckFiles,
	}
}
