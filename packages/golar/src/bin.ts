#!/usr/bin/env node

import process from 'node:process'
import fs from 'node:fs/promises'
import child_process from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const plugins = new Map<string, string[]>()
const filename = import.meta.filename.replaceAll('\\', '/')

await Promise.all(
	filename.matchAll(/\/node_modules\//g).map(async (nodeModulesMatch) => {
		const nodeModulesPath = filename.slice(
			0,
			nodeModulesMatch.index + nodeModulesMatch[0].length,
		)
		for (const orgName of await fs.readdir(nodeModulesPath)) {
			if (!orgName.startsWith('@')) {
				continue
			}
			for (const packageName of await fs.readdir(
				path.join(nodeModulesPath, orgName),
			)) {
				if (
					(orgName !== '@golar' ||
						!['astro', 'svelte', 'vue'].includes(packageName)) &&
					!/^golar-plugin(-.+)?$/.exec(packageName)
				) {
					continue
				}
				const packagePath = path.join(nodeModulesPath, orgName, packageName)
				const packageSpecifier = `${orgName}/${packageName}`
				const packageJson = JSON.parse(
					await fs.readFile(path.join(packagePath, 'package.json'), 'utf8'),
				)
				if (packageJson == null || typeof packageJson !== 'object') {
					continue
				}
				const mod = await import(`${packageSpecifier}/golar-entry`)
				if (typeof mod.getGolarEntry !== 'function') {
					continue
				}
				plugins.set(packageSpecifier, await mod.getGolarEntry())
			}
		}
	}),
)

const exePath = fileURLToPath(
	import.meta.resolve(
		`@golar/${process.platform}-${process.arch}/golar${process.platform === 'win32' ? '.exe' : ''}`,
	),
)

try {
	child_process.execFileSync(exePath, process.argv.slice(2), {
		env: {
			GOLAR_PLUGINS: Array.from(
				plugins.values().map((args) => args.join('\x1f')),
			).join('\x1e'),
			...process.env,
		},
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
