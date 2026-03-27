import { JsCodegenPlugin, type ServiceCodeError } from 'golar/unstable'
import { createRequire } from 'node:module'
import path from 'node:path'
import process from 'node:process'
import util from 'node:util'
import { sourceMapToMappings } from '@golar/sourcemap'
import { pathToFileURL } from 'node:url'

const require = createRequire(process.cwd())

const svelteTsxFilesByProject = new Map<string, string[]>()

let packages: {
	svelte2tsx: typeof import('svelte2tsx')
	svelte2tsxPath: string
	svelteMajorVersion: number
	sveltePath: string
	svelteCompiler: typeof import('svelte/compiler')
	ts: typeof import('typescript')
} | null = null

async function importPackages(
	cwd: string,
	configFileName: string | null,
): Promise<NonNullable<typeof packages>> {
	if (packages != null) {
		return packages
	}
	const resolvePaths: string[] = []
	if (configFileName != null) {
		resolvePaths.push(path.dirname(configFileName))
	}
	resolvePaths.push(cwd)
	// TODO: error message if svelte is not installed
	const sveltePackageJsonPath = require.resolve('svelte/package.json', {
		paths: resolvePaths,
	})
	const { default: svelteCompilerPackageJson } = await import(
		pathToFileURL(sveltePackageJsonPath).toString(),
		{ with: { type: 'json' } }
	)
	const majorVersion = Number.parseInt(
		svelteCompilerPackageJson.version.split('.')[0]!,
	)
	const svelte2tsxPath = require.resolve('svelte2tsx', {
		paths: [...resolvePaths, import.meta.dirname],
	})
	const [svelteCompiler, svelte2tsx, ts] = await Promise.all([
		// Copied from packages/language-server/src/plugins/typescript/service.ts
		// Svelte 4 has some fixes with regards to parsing the generics attribute.
		// Svelte 5 has new features, but we don't want to add the new compiler into language-tools. In the future it's probably
		// best to shift more and more of this into user's node_modules for better handling of multiple Svelte versions.
		majorVersion >= 4
			? import(
					pathToFileURL(
						require.resolve('svelte/compiler', {
							paths: [...resolvePaths, import.meta.dirname],
						}),
					).toString()
				)
			: undefined,
		import(pathToFileURL(svelte2tsxPath).toString()),
		import('typescript'),
	])

	return (packages = {
		svelte2tsx,
		svelte2tsxPath: path.dirname(svelte2tsxPath),
		svelteMajorVersion: majorVersion,
		sveltePath: path.dirname(sveltePackageJsonPath),
		svelteCompiler,
		ts,
	})
}

new JsCodegenPlugin({
	id: 'svelte',
	extensions: [
		{
			extension: '.svelte',
			stripFromDeclarationFileName: false,
			allowExtensionlessImports: false,
		},
	],
	async createServiceCode(cwd, configFileName, fileName, sourceText) {
		const imported = await importPackages(cwd, configFileName).catch((e) =>
			util.inspect(e),
		)
		if (typeof imported === 'string') {
			return {
				errors: [
					{
						start: 0,
						end: 0,
						message: imported,
					},
				],
			}
		}
		const {
			svelte2tsx,
			svelte2tsxPath,
			svelteMajorVersion,
			sveltePath,
			svelteCompiler,
			ts,
		} = imported

		const project = `${cwd}::${configFileName}`
		let svelteTsxFiles = svelteTsxFilesByProject.get(project)
		if (svelteTsxFiles == null) {
			svelteTsxFilesByProject.set(
				project,
				(svelteTsxFiles = svelte2tsx.internalHelpers.get_global_types(
					ts.sys,
					svelteMajorVersion === 3,
					sveltePath,
					svelte2tsxPath,
					configFileName ?? cwd,
				)),
			)
		}

		try {
			const tsx = svelte2tsx.svelte2tsx(sourceText, {
				isTsFile: true,
				mode: 'ts',
				parse: svelteCompiler.parse,
			})

			return {
				serviceText:
					tsx.code +
					'\n\n' +
					svelteTsxFiles.map((p) => `import ${JSON.stringify(p)}`).join('\n'),
				scriptKind: 'tsx',
				declarationFile: fileName.includes('/node_modules/'),
				mappings: sourceMapToMappings({
					sourceText,
					serviceText: tsx.code,
					sourceMap: tsx.map.mappings,
				}),
			}
		} catch (e) {
			const error: ServiceCodeError = {
				message: '',
				start: 0,
				end: 0,
			}
			if (typeof e === 'object' && e != null) {
				if ('message' in e && 'name' in e) {
					error.message = `${e.name}: ${e.message}`
				} else {
					error.message = util.inspect(e)
				}
				if ('position' in e && Array.isArray(e.position)) {
					error.start = e.position[0] ?? 0
					error.end = e.position[1] ?? 0
				}
			}
			return {
				errors: [error],
			}
		}
	},
})
