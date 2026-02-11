import { defineConfig } from 'tsdown'

export default defineConfig({
	entry: './src/bin.ts',
	unbundle: true,
	fixedExtension: false,
})
