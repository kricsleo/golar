import { bench, describe } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { globSync } from 'tinyglobby'

const cwd = path.resolve(import.meta.dirname, '../../..')

describe('simple pattern: **/*.ts', () => {
	bench('fs.globSync', () => {
		fs.globSync('**/*.ts', { cwd, withFileTypes: true })
			.filter((entry) => entry.isFile())
			.map((entry) => path.resolve(cwd, entry.parentPath, entry.name))
	})

	bench('tinyglobby.globSync', () => {
		globSync('**/*.ts', { cwd, absolute: true })
	})
})

describe('with exclude/ignore: **/*.ts', () => {
	const excludePatterns = ['**/node_modules', '**/.git', '**/.jj']
	const excludeNames = new Set(excludePatterns.map((p) => p.replace('**/', '')))

	bench('fs.globSync', () => {
		fs.globSync('**/*.ts', {
			cwd,
			exclude: (file) => excludeNames.has(file.name),
			withFileTypes: true,
		})
			.filter((entry) => entry.isFile())
			.map((entry) => path.resolve(cwd, entry.parentPath, entry.name))
	})

	bench('tinyglobby.globSync', () => {
		globSync('**/*.ts', { cwd, ignore: excludePatterns, absolute: true })
	})
})

describe('multiple patterns: **/*.{ts,tsx,vue}', () => {
	const patterns = ['**/*.ts', '**/*.tsx', '**/*.vue']
	const excludePatterns = ['**/node_modules', '**/.git', '**/.jj']
	const excludeNames = new Set(excludePatterns.map((p) => p.replace('**/', '')))

	bench('fs.globSync', () => {
		fs.globSync(patterns, {
			cwd,
			exclude: (file) => excludeNames.has(file.name),
			withFileTypes: true,
		})
			.filter((entry) => entry.isFile())
			.map((entry) => path.resolve(cwd, entry.parentPath, entry.name))
	})

	bench('tinyglobby.globSync', () => {
		globSync(patterns, { cwd, ignore: excludePatterns, absolute: true })
	})
})
