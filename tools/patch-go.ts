import path from 'node:path'
import {
	applyPatches,
	golarBaseTag,
	repoRoot,
	resetSubmodule,
} from './utils.ts'

const dir = path.join(repoRoot, 'thirdparty', 'go')

await resetSubmodule(dir)

await golarBaseTag(dir)

await applyPatches('go', dir)
