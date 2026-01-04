package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestMissingComponentProps(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file-foo.vue
<script setup lang="ts">
	defineProps<{ foo: string }>()
</script>

// @filename: file.vue
<script setup lang="ts">
	import CompFoo from './file-foo.vue'
</script>

<template>
	<[|CompFoo|]/>
</template>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.GoToFile(t, "file.vue")
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code: &lsproto.IntegerOrString{Integer: ptrTo[int32](2345)},
			Message: `Argument of type '{}' is not assignable to parameter of type '{ readonly foo: string; } & VNodeProps & AllowedComponentProps & ComponentCustomProps'.
  Property 'foo' is missing in type '{}' but required in type '{ readonly foo: string; }'.`,
		},
	})
}
