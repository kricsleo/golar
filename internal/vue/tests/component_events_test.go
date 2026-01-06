package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestComponentEventEmitType(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<CompFoo @foo="id => id/*1*/"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'foo', id: number): void }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `(parameter) id: number`, "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestComponentEventFunctionSetupVars(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'

	const foo = 'bar'
</script>

<template>
	<CompFoo @foo="() => foo/*1*/"/>
</template>

// @filename: file-foo.vue
<script setup lang="ts">
	defineEmits<{ (e: 'foo'): void }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `const foo: "bar"`, "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}
