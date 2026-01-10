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
