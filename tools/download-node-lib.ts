import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import stream from 'node:stream'
import { pipeline } from 'node:stream/promises'
import { devDir, repoRoot } from './utils.ts'

assert.equal(process.platform, 'win32', 'This script must run on Windows')
assert.ok(process.release.libUrl, 'process.release.libUrl is unavailable')

const outputFile = path.join(devDir, 'node.lib')

const { libUrl } = process.release
console.log(`Downloading ${libUrl} to ${path.relative(repoRoot, outputFile)}`)
const response = await fetch(libUrl)

if (!response.ok) {
	throw new Error(
		`Failed to download: ${response.status} ${response.statusText}`,
	)
}

await fs.promises.mkdir(devDir, { recursive: true })

if (!response.body) {
	throw new Error('Missing response body')
}

await pipeline(
	stream.Readable.fromWeb(response.body),
	fs.createWriteStream(outputFile),
)

console.log('Done')
