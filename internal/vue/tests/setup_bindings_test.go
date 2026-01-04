package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestSetupImportsBinding(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	import { useId/*1*/ } from 'vue'
</script>

<template>
	{{ useId/*2*/ }}
</template>
`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `(alias) function useId(): string`, "")
	f.VerifyQuickInfoAt(t, "2", `(property) useId: () => string`, "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}
