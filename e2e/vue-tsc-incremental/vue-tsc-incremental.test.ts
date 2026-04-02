import path from 'node:path'
import fs from 'node:fs/promises'
import util from 'node:util'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

const fixtureDir = path.join(import.meta.dirname, 'fixture')
const outDir = path.join(fixtureDir, 'dist')

test('vue tsc incremental preserves source positions', async () => {
	await fs.rm(outDir, { recursive: true, force: true })

	const firstRun = await runGolar({
		cwd: fixtureDir,
		args: ['tsc', '--pretty'],
	})
	const secondRun = await runGolar({
		cwd: fixtureDir,
		args: ['tsc', '--pretty'],
	})

	expect(firstRun).instanceof(SubprocessError)
	expect(secondRun).instanceof(SubprocessError)
	expect(util.stripVTControlCharacters(firstRun.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		comp.vue:6:8 - error TS2322: Type 'number' is not assignable to type 'string'.

		6  const foo: string = 123
		         ~~~


		Found 1 error in comp.vue:6
		"
	`)
	expect(util.stripVTControlCharacters(secondRun.output)).toMatchInlineSnapshot(`
		"Using config from ./golar.config.ts...
		comp.vue:6:8 - error TS2322: Type 'number' is not assignable to type 'string'.

		6  const foo: string = 123
		         ~~~


		Found 1 error in comp.vue:6
		"
	`)
})
