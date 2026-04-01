import path from 'node:path'
import { savePatches } from './utils.ts'

const reporoot = path.join(import.meta.dirname, '..')
const dir = path.join(reporoot, 'thirdparty', 'go')

await savePatches('go', dir)
