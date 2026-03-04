import path from 'node:path'
import util from 'node:util'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('vue unknown component is ignored by default', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['--noEmit', '--pretty'],
		plugins: {
			vue: true,
		},
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"comp.vue:2:7 - error TS2322: Type 'number' is not assignable to type 'string'.

		2 const foo: string = 213
		        ~~~


		Found 1 error in comp.vue:2
		"
	`)
})
