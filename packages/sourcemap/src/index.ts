import type { Mapping } from '@golar/plugin'
import { decode } from '@jridgewell/sourcemap-codec'

export function sourceMapToMappings(opts: {
	sourceText: string
	serviceText: string
	sourceMap: string
}): Mapping[] {
	const v3Mappings = decode(opts.sourceMap)
	const sourceTextWithLineMap: SourceFileWithLineMap = {
		text: opts.sourceText,
	}
	const serviceTextWithLineMap: SourceFileWithLineMap = {
		text: opts.serviceText,
	}
	const mappings: Mapping[] = []

	let current:
		| {
				serviceOffset: number
				sourceOffset: number
		  }
		| undefined

	for (const [serviceLine, segments] of v3Mappings.entries()) {
		for (const segment of segments) {
			const serviceCharacter = segment[0]
			const serviceOffset = getPositionOfColumnAndLine(serviceTextWithLineMap, {
				line: serviceLine,
				column: serviceCharacter,
			})
			if (current) {
				let length = serviceOffset - current.serviceOffset
				const sourceText = opts.sourceText.substring(
					current.sourceOffset,
					current.sourceOffset + length,
				)
				const serviceText = opts.serviceText.substring(
					current.serviceOffset,
					current.serviceOffset + length,
				)
				if (sourceText !== serviceText) {
					length = 0
					for (let i = 0; i < serviceOffset - current.serviceOffset; i++) {
						if (sourceText[i] === serviceText[i]) {
							length = i + 1
						} else {
							break
						}
					}
				}
				if (length > 0) {
					const lastMapping = mappings.length
						? mappings[mappings.length - 1]
						: undefined
					if (
						lastMapping &&
						lastMapping.serviceOffset + lastMapping.sourceLength ===
							current.serviceOffset &&
						lastMapping.sourceOffset + lastMapping.sourceLength ===
							current.sourceOffset
					) {
						lastMapping.sourceLength += length
					} else {
						mappings.push({
							sourceOffset: current.sourceOffset,
							serviceOffset: current.serviceOffset,
							sourceLength: length,
						})
					}
				}
				current = undefined
			}
			if (segment[2] != null && segment[3] != null) {
				const sourceOffset = getPositionOfColumnAndLine(sourceTextWithLineMap, {
					line: segment[2],
					column: segment[3],
				})
				current = {
					serviceOffset,
					sourceOffset,
				}
			}
		}
	}

	return mappings
}

interface SourceFileWithLineMap {
	readonly text: string;
	/** Used for caching the start positions of lines in `text` */
	lineMap?: readonly number[];
}

function computeLineStarts(source: string): readonly number[] {
	const res = [];
	let lineStart = 0;
	let cr = source.indexOf("\r", lineStart);
	let lf = source.indexOf("\n", lineStart);
	while (true) {
		if (lf === -1) {
			while (cr !== -1) {
				res.push(lineStart);
				lineStart = cr + 1;
				cr = source.indexOf("\r", lineStart);
			}
			break;
		}
		if (cr === -1) {
			while (lf !== -1) {
				res.push(lineStart);
				lineStart = lf + 1;
				lf = source.indexOf("\n", lineStart);
			}
			break;
		}
		if (cr + 1 === lf) {
			res.push(lineStart);
			lineStart = lf + 1;
			cr = source.indexOf("\r", lineStart);
			lf = source.indexOf("\n", lineStart);
			continue;
		}
		res.push(lineStart);
		if (cr < lf) {
			lineStart = cr + 1;
			cr = source.indexOf("\r", lineStart);
		} else {
			lineStart = lf + 1;
			lf = source.indexOf("\n", lineStart);
		}
	}
	res.push(lineStart);
	return res;
}

interface ColumnAndLine {
	/** 0-indexed */
	column: number;
	/** 0-indexed */
	line: number;
}


function getPositionOfColumnAndLine(
	source: SourceFileWithLineMap,
	columnAndLine: ColumnAndLine,
): number {
	if (typeof source === "string") {
		return computePositionOfColumnAndLine(
			source,
			computeLineStarts(source),
			columnAndLine,
		);
	}
	source.lineMap ??= computeLineStarts(source.text);
	return computePositionOfColumnAndLine(
		source.text,
		source.lineMap,
		columnAndLine,
	);
}

function computePositionOfColumnAndLine(
	sourceText: string,
	lineStarts: readonly number[],
	{ column, line }: ColumnAndLine,
): number {
	line = Math.min(Math.max(line, 0), lineStarts.length - 1);

	const res = lineStarts[line]! + column;
	if (line === lineStarts.length - 1) {
		return Math.min(res, sourceText.length);
	}
	return Math.min(res, lineStarts[line + 1]! - 1);
}

