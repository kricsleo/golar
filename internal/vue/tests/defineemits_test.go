package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestDefineEmitsVariableDecl(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
const emit = defineEmits<{ (e: 'foo', id: number): void  }>()
</script>

<template>
	{{ emit/*1*/ }}
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "1", `(property) emit: (e: "foo", id: number) => void`, "")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}

func TestDefineEmitsTypeMismatch(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
const emit = defineEmits<{ (e: 'foo', id: number): void  }>()
</script>

<template>
	{{ emit('foo', [|'2'|]) }}
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
			Message: "Argument of type 'string' is not assignable to parameter of type 'number'.",
		},
	})
}
func TestDuplicateDefineEmitsCallExpression(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	const e = defineEmits<{ (e: 'foo'): void }>()
	[|defineEmits|]<{ (e: 'foo'): void }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
			Message: "Duplicate defineEmits call.",
		},
	})
}

func TestDuplicateDefineEmitsVariableDecl(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	defineEmits<{ (e: 'foo'): void }>()
	const e = [|defineEmits|]<{ (e: 'foo'): void }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_007)},
			Message: "Duplicate defineEmits call.",
		},
	})
}
