import process from 'node:process'
import { fileURLToPath } from 'node:url'

export function getGolarEntry() {
	return [
		fileURLToPath(
			import.meta.resolve(
				`@golar/astro-${process.platform}-${process.arch}/golar-astro${process.platform === 'win32' ? '.exe' : ''}`,
			),
		),
	]
}
