export function ms(ms: number): string {
	ms = Math.abs(ms)

	if (ms > 60_000) {
		const totalSeconds = Math.round(ms / 1_000)
		const minutes = Math.floor(totalSeconds / 60)
		return `${minutes}m${totalSeconds - minutes * 60}s`
	}

	if (ms > 1_000) {
		const total10ms = Math.round(ms / 10)
		const seconds = Math.floor(total10ms / 100)
		return `${seconds}s${total10ms - seconds * 100}ms`
	}

	if (ms > 10) {
		return `${Number(ms.toPrecision(3))}ms`
	}

	if (ms > 1) {
		const totalHundredthsMs = Math.round(ms * 100)
		const wholeMs = Math.floor(totalHundredthsMs / 100)
		return `${wholeMs}ms${totalHundredthsMs - wholeMs * 100}µs`
	}

	return `${Math.round(ms * 1_000)}µs`
}
