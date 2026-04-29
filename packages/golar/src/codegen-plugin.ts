import os from 'node:os'
import worker_threads from 'node:worker_threads'
import { addon, loadAddon, syncBuf, syncView } from './addon.ts'

export type FileExtension = {
	/** Include the leading dot, e.g. '.vue'. */
	extension: string
	stripFromDeclarationFileName: boolean
	allowExtensionlessImports: boolean
}

export type Promisable<T> = T | Promise<T>

export type CodegenMapping = {
	sourceOffset: number
	serviceOffset: number
	sourceLength: number
	serviceLength?: number | undefined
	suppressedDiagnostics?: number[] | undefined
}

export type IgnoreDirectiveCodegenMapping = {
	serviceOffset: number
	serviceLength: number
}

export type ExpectErrorDirectiveCodegenMapping = {
	sourceOffset: number
	serviceOffset: number
	sourceLength: number
	serviceLength: number
}

export type CodegenScriptKind = 'js' | 'jsx' | 'ts' | 'tsx'

export type ServiceCodeError = {
	message: string
	start: number
	end: number
}

export type ServiceCode =
	| {
			mappings: CodegenMapping[]
			errors?: never
			serviceText: string
			scriptKind: CodegenScriptKind
			/** @default false */
			declarationFile?: boolean | undefined
			ignoreMappings?: IgnoreDirectiveCodegenMapping[] | undefined
			expectErrorMappings?: ExpectErrorDirectiveCodegenMapping[] | undefined
			ignoreNotMappedDiagnostics?: boolean | undefined
	  }
	| {
			mappings?: never
			errors: ServiceCodeError[]
	  }

const textDecoder = new TextDecoder()
const textEncoder = new TextEncoder()

const SERVICE_CODE_PROPERTIES = {
	ERROR: 1 << 0,
	DECLARATION_FILE: 1 << 1,
	IGNORE_NOT_MAPPED_DIAGNOSTICS: 1 << 2,
}

const SCRIPT_KIND = {
	js: 0,
	jsx: 1,
	ts: 2,
	tsx: 3,
} as const satisfies Record<CodegenScriptKind, number>

const stateSymbol = Symbol.for('golar-global-state')
const globalTyped = globalThis as typeof globalThis & {
	[stateSymbol]?: {
		codegenPlugins: Map<string, CodegenPlugin>
	}
}

export type CodegenPlugin = JsCodegenPlugin | IpcCodegenPlugin

export const globalState = (globalTyped[stateSymbol] ??= {
	codegenPlugins: new Map(),
})

function registerCodegenPlugin(plugin: CodegenPlugin) {
	// TODO: check duplicates, but allow re-registration
	// if (globalState.codegenPlugins.get(plugin.id)) {
	// 	throw new Error(`Duplicate ${plugin.id} codegen plugin`)
	// }
	globalState.codegenPlugins.set(plugin.id, plugin)
}

export type CreateServiceCodeFn = (
	cwd: string,
	configFileName: string | null,
	fileName: string,
	sourceText: string,
) => Promisable<ServiceCode>

export type JsCodegenPluginOptions = {
	id: string
	extensions: FileExtension[]
	createServiceCode: CreateServiceCodeFn
}

export class JsCodegenPlugin {
	readonly id: string
	readonly idNumeric: number
	readonly extensions: FileExtension[]
	readonly createServiceCode: CreateServiceCodeFn

	static pluginIdCounter = 0

	constructor(opts: JsCodegenPluginOptions) {
		this.id = opts.id
		this.idNumeric = JsCodegenPlugin.pluginIdCounter++
		this.extensions = opts.extensions
		this.createServiceCode = opts.createServiceCode

		registerCodegenPlugin(this)
	}

	register() {
		let offset = 0
		syncView.setUint32(offset, this.idNumeric, true)
		offset += 4
		const { written: initializationLen } = textEncoder.encodeInto(
			JSON.stringify({
				extensions: this.extensions,
			}),
			new Uint8Array(syncBuf, offset + 4),
		)
		syncView.setUint32(offset, initializationLen, true)
		offset += 4 + initializationLen

		loadAddon()
		addon.registerJsCodegen(worker_threads.threadId, () =>
			this.executeCreateServiceCode(),
		)
	}

