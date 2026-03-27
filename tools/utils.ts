import path from 'node:path'
import fs from 'node:fs/promises'
import spawn from 'nano-spawn'

export const repoRoot = path.join(import.meta.dirname, '..')
export const devDir = path.join(repoRoot, 'node_modules', '.golar-dev')

export async function resetSubmodule(dir: string) {
	await spawn('git', ['tag', '-d', 'golar-base'], {
		cwd: dir,
		stdio: 'inherit',
	}).catch(() => {})

	await spawn('git', ['submodule', 'update', '--force', dir], {
		cwd: repoRoot,
		stdio: 'inherit',
	})

	await spawn('git', ['clean', '-d', '--force'], {
		cwd: dir,
		stdio: 'inherit',
	})
}

export async function uninternal(
	directories: string[],
	from: string,
	to: string,
) {
	const entries = (
		await Promise.all(
			directories.map((dir) =>
				fs.readdir(dir, {
					recursive: true,
					withFileTypes: true,
				}),
			),
		)
	).flat()

	const goFiles = entries
		.filter((e) => e.isFile() && e.name.endsWith('.go'))
		.map((e) => path.join(e.parentPath, e.name))

	await Promise.all(
		goFiles.map(async (p) => {
			const content = await fs.readFile(p, 'utf8')
			await fs.writeFile(p, content.replaceAll(from, to))
		}),
	)
}

export async function commitUninternal(dir: string) {
	await spawn('git', ['add', '--all'], {
		cwd: dir,
		stdio: 'inherit',
	})

	await spawn('git', ['commit', '--message', 'Uninternal'], {
		cwd: dir,
		stdio: 'inherit',
	})

	await spawn('git', ['tag', '--message', 'golar-base', 'golar-base'], {
		cwd: dir,
		stdio: 'inherit',
	})
}

export async function applyPatches(
	patchesDirName: string,
	submoduleDir: string,
) {
	const patchesdir = path.join(repoRoot, 'patches', patchesDirName)

	const patches = (
		await Array.fromAsync(
			fs.glob('*.patch', {
				cwd: patchesdir,
			}),
		)
	).map((p) => path.join(patchesdir, p))

	await spawn(
		'git',
		['am', '--keep-cr', '--3way', '--no-gpg-sign', ...patches],
		{
			cwd: submoduleDir,
			stdio: 'inherit',
		},
	)
}

export async function savePatches(
	patchesDirName: string,
	submoduleDir: string,
) {
	const patchesdir = path.join(repoRoot, 'patches', patchesDirName)

	await fs.rm(patchesdir, {
		recursive: true,
		force: true,
	})

	await fs.mkdir(patchesdir)

	await spawn(
		'git',
		['format-patch', '--output-directory', patchesdir, 'golar-base'],
		{
			cwd: submoduleDir,
			stdio: 'inherit',
		},
	)
}

export const GOOS2PROCESS_PLATFORM = {
	windows: 'win32',
	linux: 'linux',
	darwin: 'darwin',
}
export const GOARCH2PROCESS_ARCH = {
	amd64: 'x64',
	arm64: 'arm64',
}
export const PROCESS_PLATFORM2GOOS = Object.fromEntries(
	Object.entries(GOOS2PROCESS_PLATFORM).map(([go, node]) => [node, go]),
)
export const PROCESS_ARCH2GOARCH = Object.fromEntries(
	Object.entries(GOARCH2PROCESS_ARCH).map(([go, node]) => [node, go]),
)
