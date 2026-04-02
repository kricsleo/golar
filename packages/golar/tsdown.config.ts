import { defineConfig } from 'tsdown'

export default defineConfig({
	entry: [
		'./src/bin.ts',
		'./src/cli.ts',
		'./src/unstable.ts',
		'./src/unstable-tsgo.ts',
		'./src/worker.ts',
	],
	dts: true,
	exports: {
		devExports: true,
		packageJson: false,
		exclude: ['bin', 'cli', 'worker'],
	},
	// TODO: don't inline
	inlineOnly: ['valibot'],
	unbundle: true,
	fixedExtension: false,
})
