package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestComponentSlots(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo v-slot="{ msg }">
		{{ msg/*1*/ }}
	</CompFoo>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineSlots<{
		default(props: { msg: "hello" }): any
	}>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `const msg: "hello"`, "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}
