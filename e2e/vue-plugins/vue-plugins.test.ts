import path from 'node:path'
import util from 'node:util'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('vue plugins', async () => {
	const res = await runGolar({
		cwd: path.join(import.meta.dirname, 'fixture'),
		args: [],
	})

	expect(res).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(res.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		src/pages/[id].vue(5,15): error TS2339: Property 'unknown' does not exist on type '{ id: string; }'."
	`)
})
