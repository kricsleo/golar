import path from 'node:path'
import { savePatches } from './utils.ts'

const reporoot = path.join(import.meta.dirname, '..')
const dir = path.join(reporoot, 'thirdparty', 'astro-compiler')

await savePatches('astro-compiler', dir)
