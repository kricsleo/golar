import { defineConfig } from 'golar/unstable'

export default defineConfig({
	typecheck: {
		include: ['**/*.ts'],
		exclude: ['**/dist', './e2e/*/fixture', './thirdparty'],
	},
})
