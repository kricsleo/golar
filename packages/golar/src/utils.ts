import process from 'node:process'
import util from 'node:util'

function checkCi(key: string): boolean {
	return (
		key in process.env &&
		process.env[key] !== '0' &&
		process.env[key] !== 'false'
	)
}
export const isInCi = checkCi('CI') || checkCi('CONTINUOUS_INTEGRATION')

export const isColorOutput =
	process.env.FORCE_COLOR !== '0' && process.env.NO_COLOR !== '1' && !isInCi

export function styleText(
	format: util.InspectColor | readonly util.InspectColor[],
	text: string,
): string {
	return isColorOutput ? util.styleText(format, text) : text
}
