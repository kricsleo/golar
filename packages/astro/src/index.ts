import process from 'node:process'
import { fileURLToPath } from 'node:url'
import { IpcCodegenPlugin } from 'golar/unstable'

new IpcCodegenPlugin({
	id: 'astro',
	cmd: [
		fileURLToPath(
			import.meta.resolve(
				`@golar/astro-${process.platform}-${process.arch}/golar-astro${process.platform === 'win32' ? '.exe' : ''}`,
			),
		),
	],
	extensions: [
		{
			extension: '.astro',
			stripFromDeclarationFileName: false,
			allowExtensionlessImports: false,
		},
	],
})
