package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
)

func TestSetupImportsBinding(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	import { useCssModule/*1*/ } from 'vue'
</script>

<template>
	{{ useCssModule/*2*/ }}
</template>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(alias) function useCssModule(name?: string): Record<string, string>`, "")
		f.VerifyQuickInfoAt(t, "2", `(property) useCssModule: (name?: string) => Record<string, string>`, "")
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}

func TestSetupRefsUnwrap(t *testing.T) {
	runFourslashTest(t, `// @filename: file.vue
<script lang="ts" setup>
	import { ref } from 'vue'

	const foo = ref('123')
	const nestedRef = {
		bar: ref(123),
	}
</script>

<template>
	{{ foo/*1*/ }}
	{{ nestedRef.bar/*2*/ }}
</template>
`, func(t *testing.T, f *fourslash.FourslashTest, version vueVersion) {
		f.VerifyQuickInfoAt(t, "1", `(property) foo: string`, "")
		switch version {
		case vue_3_2, vue_3_3, vue_3_4:
			f.VerifyQuickInfoAt(t, "2", `(property) bar: Ref<number>`, "")
		default:
			f.VerifyQuickInfoAt(t, "2", `(property) bar: Ref<number, number>`, "")
		}
		f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
	})
}
