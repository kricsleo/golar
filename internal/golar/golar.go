package golar

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/auvred/golar/internal/mapping"
	"github.com/auvred/golar/internal/pluginhost"

	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/core"
	"github.com/microsoft/typescript-go/pkg/diagnostics"
	"github.com/microsoft/typescript-go/pkg/parser"
	"github.com/microsoft/typescript-go/pkg/tsoptions"
	"github.com/microsoft/typescript-go/pkg/tspath"
)

var pluginErrorDiagnostic = diagnostics.NewMessage(1_000_000, diagnostics.CategoryError, "plugin_error_diagnostic", "{0}")

type compilerHostProxy struct {
	compiler.CompilerHost
	config *tsoptions.ParsedCommandLine
}

type languageData struct {
	sourceText            string
	sourceMap             *mapping.SourceMap
	ignoreDirectives      []mapping.IgnoreDirectiveMapping
	expectErrorDirectives []mapping.ExpectErrorDirectiveMapping
}

func (h *compilerHostProxy) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	sourceText, ok := h.CompilerHost.FS().ReadFile(opts.FileName)
	if !ok {
		return nil
	}
	res := h.parseFile(opts, sourceText, core.GetScriptKindFromFileName(opts.FileName))
	if res != nil {
		return res
	}
	return h.CompilerHost.GetSourceFile(opts)
}

func wrapCompilerHost(config *tsoptions.ParsedCommandLine, host compiler.CompilerHost) compiler.CompilerHost {
	return &compilerHostProxy{host, config}
}

var pluginByExtension = map[string]*pluginhost.Plugin{}

func init() {
	plugins, ok := os.LookupEnv("GOLAR_PLUGINS")
	if !ok || plugins == "" {
		return
	}

	for pluginCommand := range strings.SplitSeq(plugins, "\x1e") {
		if pluginCommand == "" {
			continue
		}
		args := []string{}
		for arg := range strings.SplitSeq(pluginCommand, "\x1f") {
			args = append(args, arg)
		}
		if len(args) == 0 || args[0] == "" {
			continue
		}

		plugin, err := pluginhost.NewPlugin(args)
		if err != nil {
			panic(err)
		}
		for _, ext := range plugin.Extensions {
			tspath.RegisterSupportedExtension(ext.Extension)
			pluginByExtension[ext.Extension] = plugin
			if ext.AllowExtensionlessImports {
				tspath.RegisterSupportedExtensionless(ext.Extension)
			}
			if ext.StripFromDeclarationFileName {
				tspath.RegisterExtensionToRemove(ext.Extension)
			}
		}
	}
}

func (h *compilerHostProxy) parseFile(opts ast.SourceFileParseOptions, sourceText string, scriptKind core.ScriptKind) *ast.SourceFile {
	ext := filepath.Ext(opts.FileName)
	if plugin, ok := pluginByExtension[ext]; ok {
		resp := <-plugin.CreateServiceCode(h.GetCurrentDirectory(), h.config.ConfigName(), opts.FileName, sourceText)
		if resp.Errors != nil {
			file := ast.SourceFile{}
			file.SetText(sourceText)
			file.SetParseOptions(opts)
			diags := make([]*ast.Diagnostic, len(resp.Errors))
			for i, err := range resp.Errors {
				diags[i] = ast.NewDiagnostic(&file, err.Loc, pluginErrorDiagnostic, err.Message)
			}
			file.SetDiagnostics(diags)
			return &file
		}

		file := parser.ParseSourceFile(opts, resp.ServiceText, resp.ScriptKind)
		file.IsDeclarationFile = resp.DeclarationFile
		file.WrapDiagnostics = func(diags []*ast.Diagnostic) []*ast.Diagnostic {
			newFile := ast.SourceFile{}
			newFile.GolarLanguageData = file.GolarLanguageData
			newFile.SetText(sourceText)
			newFile.SetParseOptions(file.ParseOptions())
			return wrapDiagnostics(&newFile, diags, false, resp.IgnoreNotMappedDiagnostics)
		}
		file.WrapSemanticDiagnostics = func(diags []*ast.Diagnostic) []*ast.Diagnostic {
			// TODO: this is hack
			newFile := ast.SourceFile{}
			newFile.GolarLanguageData = file.GolarLanguageData
			newFile.SetText(sourceText)
			newFile.SetParseOptions(file.ParseOptions())
			return wrapDiagnostics(&newFile, diags, true, resp.IgnoreNotMappedDiagnostics)
		}
		langData := languageData{
			sourceText: sourceText,
		}
		langData.sourceMap = mapping.NewSourceMap(resp.Mappings)
		langData.ignoreDirectives = resp.IgnoreMappings
		langData.expectErrorDirectives = resp.ExpectErrorMappings
		file.GolarLanguageData = langData

		return file
	}

	return nil
}

