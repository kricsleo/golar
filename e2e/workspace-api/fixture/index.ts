interface User {
	name: string
	age: number
}

declare const unionValue: string | number

import { Foo } from './extra.ts'

declare const foo: typeof Foo
