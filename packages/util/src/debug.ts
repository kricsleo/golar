import util from 'node:util'
import process from 'node:process'

type CompiledDebug = {
	matches?: RegExp | undefined
	skips?: RegExp | undefined
}

let envCompiled: CompiledDebug | null = null

export class Debug {
	private namespace: string
	private enabled: boolean

	static create(namespace: string) {
		return new Debug(namespace, envCompiled ??= compileDebug(process.env.DEBUG))
	}

	private constructor(namespace: string, compiled: CompiledDebug) {
		this.namespace = 'golar:' + namespace
		this.enabled = !compiled.skips?.test(this.namespace) && compiled.matches?.test(this.namespace) != null
	}

	print(...args: string[]): void {
		if (this.enabled) {
			process.stderr.write(this.prefix() + args.join(' ') + '\n')
		}
	}

	printf(format: string, ...args: unknown[]): void {
		if (this.enabled) {
			process.stderr.write(this.prefix() + util.formatWithOptions({ depth: 15 }, format + '\n', ...args))
		}
	}

	private prefix(): string {
		return `${this.namespace} `
	}
}

export function createDebugFromString(debug: string, namespace: string): Debug {
	// @ts-expect-error
	return new Debug(namespace, compileDebug(debug))
}

function compileDebug(debug: string | undefined): CompiledDebug {
	if (!debug) {
		return {}
	}

	let matchesPattern = '^'
	let skipsPattern = '^'

	for (const rawPart of debug.split(',')) {
		if (rawPart === '') {
			continue
		}

		let part = rawPart
		let isSkip = false
		if (part[0] === '-') {
			part = part.slice(1)
			isSkip = true
			if (part === '') {
				continue
			}
		}

		const patternPart = part.replaceAll('*', '.*')
		if (isSkip) {
			if (skipsPattern.length > 1) {
				skipsPattern += '|'
			}
			skipsPattern += patternPart
			continue
		}
		if (matchesPattern.length > 1) {
			matchesPattern += '|'
		}
		matchesPattern += patternPart
	}

	if (matchesPattern.length <= 1) {
		return {}
	}

	matchesPattern += '$'
	skipsPattern += '$'

	return {
		matches: new RegExp(matchesPattern),
		skips: skipsPattern.length > 2 ? new RegExp(skipsPattern) : undefined,
	}
}
