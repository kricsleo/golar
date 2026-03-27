import { configDefaults, defineConfig } from 'vitest/config'

export default defineConfig({
	test: {
		projects: ['packages/*', './e2e', './internal/linter/rules'],
		exclude: [...configDefaults.exclude, 'thirdparty/**'],
	},
})
