import path from 'node:path'
import util from 'node:util'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('service code error', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: ['tsc', '--noEmit', '--pretty'],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		comp.vue:2:2 - error TS1000000: Element is missing end tag.

		2  <div
		   ~


		Found 1 error in comp.vue:2
		"
	`)
})
