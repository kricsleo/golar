import * as v from 'valibot'
import path from 'node:path'

export declare const GolarBrand: unique symbol

type SchemaWithDefault = v.OptionalSchema<v.GenericSchema, unknown>

export type RuleDefinition<TOptions extends Record<string, SchemaWithDefault>> =
	{
		[GolarBrand]: {
			options: TOptions
		}
	}

export function ruleConfig<
	TOptions extends Record<string, SchemaWithDefault> = never,
>(config: {
	dirname: string
	options?: TOptions
	// presets?: {
	// 	logical?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: v.InferInput<v.StrictObjectSchema<NoInfer<TOptions>, undefined>>)
	// 	logicalStrict?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: v.InferInput<v.StrictObjectSchema<NoInfer<TOptions>, undefined>>)
	// 	stylistic?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: v.InferInput<v.StrictObjectSchema<NoInfer<TOptions>, undefined>>)
	// 	stylisticStrict?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: v.InferInput<v.StrictObjectSchema<NoInfer<TOptions>, undefined>>)
	// 	javascript?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: v.InferInput<v.StrictObjectSchema<NoInfer<TOptions>, undefined>>)
	// }
}) {
	return {
		...config,
		isBuiltin: true,
		name: path.basename(config.dirname),
	} as unknown as RuleDefinition<TOptions>
}

export const typeOrValueSpecifier = v.pipe(
	v.array(
		v.variant('from', [
			v.strictObject({
				from: v.literal('file'),
				name: v.union([v.string(), v.array(v.string())]),
				path: v.optional(v.string()),
			}),
			v.strictObject({
				from: v.literal('lib'),
				name: v.union([v.string(), v.array(v.string())]),
			}),
			v.strictObject({
				from: v.literal('package'),
				name: v.union([v.string(), v.array(v.string())]),
				package: v.string(),
			}),
		]),
	),
	v.transform((items) =>
		items.map((item) => {
			const res = {
				from: item.from,
				name: Array.isArray(item.name) ? item.name : [item.name],
			}
			switch (item.from) {
				case 'file':
					return {
						...res,
						filePath: item.path ?? '',
					}
				case 'lib':
					return res
				case 'package':
					return {
						...res,
						package: item.package ?? '',
					}
			}
		}),
	),
)
