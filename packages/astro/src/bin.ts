#!/usr/bin/env node

import process from 'node:process'
import child_process from 'node:child_process'
import { fileURLToPath } from 'node:url'

const exePath = fileURLToPath(
	import.meta.resolve(
		`@golar/astro-${process.platform}-${process.arch}/golar-astro${process.platform === 'win32' ? '.exe' : ''}`,
	),
)

try {
	child_process.execFileSync(exePath, process.argv.slice(2), {
		stdio: 'inherit',
	})
} catch (e) {
	if (e instanceof Error) {
		if ('status' in e && typeof e.status === 'number') {
			process.exit(e.status)
		}
	}
	throw e
}
