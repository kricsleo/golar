import path from 'node:path'
import fs from 'node:fs/promises'
import { test, expect } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

test('vue emit', async () => {
	const fixtureDir = path.join(import.meta.dirname, 'fixture')
	const distDir = path.join(fixtureDir, 'dist')

	await fs.rm(distDir, { recursive: true, force: true })

	const res = await runGolar({
		cwd: fixtureDir,
		args: ['--declaration', '--emitDeclarationOnly'],
		plugins: {
			vue: true,
		},
	})
	expect(res).not.instanceof(SubprocessError)

	const entries = await fs.readdir(distDir)
	expect(entries).toStrictEqual(['comp.vue.d.ts'])

	const dts = await fs.readFile(path.join(distDir, 'comp.vue.d.ts'), 'utf8')
	expect(dts).toMatchInlineSnapshot(`
		"type __VLS_Props = {
		    foo: string;
		};
		declare const _default: import("vue").DefineComponent<__VLS_Props, {}, {}, {}, {}, import("vue").ComponentOptionsMixin, import("vue").ComponentOptionsMixin, {}, string, import("vue").PublicProps, Readonly<__VLS_Props> & Readonly<{}>, {}, {}, {}, {}, string, import("vue").ComponentProvideOptions, false, {}, any>;
		export default _default;
		"
	`)
})
