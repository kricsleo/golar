import { defineConfig } from 'tsdown'

export default defineConfig({
	entry: ['./src/golar-entry.ts'],
	unbundle: true,
	fixedExtension: false,
})
