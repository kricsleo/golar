import path from 'node:path'
import stream from 'node:stream'
import fs from 'node:fs/promises'
import * as tar from 'tar'

const VERSION = '1.8.0'
const ARCHIVE_URL = `https://github.com/nodejs/node-api-headers/archive/refs/tags/v${VERSION}.tar.gz`
const ARCHIVE_ROOT = `node-api-headers-${VERSION}`

const repoRoot = path.join(import.meta.dirname, '..')
const outDir = path.join(repoRoot, 'napi-include')
const includePrefix = `${ARCHIVE_ROOT}/include`
const includePrefixWithSlash = `${includePrefix}/`

await fs.mkdir(outDir, { recursive: true })

console.log(`Downloading ${ARCHIVE_URL}`)
const response = await fetch(ARCHIVE_URL)
if (!response.ok || !response.body) {
	throw new Error(
		`Failed to download ${ARCHIVE_URL} (status ${response.status})`,
	)
}

await stream.promises.pipeline(
	stream.Readable.fromWeb(response.body),
	tar.x({
		cwd: outDir,
		gzip: true,
		strip: 2,
		filter: (entryPath) =>
			entryPath === includePrefix ||
			entryPath.startsWith(includePrefixWithSlash),
	}),
)

console.log(`Extracted ${ARCHIVE_ROOT}/include to ${outDir}`)
