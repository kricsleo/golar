import os from 'node:os'
import process from 'node:process'
import url from 'node:url'
import worker_threads from 'node:worker_threads'

const addonModule = {
	exports: {} as {
		setSyncBuffer(threadId: number, buffer: ArrayBuffer): void
		registerJsCodegen(threadId: number, createServiceCode: () => void): void
		registerIpcCodegen(): void
		tsc(threadId: number, done: (exitCode: number) => void): void

		workspace_New(cb: () => void): void
		workspace_ReadRequestedFileAt(threadId: number): void
		workspace_ReadFileById(threadId: number): void
		workspace_GetTypeAtLocation(threadId: number): void
		workspace_Lint(): void
		workspace_TypeCheck(): number
		workspace_Report(threadId: number): void
	},
}

export const golarAddonPath = url.fileURLToPath(
	import.meta.resolve(`@golar/${process.platform}-${process.arch}/golar.node`),
)

process.dlopen(
	addonModule,
	golarAddonPath,
	os.constants.dlopen.RTLD_NOW | os.constants.dlopen.RTLD_GLOBAL,
)
export const { exports: addon } = addonModule

// TODO: grow dynamically?
export const syncBuf = new ArrayBuffer(10 * 1024 * 1024)
export const syncView = new DataView(syncBuf)

addon.setSyncBuffer(worker_threads.threadId, syncBuf)
