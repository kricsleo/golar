package vue_tests

import (
	"testing"

	"github.com/auvred/golar/internal/vue/codegen"
	"github.com/microsoft/typescript-go/shim/fourslash"
	"github.com/microsoft/typescript-go/shim/lsp/lsproto"
	"github.com/microsoft/typescript-go/shim/testutil"
)

func TestGlobalTypesNoErrors(t *testing.T) {
	t.Parallel()

	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	content := withVueNodeModules(t, "// @filename: types.ts\n" + vue_codegen.GlobalTypes)
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyNonSuggestionDiagnostics(t, []*lsproto.Diagnostic{})
}
