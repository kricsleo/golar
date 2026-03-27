import { defineConfig } from 'tsdown'

export default defineConfig({
	entry: ['./src/bin.ts', './src/unstable.ts', './src/unstable-tsgo.ts'],
	dts: true,
	exports: {
		devExports: true,
		packageJson: false,
		exclude: ['bin'],
	},
	// TODO: don't inline
	inlineOnly: ['zod'],
	unbundle: true,
	fixedExtension: false,
})
