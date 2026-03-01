import { configDefaults, defineConfig } from 'vitest/config'

export default defineConfig({
	test: {
		projects: ['packages/*', './e2e'],
		exclude: [...configDefaults.exclude, 'thirdparty/**'],
	},
})
