import fs from 'node:fs/promises'
import path from 'node:path'
import packageJson from '../package.json' with { type: 'json' }
import process from 'node:process'

const copyBinaries = process.argv[2] === 'copy-binaries'

const GOOS2PROCESS_PLATFORM = {
	windows: 'win32',
	linux: 'linux',
	darwin: 'darwin',
}
const GOARCH2PROCESS_ARCH = {
	amd64: 'x64',
	arm64: 'arm64',
}

const binariesMatrix = Object.entries(GOOS2PROCESS_PLATFORM).flatMap(
	([goos, platform]) =>
		Object.entries(GOARCH2PROCESS_ARCH).map(([goarch, arch]) => ({
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

await genPackageWithBinary('@golar/', ({ goos, goarch }) =>
	path.join(buildDir, `golar-${goos}-${goarch}`, 'golar'),
)
await genPackageWithBinary('@golar/astro', ({ goos, goarch }) =>
	path.join(buildDir, `golar-${goos}-${goarch}`, 'golar-astro'),
)

async function genPackageWithBinary(
	npmPackageName: string,
	binaryPathFn: (pl: (typeof binariesMatrix)[number]) => string,
) {
	await Promise.all([
		...binariesMatrix.map(async ({ goarch, goos, arch, platform }) => {
			const npmPackageNamePlatform = `${npmPackageName}${npmPackageName.endsWith('/') ? '' : '-'}${platform}-${arch}`
			const packageDir = path.join(
				npmDir,
				npmPackageNamePlatform.replaceAll('/', '-'),
			)
			const binaryPath = binaryPathFn({ goarch, goos, arch, platform })
			const binaryName = `./${path.basename(binaryPath)}${platform === 'win32' ? '.exe' : ''}`

			await fs.mkdir(packageDir, { recursive: true })
			await Promise.all([
				fs.writeFile(
					path.join(packageDir, 'package.json'),
					JSON.stringify(
						{
							...commonPackageJson,
							publishConfig: {
								...commonPackageJson.publishConfig,
								executableFiles: [binaryName],
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
