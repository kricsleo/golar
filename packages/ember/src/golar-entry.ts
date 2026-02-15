import process from 'node:process'
import path from 'node:path'

export function getGolarEntry() {
	return [process.execPath, path.join(import.meta.dirname, 'bin.js')]
}
