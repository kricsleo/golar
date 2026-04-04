import child_process from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
import { test, expect, beforeAll, afterAll } from 'vitest'
import { SubprocessError } from 'nano-spawn'
import { runGolar } from '../utils.ts'

const fixtureDir = path.join(import.meta.dirname, 'fixture')
const gitDir = path.join(fixtureDir, '.git')
const gitignorePath = path.join(fixtureDir, '.gitignore')

beforeAll(async () => {
	await fs.rm(gitDir, { recursive: true, force: true })
	await fs.writeFile(gitignorePath, 'generated/\nsecret.ts\n*.auto.ts\n')
	child_process.execFileSync('git', ['init'], {
		cwd: fixtureDir,
		stdio: 'pipe',
	})
})

afterAll(async () => {
	await fs.rm(gitDir, { recursive: true, force: true })
	await fs.rm(gitignorePath, { force: true })
})

test('typecheck respects .gitignore', async () => {
	const res = await runGolar({
		cwd: fixtureDir,
		args: [],
	})

	// golar should succeed: src/index.ts has no errors,
	// generated/output.ts (directory ignore), secret.ts (file ignore),
	// and src/data.auto.ts (wildcard ignore) all have errors but are git-ignored
	expect(res).not.toBeInstanceOf(SubprocessError)
})
