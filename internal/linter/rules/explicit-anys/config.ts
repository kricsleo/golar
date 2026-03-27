import { z } from 'zod'
import { ruleConfig } from '../../rule-creator.ts'

export const rule = ruleConfig({
	dirname: import.meta.dirname,
	options: {
		fixToUnknown: z
			.boolean()
			.default(false)
			.describe(
				'Whether to enable auto-fixing in which the `any` type is converted to the `unknown` type.',
			),
		/**
		 * jsdoc
		 * @default false
		 */
		ignoreRestArgs: z
			.boolean()
			.default(false)
			.describe('Whether to ignore rest parameter arrays.'),
	},
})