	// TODO: adjust exports.c to understand sync/async
	async executeCreateServiceCode() {
		let offset = 0

		const codegenPluginPtr = syncView.getBigUint64(offset, true)
		offset += 8

		const cwdLen = syncView.getUint32(offset, true)
		const cwd = textDecoder.decode(
			new Uint8Array(syncBuf, (offset += 4), cwdLen),
		)
		offset += cwdLen

		const configFileNameLen = syncView.getUint32(offset, true)
		const configFileName = textDecoder.decode(
			new Uint8Array(syncBuf, (offset += 4), configFileNameLen),
		)
		offset += configFileNameLen

		const fileNameLen = syncView.getUint32(offset, true)
		const fileName = textDecoder.decode(
			new Uint8Array(syncBuf, (offset += 4), fileNameLen),
		)
		offset += fileNameLen

		const sourceTextLen = syncView.getUint32(offset, true)
		const sourceText = textDecoder.decode(
			new Uint8Array(syncBuf, (offset += 4), sourceTextLen),
		)
		offset += sourceTextLen

		const resp = this.createServiceCode(
			cwd,
			configFileName === '/dev/null/inferred' ? null : configFileName,
			fileName,
			sourceText,
		)

		if (resp instanceof Promise) {
			return resp.then((resp) =>
				this.processCreatedServiceCode(sourceText, codegenPluginPtr, resp),
			)
		}
		return this.processCreatedServiceCode(sourceText, codegenPluginPtr, resp)
	}

