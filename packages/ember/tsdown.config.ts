import { defineConfig } from 'tsdown'

export default defineConfig({
	entry: ['./src/bin.ts', './src/golar-entry.ts'],
	unbundle: true,
	fixedExtension: false,
})
