package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestDefineModel(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
// @strict: true
<script lang="ts" setup>
	const model1/*1*/ = defineModel<string>('first')

	const opts = { required: true } as const
	const model2/*3*/ = defineModel<string>('second', {
		...opts,
	})

	const model3/*5*/ = defineModel('third', {
		type: String,
		required: true,
		get(v) {
			return Number.parseInt(v)
		}
	})
</script>

<template>
	{{ first/*2*/ }}
	{{ second/*4*/ }}
	{{ third/*6*/ }}
	{{ $props.third/*7*/ }}
	{{ model3/*8*/ }}
</template>`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		switch version {
		case vue_3_2, vue_3_3:
			return
		case vue_3_4:
			f.VerifyQuickInfoAt(t, "1", `const model1: ModelRef<string | undefined, string>`, "")
			f.VerifyQuickInfoAt(t, "3", `const model2: ModelRef<string, string>`, "")
			f.VerifyQuickInfoAt(t, "5", `const model3: ModelRef<string, string>`, "")
			f.VerifyQuickInfoAt(t, "8", `(property) model3: string`, "")
		default:
			f.VerifyQuickInfoAt(t, "1", `const model1: ModelRef<string | undefined, string, string | undefined, string | undefined>`, "")
			f.VerifyQuickInfoAt(t, "3", `const model2: ModelRef<string, string, string, string>`, "")
			f.VerifyQuickInfoAt(t, "5", `const model3: ModelRef<string, string, number, string>`, "")
			f.VerifyQuickInfoAt(t, "8", `(property) model3: number`, "")
		}
		f.VerifyQuickInfoAt(t, "2", `(property) 'first': string | undefined`, "")
		f.VerifyQuickInfoAt(t, "4", `(property) 'second': string`, "")
		f.VerifyQuickInfoAt(t, "6", `(property) 'third': string`, "")
		f.VerifyQuickInfoAt(t, "7", `(property) 'third': string`, "")

		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
