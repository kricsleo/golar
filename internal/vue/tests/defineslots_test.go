package vue_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestDuplicateDefineSlotsCallExpression(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	const p = defineSlots<{ default(props: { msg: string }): any}>()
	[|defineSlots|]<{ default(props: { msg: string }): any}>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_006)},
			Message: "Duplicate defineSlots call.",
		},
	})
}

func TestDuplicateDefineSlotsVariableDecl(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, `// @filename: file.vue
<script lang="ts" setup>
	defineSlots<{ default(props: { msg: string }): any }>()
	const p = [|defineSlots|]<{ default(props: { msg: string }): any }>()
</script>`)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()

	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{
		{
			Code:    &lsproto.IntegerOrString{Integer: ptrTo[int32](1_000_006)},
			Message: "Duplicate defineSlots call.",
		},
	})
}
