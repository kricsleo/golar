import path from 'node:path'
import fs from 'node:fs/promises'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('ember emit', async () => {
	const fixtureDir = path.join(import.meta.dirname, 'fixture')
	const distDir = path.join(fixtureDir, 'dist')

	await fs.rm(distDir, { recursive: true, force: true })

	const res = await runGolar({
		cwd: fixtureDir,
		args: ['--declaration', '--emitDeclarationOnly'],
		plugins: {
			ember: true,
		},
	})
	console.log(res.output)
	expect(res).not.instanceof(SubprocessError)

	const entries = await fs.readdir(distDir)
	expect(entries).toStrictEqual(['comp.d.ts', 'index.d.ts', 'parent-comp.d.ts'])

	const comp = await fs.readFile(path.join(distDir, 'comp.d.ts'), 'utf8')
	expect(comp).toMatchInlineSnapshot(`
		"declare const _default: any;
		export default _default;
		"
	`)
})
