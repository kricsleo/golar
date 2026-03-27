import path from 'node:path'
import spawn, { SubprocessError } from 'nano-spawn'

const repoRoot = path.join(import.meta.dirname, '..')

export async function runGolar(opts: { cwd: string; args: string[] }) {
	return await spawn(
		process.execPath,
		[path.join(repoRoot, 'packages', 'golar', 'src', 'bin.ts'), ...opts.args],
		{ cwd: opts.cwd },
	).catch((e) => e as SubprocessError)
}
