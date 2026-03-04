import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('@vue-expect-error', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['--noEmit', '--pretty'],
		plugins: {
			vue: true,
		},
	})

	expect(res).not.instanceof(SubprocessError)
})
