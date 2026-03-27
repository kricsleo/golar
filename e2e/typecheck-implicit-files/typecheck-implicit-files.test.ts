import util from 'node:util'
import path from 'node:path'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('typecheck implicit files', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		src/comp.vue(2,8): error TS2322: Type 'string' is not assignable to type 'number'.
		src/index.ts(1,7): error TS2322: Type 'number' is not assignable to type 'string'.
		src/index.tsx(1,7): error TS2322: Type 'number' is not assignable to type 'string'."
	`)
})
