import * as v from 'valibot'
import { ruleConfig } from '../../rule-creator.ts'

export const rule = ruleConfig({
	dirname: import.meta.dirname,
	options: {
		fixToUnknown: v.optional(
			v.pipe(
				v.boolean(),
				v.description(
					'Whether to enable auto-fixing in which the `any` type is converted to the `unknown` type.',
				),
			),
			false,
		),
		/**
		 * jsdoc
		 * @default false
		 */
		ignoreRestArgs: v.optional(
			v.pipe(
				v.boolean(),
				v.description('Whether to ignore rest parameter arrays.'),
			),
			false,
		),
	},
})
