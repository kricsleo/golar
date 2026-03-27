import { z } from 'zod'
import path from 'node:path'

export declare const GolarBrand: unique symbol

export type RuleDefinition<TOptions extends Record<string, z.ZodDefault>> = {
	[GolarBrand]: {
		options: TOptions
	}
}

export function ruleConfig<
	TOptions extends Record<string, z.ZodDefault> = never,
>(config: {
	dirname: string
	options?: TOptions
	// presets?: {
	// 	logical?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: z.input<z.ZodObject<NoInfer<TOptions>, z.core.$strict>>)
	// 	logicalStrict?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: z.input<z.ZodObject<NoInfer<TOptions>, z.core.$strict>>)
	// 	stylistic?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: z.input<z.ZodObject<NoInfer<TOptions>, z.core.$strict>>)
	// 	stylisticStrict?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: z.input<z.ZodObject<NoInfer<TOptions>, z.core.$strict>>)
	// 	javascript?:
	// 		| true
	// 		| ([TOptions] extends [never]
	// 				? never
	// 				: z.input<z.ZodObject<NoInfer<TOptions>, z.core.$strict>>)
	// }
}) {
	return {
		...config,
		isBuiltin: true,
		name: path.basename(config.dirname),
	} as unknown as RuleDefinition<TOptions>
}

export const typeOrValueSpecifier = z.array(
	z
		.discriminatedUnion('from', [
			z.strictObject({
				from: z.literal('file'),
				name: z.union([z.string(), z.array(z.string())]),
				path: z.string().optional(),
			}),
			z.strictObject({
				from: z.literal('lib'),
				name: z.union([z.string(), z.array(z.string())]),
			}),
			z.strictObject({
				from: z.literal('package'),
				name: z.union([z.string(), z.array(z.string())]),
				package: z.string(),
			}),
		])
		.transform((v) => {
			const res = {
				from: v.from,
				name: Array.isArray(v.name) ? v.name : [v.name],
			}
			switch (v.from) {
				case 'file':
					return {
						...res,
						filePath: v.path ?? '',
					}
				case 'lib':
					return res
				case 'package':
					return {
						...res,
						package: v.package ?? '',
					}
			}
		}),
)
