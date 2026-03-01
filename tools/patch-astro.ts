import path from 'node:path'
import fs from 'node:fs/promises'
import {
	applyPatches,
	commitUninternal,
	repoRoot,
	resetSubmodule,
	uninternal,
} from './utils.ts'

const dir = path.join(repoRoot, 'thirdparty', 'astro-compiler')

await resetSubmodule(dir)

await fs.rm(path.join(dir, 'pkg'), {
	recursive: true,
	force: true,
})

await uninternal(
	[path.join(dir, 'cmd'), path.join(dir, 'internal')],
	'github.com/withastro/compiler/internal',
	'github.com/withastro/compiler/pkg',
)

await fs.rename(path.join(dir, 'internal'), path.join(dir, 'pkg'))

await commitUninternal(dir)

await applyPatches('astro-compiler', dir)