func adjustDiagnostic(file *ast.SourceFile, diagnostic *ast.Diagnostic, dropUnmatched bool) *ast.Diagnostic {
	diagnostic.SetFile(file)
	for _, s := range diagnostic.MessageChain() {
		s.SetFile(file)
	}
	for _, s := range diagnostic.RelatedInformation() {
		s.SetFile(file)
	}
	if file.GolarLanguageData == nil || diagnostic.Code() >= 1_000_000 {
		return diagnostic
	}
	langData := file.GolarLanguageData.(languageData)
	for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(diagnostic.Pos()), uint32(diagnostic.End()), true) {
		diagnostic.SetLocation(core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)))
	MessageChain:
		for _, d := range diagnostic.MessageChain() {
			for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(d.Pos()), uint32(d.End()), true) {
				d.SetLocation(core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)))
				continue MessageChain
			}
			d.SetLocation(core.NewTextRange(0, 0))
		}
	RelatedInformation:
		for _, d := range diagnostic.RelatedInformation() {
			for _, sourceRange := range langData.sourceMap.ToSourceRange(uint32(d.Pos()), uint32(d.End()), true) {
				d.SetLocation(core.NewTextRange(int(sourceRange.MappedStart), int(sourceRange.MappedEnd)))
				continue RelatedInformation
			}
			d.SetLocation(core.NewTextRange(0, 0))
		}
		return diagnostic
	}

	if dropUnmatched {
		return nil
	}

	diagnostic.SetLocation(core.NewTextRange(0, 0))

	return diagnostic
}

func wrapDiagnostics(file *ast.SourceFile, diags []*ast.Diagnostic, collectUnused bool, dropUnmatched bool) []*ast.Diagnostic {
	res := []*ast.Diagnostic{}
	if file.GolarLanguageData == nil {
		return nil
	}
	langData := file.GolarLanguageData.(languageData)
	directiveMap := mapping.NewDirectiveMap(langData.ignoreDirectives, langData.expectErrorDirectives)
	for _, diag := range diags {
		if directiveMap.IsServiceRangeIgnored(diag.Loc()) {
			continue
		}
		adjusted := adjustDiagnostic(file, diag, dropUnmatched)
		if adjusted != nil {
			res = append(res, adjusted)
		}
	}
	if !collectUnused {
		return res
	}
	for _, loc := range directiveMap.CollectUnused() {
		res = append(res, ast.NewDiagnostic(file, loc, diagnostics.Unused_ts_expect_error_directive))
	}
	return res
}

// TODO: for hover and other LS methods we should analyze multiple mappings
// instead of returning the first mapping
func positionToService(file *ast.SourceFile, pos int) int {
	if file.GolarLanguageData == nil {
		return pos
	}

	langData := file.GolarLanguageData.(languageData)
	for _, serviceLoc := range langData.sourceMap.ToServiceLocation(uint32(pos)) {
		return int(serviceLoc.Offset)
	}
	return pos
}

func init() {
	compiler.GolarExt.WrapCompilerHost = wrapCompilerHost
}