	private processCreatedServiceCode(
		sourceText: string,
		codegenPluginPtr: bigint,
		resp: ServiceCode,
	) {
		let offset = 0
		let properties = 0

		if ('errors' in resp) {
			const errorLocationsUtf8 = mapIndicesToUtf8(
				sourceText,
				resp.errors[Symbol.iterator]().flatMap((e) => [e.start, e.end]),
			)
			syncView.setBigUint64(offset, codegenPluginPtr, true)
			syncView.setUint8(
				(offset += 8),
				properties | SERVICE_CODE_PROPERTIES.ERROR,
			)
			syncView.setUint32((offset += 1), resp.errors.length, true)
			offset += 4
			for (const err of resp.errors) {
				const { written: msgLen } = textEncoder.encodeInto(
					err.message,
					new Uint8Array(syncBuf, offset + 4),
				)
				syncView.setUint32(offset, msgLen, true)
				offset += 4 + msgLen
				syncView.setUint32(offset, errorLocationsUtf8(err.start)!, true)
				syncView.setUint32((offset += 4), errorLocationsUtf8(err.end)!, true)
				offset += 4
			}

			return
		}

		if (resp.declarationFile) {
			properties |= SERVICE_CODE_PROPERTIES.DECLARATION_FILE
		}
		if (resp.ignoreNotMappedDiagnostics) {
			properties |= SERVICE_CODE_PROPERTIES.IGNORE_NOT_MAPPED_DIAGNOSTICS
		}

		const writeUint32 =
			os.endianness() === 'LE'
				? (offset: number, value: number) =>
						syncView.setUint32(offset, value, true)
				: (offset: number, value: number) =>
						syncView.setUint32(offset, value, false)

		syncView.setBigUint64(offset, codegenPluginPtr, true)
		syncView.setUint8((offset += 8), properties)
		syncView.setUint8((offset += 1), SCRIPT_KIND[resp.scriptKind])
		offset += 1
		const { written: serviceTextLen } = textEncoder.encodeInto(
			resp.serviceText,
			new Uint8Array(syncBuf, offset + 4),
		)
		syncView.setUint32(offset, serviceTextLen, true)
		offset += 4 + serviceTextLen

		const sourceIndicesUtf8 = mapIndicesToUtf8(
			sourceText,
			(function* () {
				for (const m of resp.mappings) {
					yield m.sourceOffset
					yield m.sourceOffset + m.sourceLength
				}
				for (const m of resp.expectErrorMappings ?? []) {
					yield m.sourceOffset
					yield m.sourceOffset + m.sourceLength
				}
			})(),
		)
		const serviceIndicesUtf8 = mapIndicesToUtf8(
			resp.serviceText,
			(function* () {
				for (const m of resp.mappings) {
					yield m.serviceOffset
					yield m.serviceOffset + (m.serviceLength ?? m.sourceLength)
				}
				for (const m of resp.ignoreMappings ?? []) {
					yield m.serviceOffset
					yield m.serviceOffset + m.serviceLength
				}
				for (const m of resp.expectErrorMappings ?? []) {
					yield m.serviceOffset
					yield m.serviceOffset + m.serviceLength
				}
			})(),
		)

		syncView.setUint32(offset, resp.mappings.length, true)
		offset += 4
		for (const m of resp.mappings) {
			const sourceOffsetUtf8 = sourceIndicesUtf8(m.sourceOffset)!
			const serviceOffsetUtf8 = serviceIndicesUtf8(m.serviceOffset)!
			writeUint32(offset, sourceOffsetUtf8)
			writeUint32((offset += 4), serviceOffsetUtf8)
			writeUint32(
				(offset += 4),
				sourceIndicesUtf8(m.sourceOffset + m.sourceLength)! - sourceOffsetUtf8,
			)
			writeUint32(
				(offset += 4),
				serviceIndicesUtf8(
					m.serviceOffset + (m.serviceLength ?? m.sourceLength),
				)! - serviceOffsetUtf8,
			)
			const suppressedDiagnostics = m.suppressedDiagnostics ?? []
			writeUint32((offset += 4), suppressedDiagnostics.length)
			offset += 4
			for (const code of suppressedDiagnostics) {
				writeUint32(offset, code)
				offset += 4
			}
		}

		syncView.setUint32(offset, resp.ignoreMappings?.length ?? 0, true)
		offset += 4
		for (const m of resp.ignoreMappings ?? []) {
			const serviceOffsetUtf8 = serviceIndicesUtf8(m.serviceOffset)!
			writeUint32(offset, serviceOffsetUtf8)
			writeUint32(
				(offset += 4),
				serviceIndicesUtf8(m.serviceOffset + m.serviceLength)! -
					serviceOffsetUtf8,
			)
			offset += 4
		}

		syncView.setUint32(offset, resp.expectErrorMappings?.length ?? 0, true)
		offset += 4
		for (const m of resp.expectErrorMappings ?? []) {
			const sourceOffsetUtf8 = sourceIndicesUtf8(m.sourceOffset)!
			const serviceOffsetUtf8 = serviceIndicesUtf8(m.serviceOffset)!
			writeUint32(offset, sourceOffsetUtf8)
			writeUint32((offset += 4), serviceOffsetUtf8)
			writeUint32(
				(offset += 4),
				sourceIndicesUtf8(m.sourceOffset + m.sourceLength)! - sourceOffsetUtf8,
			)
			writeUint32(
				(offset += 4),
				serviceIndicesUtf8(m.serviceOffset + m.serviceLength)! -
					serviceOffsetUtf8,
			)
			offset += 4
		}
	}
}

export type IpcCodegenPluginOptions = {
	id: string
	cmd: readonly string[]
	extensions: FileExtension[]
}

export class IpcCodegenPlugin {
	readonly id: string
	readonly cmd: readonly string[]
	readonly extensions: FileExtension[]

	constructor(opts: IpcCodegenPluginOptions) {
		this.id = opts.id
		this.cmd = opts.cmd
		this.extensions = opts.extensions

		registerCodegenPlugin(this)
	}

	register() {
		const { written: initializationLen } = textEncoder.encodeInto(
			JSON.stringify({
				cmd: this.cmd,
				extensions: this.extensions,
			}),
			new Uint8Array(syncBuf, 4),
		)
		syncView.setUint32(0, initializationLen, true)

		loadAddon()
		addon.registerIpcCodegen()
	}
}

function mapIndicesToUtf8(text: string, indices: Iterable<number>) {
	const mapped = new Map<number, number>()
	const sorted = Array.from(new Set<number>(indices)).sort((a, b) => a - b)
	let utf8Offset = 0
	for (const [i, idx] of sorted.entries()) {
		mapped.set(
			idx,
			(utf8Offset += Buffer.byteLength(text.slice(sorted[i - 1] ?? 0, idx))),
		)
	}
	return mapped.get.bind(mapped)
}
