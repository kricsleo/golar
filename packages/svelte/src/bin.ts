import { svelte2tsx, internalHelpers } from 'svelte2tsx'
import { createPlugin, type ServiceCodeError } from '@golar/plugin'
import { createRequire } from 'node:module'
import path from 'node:path'
import process from 'node:process'
import util from 'node:util'
import { sourceMapToMappings } from '@golar/sourcemap'

const require = createRequire(process.cwd())

const svelteTsxFilesByProject = new Map<string, Promise<string[]>>()

async function importSvelteTsxFiles(
	cwd: string,
	configFileName: string | null,
): Promise<string[]> {
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
		sveltePackageJsonPath,
		{ with: { type: 'json' } }
	)
	const majorVersion = Number.parseInt(
		svelteCompilerPackageJson.version.split('.')[0]!,
	)
	// Copied from packages/language-server/src/plugins/typescript/service.ts
	// Svelte 4 has some fixes with regards to parsing the generics attribute.
	// Svelte 5 has new features, but we don't want to add the new compiler into language-tools. In the future it's probably
	// best to shift more and more of this into user's node_modules for better handling of multiple Svelte versions.
	// const svelteCompiler =
	//     majorVersion >= 4
	//         ? await import(require.resolve('svelte/compiler'))
	//         : undefined;

	const ts = await import('typescript')
	return internalHelpers.get_global_types(
		ts.sys,
		majorVersion === 3,
		path.dirname(sveltePackageJsonPath),
		path.dirname(
			require.resolve('svelte2tsx', {
				paths: [...resolvePaths, import.meta.dirname],
			}),
		),
		configFileName ?? cwd,
	)
}

createPlugin({
	filename: import.meta.filename,
	extraExtensions: ['.svelte'],
	async createServiceCode(cwd, configFileName, fileName, sourceText) {
		const project = `${cwd}::${configFileName}`
		let svelteTsxFiles: string[]
		{
			let files = svelteTsxFilesByProject.get(project)
			if (files == null) {
				svelteTsxFilesByProject.set(
					project,
					(files = importSvelteTsxFiles(cwd, configFileName)),
				)
			}
			svelteTsxFiles = await files
		}

		try {
			const tsx = svelte2tsx(sourceText, {
				isTsFile: true,
				mode: 'ts',
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
