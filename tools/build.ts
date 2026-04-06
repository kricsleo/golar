import fs from 'node:fs/promises'
import path from 'node:path'
import spawn from 'nano-spawn'
import {
	devDir,
	getExecutableExtension,
	getAddonPackageDirName,
	repoRoot,
	type ProcessArch,
	type ProcessPlatform,
} from './utils.ts'
import { family as libcFamily, MUSL } from 'detect-libc'

const args = process.argv.slice(2)

const debug = args.includes('debug')
const release = args.includes('release')
const putToBin = args.includes('put-to-bin')

const goDebug = debug ? ['-gcflags', 'all=-N -l'] : []
const goRelease = release ? ['-trimpath'] : []

const libc =
	process.env.GOLAR_LIBC === 'musl' || (await libcFamily()) === MUSL
		? 'musl'
		: 'glibc'

const objDir = path.join(devDir, 'build-obj')
await fs.mkdir(objDir, { recursive: true })

await spawn(
	'go',
	[
		'tool',
		'cgo',
		'-exportheader',
		'exports.h',
		'-objdir',
		objDir,
		'exports.go',
		'js_plugin.go',
	],
	{
		cwd: path.join(repoRoot, 'cmd', 'golar'),
		stdio: 'inherit',
	},
)

await spawn(
	'go',
	[
		'build',
		...goDebug,
		...goRelease,
		// '-cover',
		// '-covermode=atomic',
		'-o',
		'golar.node',
		'-buildmode=c-shared',
		'./cmd/golar',
	].filter((x) => typeof x === 'string'),
	{
		env: {
			...process.env,
			...(libc === 'musl'
				? {
						CGO_CFLAGS: [process.env.CGO_CFLAGS, '-DGOLAR_NAPI_DYNAMIC']
							.filter((v) => typeof v === 'string')
							.join(' '),
					}
				: {}),
		},
		stdio: 'inherit',
	},
)

await fs.rm('golar.h')

if (putToBin) {
	await fs.mkdir('bin')
}

await fs.copyFile(
	'golar.node',
	putToBin
		? path.join(repoRoot, 'bin', 'golar.node')
		: path.join(
				repoRoot,
				'generated-packages',
				getAddonPackageDirName(
					process.platform as ProcessPlatform,
					process.arch as ProcessArch,
					libc,
				),
				'golar.node',
			),
)

await spawn(
	'go',
	[
		'build',
		...goDebug,
		...goRelease,
		'-o',
		putToBin
			? path.join(
					repoRoot,
					'bin',
					`golar-astro${getExecutableExtension(process.platform as ProcessPlatform)}`,
				)
			: path.join(
					repoRoot,
					'generated-packages',
					`@golar-astro-${process.platform}-${process.arch}`,
					`golar-astro${getExecutableExtension(process.platform as ProcessPlatform)}`,
				),
		'.',
	],
	{
		cwd: path.join(repoRoot, 'packages', 'astro'),
		stdio: 'inherit',
	},
)
