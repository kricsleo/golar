const literalString = 'hello' as const
const literalNumber = 42 as const
const literalBoolean = true as const
const literalBigInt = 123n as const

interface Box<T, U = number> {
	value: T
	other: U
}

const arr: Array<number> = [1, 2, 3]
const tuple: readonly [number, string?, ...boolean[]] = [1]

type KeyOf<T> = keyof T
type Lookup<T, K extends keyof T> = T[K]
type Cond<T> = T extends string ? 'yes' : 'no'
const tpl: `hello ${string}` = 'hello world'
type Upper<T extends string> = Uppercase<T>
