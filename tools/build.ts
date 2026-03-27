import os from 'node:os'
import fs from 'node:fs/promises'
import path from 'node:path'
import spawn from 'nano-spawn'
import { devDir, repoRoot } from './utils.ts'

const args = process.argv.slice(2)

const debug = args[0] === 'debug'
const putToBin = args[0] === 'put-to-bin'

const goDebug = debug ? ['-gcflags', 'all=-N -l'] : []

const ext = os.platform() === 'win32' ? '.exe' : ''

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
		// '-cover',
		// '-covermode=atomic',
		'-o',
		'golar.node',
		'-buildmode=c-shared',
		'./cmd/golar',
	].filter((x) => typeof x === 'string'),
	{
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
				`@golar-${process.platform}-${process.arch}`,
				'golar.node',
			),
)

await spawn(
	'go',
	[
		'build',
		...goDebug,
		'-o',
		putToBin
			? path.join(repoRoot, 'bin', `golar-astro${ext}`)
			: path.join(
					repoRoot,
					'generated-packages',
					`@golar-astro-${process.platform}-${process.arch}`,
					`golar-astro${ext}`,
				),
		'.',
	],
	{
		cwd: path.join(repoRoot, 'packages', 'astro'),
		stdio: 'inherit',
	},
)
