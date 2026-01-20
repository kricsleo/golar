/// <reference types="@volar/typescript" />

import { createPlugin, type IgnoreDirectiveMapping, type Mapping } from '@golar/plugin'
import type { LanguagePlugin } from '@volar/language-core'

type Promisable<T> = T | Promise<T>

export type CreateVolarPluginOptions = {
	filename: string
	getLanguagePlugins: () => Promisable<LanguagePlugin<string>[]>
}

export function createVolarPlugin(opts: CreateVolarPluginOptions) {
	let _languagePlugins: Promisable<LanguagePlugin<string>[]> | null = null
	createPlugin({
		filename: opts.filename,
		async createServiceCode(fileName, sourceText) {
			for (const plugin of await (_languagePlugins ??= opts.getLanguagePlugins())) {
				if (plugin.createVirtualCode == null) {
					continue
				}
				const languageId = plugin.getLanguageId(fileName)
				if (languageId == null) {
					continue
				}

				const virtualCode = plugin.createVirtualCode(fileName, languageId, {
					getLength() {
						return sourceText.length
					},
					getText(start, end) {
					  return sourceText.slice(start, end)
					},
					dispose() {},
					getChangeRange() {
						return undefined
					},
				}, {
					getAssociatedScript(scriptId) {
						return undefined
					},
				})
				if (virtualCode == null) {
					continue
				}

				const serviceScript = plugin.typescript!.getServiceScript(virtualCode)
				const serviceText = serviceScript!.code.snapshot.getText(0, serviceScript!.code.snapshot.getLength())

				const verificationMappings = serviceScript!.code.mappings.filter(m => m.data.verification)
				const sourceOffsets = new Set<number>()
				const serviceOffsets = new Set<number>()

				for (const m of verificationMappings) {
					for (const [i, offset] of m.sourceOffsets.entries()) {
						sourceOffsets.add(offset)
						sourceOffsets.add(offset + m.lengths[i]!)
					}
					for (const [i, offset] of m.generatedOffsets.entries()) {
						serviceOffsets.add(offset)
						serviceOffsets.add(offset + (m.generatedLengths ?? m.lengths)[i]!)
					}
				}

				const sourceOffsetsUtf8 = new Map<number, number>()
				const serviceOffsetsUtf8 = new Map<number, number>()

				let currentUtf8Pos = 0
				const sortedSourceOffsets = Array.from(sourceOffsets).sort((a, b) => a - b)
				for (const [i, offset] of sortedSourceOffsets.entries()) {
					sourceOffsetsUtf8.set(offset, currentUtf8Pos += Buffer.byteLength(sourceText.slice(sortedSourceOffsets[i-1] ?? 0, offset)))
				}
				currentUtf8Pos = 0
				const sortedServiceOffsets = Array.from(serviceOffsets).sort((a, b) => a - b)
				for (const [i, offset] of sortedServiceOffsets.entries()) {
					serviceOffsetsUtf8.set(offset, currentUtf8Pos += Buffer.byteLength(serviceText.slice(sortedServiceOffsets[i-1] ?? 0, offset)))
				}

				const serviceCovered: [number, number][] = []
				const mappings = verificationMappings
					.map((m): Mapping => {
						for (const [i, offset] of m.generatedOffsets.entries()) {
  			   	const len = m.generatedLengths?.[i] ?? m.lengths[i]!;
  			    	if (len > 0) {
								serviceCovered.push([serviceOffsetsUtf8.get(offset)!, serviceOffsetsUtf8.get(offset + len)!])
							}
						}
						return {
							sourceOffsets: m.sourceOffsets.map(o => sourceOffsetsUtf8.get(o)!),
							serviceOffsets: m.generatedOffsets.map(o => serviceOffsetsUtf8.get(o)!),
							sourceLengths: m.lengths.map((l, i) => sourceOffsetsUtf8.get(m.sourceOffsets[i]! + l)! - sourceOffsetsUtf8.get(m.sourceOffsets[i]!)!),
							serviceLengths: m.generatedLengths?.map((l, i) => serviceOffsetsUtf8.get(m.generatedOffsets[i]! + l)! - serviceOffsetsUtf8.get(m.generatedOffsets[i]!)!),
						}
					})

  			serviceCovered.sort((a, b) => a[0] - b[0] || a[1] - b[1]);

  			const merged: [number, number][] = [];
  			for (const [s, e] of serviceCovered) {
  			  const last = merged.at(-1);
  			  if (!last) {
						merged.push([s, e])
					} else if (s <= last[1]) {
						last[1] = Math.max(last[1], e)
					} else {
						merged.push([s, e]);
					}
  			}

  			const ignoreMappings: IgnoreDirectiveMapping[] = [];
  			let cursor = 0;

  			for (const [s, e] of merged) {
  			  if (s > cursor) {
						ignoreMappings.push({ serviceOffset: cursor, serviceLength: s - cursor });
					}
  			  cursor = Math.max(cursor, e);
  			}

  			ignoreMappings.push({ serviceOffset: cursor, serviceLength: Buffer.byteLength(serviceText) - cursor });

				return {
					serviceText,
					mappings,
					ignoreMappings,
				}
			}
			throw new Error('Unknown language')
		},
	})
}

