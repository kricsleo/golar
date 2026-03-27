import fs from 'node:fs/promises'
import path from 'node:path'
import packageJson from '../package.json' with { type: 'json' }
import process from 'node:process'
import { GOARCH2PROCESS_ARCH, GOOS2PROCESS_PLATFORM } from './utils.ts'

const copyBinaries = process.argv[2] === 'copy-binaries'

const binariesMatrix = Object.entries(GOOS2PROCESS_PLATFORM).flatMap(
	([goos, platform]) =>
		Object.entries(GOARCH2PROCESS_ARCH)
			// windows-arm64 build is broken (see release.yml)
			.filter(([goarch]) => goos !== 'windows' || goarch !== 'arm64')
			.map(([goarch, arch]) => ({
				goarch,
				goos,
				arch,
				platform,
			})),
)

const commonPackageJson = {
	version: packageJson.version,
	type: 'module',
	license: 'MIT',
	author: 'auvred <aauvred@gmail.com> (https://github.com/auvred)',
	repository: 'github:auvred/golar',
	bugs: 'https://github.com/auvred/golar/issues',
	homepage: 'https://github.com/auvred/golar#readme',
	publishConfig: {
		access: 'public',
	},
} as const

const repoRoot = path.join(import.meta.dirname, '..')
const buildDir = path.join(repoRoot, 'build')

const npmDir = path.join(repoRoot, 'generated-packages')

await fs.rm(npmDir, { recursive: true, force: true })

await genPackageWithBinary(
	'@golar/',
	({ goos, goarch }) =>
		path.join(buildDir, `golar-${goos}-${goarch}`, 'golar.node'),
	false,
)
await genPackageWithBinary(
	'@golar/astro',
	({ goos, goarch, platform }) =>
		path.join(
			buildDir,
			`golar-${goos}-${goarch}`,
			`golar-astro${platform === 'win32' ? '.exe' : ''}`,
		),
	true,
)

async function genPackageWithBinary(
	npmPackageName: string,
	binaryPathFn: (pl: (typeof binariesMatrix)[number]) => string,
	executable: boolean,
) {
	await Promise.all([
		...binariesMatrix.map(async ({ goarch, goos, arch, platform }) => {
			const npmPackageNamePlatform = `${npmPackageName}${npmPackageName.endsWith('/') ? '' : '-'}${platform}-${arch}`
			const packageDir = path.join(
				npmDir,
				npmPackageNamePlatform.replaceAll('/', '-'),
			)
			const binaryPath = binaryPathFn({ goarch, goos, arch, platform })
			const binaryName = `./${path.basename(binaryPath)}`

			await fs.mkdir(packageDir, { recursive: true })
			await Promise.all([
				fs.writeFile(
					path.join(packageDir, 'package.json'),
					JSON.stringify(
						{
							...commonPackageJson,
							publishConfig: {
								...commonPackageJson.publishConfig,
								...(executable && { executableFiles: [binaryName] }),
							},
							name: npmPackageNamePlatform,
							preferUnplugged: true,
							files: [binaryName],
							os: [platform],
							cpu: [arch],
						},
						null,
						2,
					),
				),
				copyBinaries &&
					fs.copyFile(binaryPath, path.join(packageDir, binaryName)),
			])
		}),
	])
}
