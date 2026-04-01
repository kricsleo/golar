import fs from 'node:fs/promises'
import path from 'node:path'
import packageJson from '../package.json' with { type: 'json' }
import process from 'node:process'
import {
	getExecutableExtension,
	getBuildArtifactName,
	GOARCH2PROCESS_ARCH,
	GOOS2PROCESS_PLATFORM,
	type GoArch,
	type GoOs,
	type LinuxLibc,
	type ProcessArch,
	type ProcessPlatform,
} from './utils.ts'

const copyBinaries = process.argv[2] === 'copy-binaries'

type BinaryVariant = {
	goarch: GoArch
	goos: GoOs
	arch: ProcessArch
	platform: ProcessPlatform
	npmPackageName: string
	libc?: LinuxLibc | undefined
}

const addonBinariesMatrix: BinaryVariant[] = [
	...(['amd64', 'arm64'] as const satisfies GoArch[])
		.flatMap((goarch) =>
			(['glibc', 'musl'] as const).map((libc) => [goarch, libc] as const),
		)
		.map(([goarch, libc]) => {
			const arch = GOARCH2PROCESS_ARCH[goarch]
			const goos = 'linux' as const satisfies GoOs
			const platform = GOOS2PROCESS_PLATFORM[goos]
			return {
				goos,
				goarch,
				platform,
				arch,
				libc,
				npmPackageName: `@golar/${platform}-${arch}${libc === 'musl' ? '-musl' : ''}`,
			}
		}),
	(() => {
		const goarch = 'amd64' as const satisfies GoArch
		const arch = GOARCH2PROCESS_ARCH[goarch]
		const goos = 'windows' as const satisfies GoOs
		const platform = GOOS2PROCESS_PLATFORM[goos]
		return {
			goos,
			goarch,
			platform,
			arch,
			npmPackageName: `@golar/${platform}-${arch}`,
		}
	})(),
	...(['amd64', 'arm64'] as const satisfies GoArch[]).map((goarch) => {
		const arch = GOARCH2PROCESS_ARCH[goarch]
		const goos = 'darwin' as const satisfies GoOs
		const platform = GOOS2PROCESS_PLATFORM[goos]
		return {
			goos,
			goarch,
			platform,
			arch,
			npmPackageName: `@golar/${platform}-${arch}`,
		}
	}),
]

const astroBinariesMatrix: BinaryVariant[] = [
	...(['linux', 'darwin'] as const satisfies GoOs[])
		.flatMap((goos) =>
			(['amd64', 'arm64'] as const satisfies GoArch[]).map(
				(goarch) => [goos, goarch] as const,
			),
		)
		.map(([goos, goarch]) => {
			const arch = GOARCH2PROCESS_ARCH[goarch]
			const platform = GOOS2PROCESS_PLATFORM[goos]
			return {
				goos,
				goarch,
				platform,
				arch,
				npmPackageName: `@golar/astro-${platform}-${arch}`,
			}
		}),
	(() => {
		const goarch = 'amd64' as const satisfies GoArch
		const arch = GOARCH2PROCESS_ARCH[goarch]
		const goos = 'windows' as const satisfies GoOs
		const platform = GOOS2PROCESS_PLATFORM[goos]
		return {
			goos,
			goarch,
			platform,
			arch,
			npmPackageName: `@golar/astro-${platform}-${arch}`,
		}
	})(),
]

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
	engines: {
		node: ">=22.12.0"
	},
} as const

const repoRoot = path.join(import.meta.dirname, '..')
const buildDir = path.join(repoRoot, 'build')

const npmDir = path.join(repoRoot, 'generated-packages')

await fs.rm(npmDir, { recursive: true, force: true })

await genPackageWithBinary(
	addonBinariesMatrix,
	({ goos, goarch, libc }) =>
		path.join(buildDir, getBuildArtifactName(goos, goarch, libc), 'golar.node'),
	false,
)
await genPackageWithBinary(
	astroBinariesMatrix,
	({ goos, goarch }) =>
		path.join(
			buildDir,
			getBuildArtifactName(goos, goarch),
			`golar-astro${getExecutableExtension(goos)}`,
		),
	true,
)

async function genPackageWithBinary(
	binariesMatrix: BinaryVariant[],
	binaryPathFn: (pl: BinaryVariant) => string,
	executable: boolean,
) {
	await Promise.all([
		...binariesMatrix.map(
			async ({ goarch, goos, arch, platform, libc, npmPackageName }) => {
				const packageDir = path.join(
					npmDir,
					npmPackageName.replaceAll('/', '-'),
				)
				const binaryPath = binaryPathFn({
					goarch,
					goos,
					arch,
					platform,
					npmPackageName,
					libc,
				})
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
								name: npmPackageName,
								preferUnplugged: true,
								files: [binaryName],
								os: [platform],
								cpu: [arch],
								...(platform === 'linux' && libc && { libc: [libc] }),
							},
							null,
							2,
						),
					),
					copyBinaries &&
						fs.copyFile(binaryPath, path.join(packageDir, binaryName)),
				])
			},
		),
	])
}
