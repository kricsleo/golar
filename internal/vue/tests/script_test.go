package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestScriptAndSetup(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts">
	const foo/*1*/ = 'foo'
</script>
<script lang="ts" setup>
	const bar = 'bar'
</script>

<template>
	<!-- TODO: -->
	<!-- {{ [|foo|] }} -->
	{{ bar/*2*/ }}
</template>
`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `const foo: "foo"`, "")
	f.VerifyQuickInfoAt(t, "2", `(property) bar: "bar"`, "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		// {
		// 	Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2339)},
		// 	Message: "Type 'number' is not assignable to type 'string'.",
		// },
	})
}
