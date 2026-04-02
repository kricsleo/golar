import * as v from 'valibot'
import { rules as generated } from './builtin-rules.generated.ts'
import type { GolarBrand } from '../../../internal/linter/rule-creator.ts'
import type { LintConfiguredRule } from './config.ts'

export const rules: (configs: {
	[TRule in keyof typeof generated]?:
		| true
		| undefined
		| v.InferInput<
				v.ObjectSchema<(typeof generated)[TRule][typeof GolarBrand]['options'] & v.ObjectEntries, undefined>
		  >
}) => LintConfiguredRule[] = (configs) =>
	Object.entries(configs).map(
		([name, config]) =>
			({
				// TODO: better message
				// @ts-expect-error
				rule: generated[name],
				// TODO: what if false?
				options: config === true ? {} : config,
			}) as unknown as LintConfiguredRule,
	)
