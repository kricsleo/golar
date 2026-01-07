package vue_codegen

import (
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
)

// TODO: <script src="">

type scriptCodegenCtx struct {
	*codegenCtx
	scriptSetupEl *vue_ast.ElementNode
	scriptEl      *vue_ast.ElementNode
}

func generateScript(base *codegenCtx, scriptSetupEl *vue_ast.ElementNode, scriptEl *vue_ast.ElementNode, templateEl *vue_ast.ElementNode) {
	c := scriptCodegenCtx{
		codegenCtx:    base,
		scriptSetupEl: scriptSetupEl,
		scriptEl:      scriptEl,
	}

	// we don't import define* macros because they're globally available
	// https://github.com/vuejs/core/blob/aac7e1898907445c8f89b22047a9bfcf0a6e91b8/packages/runtime-core/types/scriptSetupHelpers.d.ts
	c.serviceText.WriteString("import { defineComponent as __VLS_DefineComponent } from 'vue'\n")

	var selfType string
	// TODO: revisit <script>
	if c.scriptEl != nil {
		if len(c.scriptEl.Children) != 1 {
			panic("TODO: len of <script> children != 1")
		}

		innerStart := c.scriptEl.InnerLoc.Pos()
		text := c.scriptEl.Children[0].AsText()

		mapStart := text.Loc.Pos()
		hasExportDefault := false

		for _, statement := range c.scriptEl.Ast.Statements.Nodes {
			if !ast.IsExportAssignment(statement) {
				continue
			}

			hasExportDefault = true
			export := statement.AsExportAssignment()
			c.mapText(mapStart, innerStart+export.Expression.Pos())
			c.serviceText.WriteString(" {} as unknown as typeof __VLS_Export\n")
			if c.scriptSetupEl == nil {
				c.serviceText.WriteString("const __VLS_Export = ")
				selfType = "__VLS_Export"
			} else {
				c.serviceText.WriteString("const __VLS_Self = ")
				selfType = "__VLS_Self"
			}
			mapStart = innerStart + export.Expression.Pos()

			break
		}

		c.mapText(mapStart, text.Loc.End())
		c.serviceText.WriteString("\n\n")

		if !hasExportDefault {
			c.serviceText.WriteString("const __VLS_Export = __VLS_DefineComponent({})\nexport default __VLS_Export\n")
		}

		// TODO: options wrapper - wrap export default |defineComponent(|{}|)|
	}

	// TODO: generic support
	if c.scriptSetupEl != nil {
		if len(c.scriptSetupEl.Children) != 1 {
			panic("TODO: len of <script setup> children != 1")
		}

		text := c.scriptSetupEl.Children[0].AsText()

		c.serviceText.WriteString("const __VLS_Export = (async () => {\n")
		innerStart := c.scriptSetupEl.InnerLoc.Pos()

		lastMappedPos := text.Loc.Pos()

		var (
			propsVariableName string
			emitsVariableName string
			slotsVariableName string
		)


		// TODO: report nested compiler macros (vue compiler errors on them)
		// TODO: $emits, $props, emitstoprops

		bindingRanges := []core.TextRange{}
		importRanges := []core.TextRange{}
		for _, statement := range c.scriptSetupEl.Ast.Statements.Nodes {
			switch statement.Kind {
			case ast.KindVariableStatement:
				for _, d := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
					decl := d.AsVariableDeclaration()
					name := decl.Name()
					var visitor ast.Visitor
					// TODO: binding pattern?
					// TODO: declare const?
					visitor = func(n *ast.Node) bool {
						if ast.IsIdentifier(n) {
							bindingRanges = append(bindingRanges, n.Loc)
						}
						return n.ForEachChild(visitor)
					}
					visitor(name)

					// TODO: report props destructuring?
					if !ast.IsIdentifier(name) {
						break
					}
					if decl.Initializer == nil || !ast.IsCallExpression(decl.Initializer) {
						break
					}

					call := decl.Initializer.AsCallExpression()
					callee := call.Expression
					if !ast.IsIdentifier(callee) {
						break
					}
					calleeName := callee.Text()
					switch calleeName {
					case "defineProps":
						if propsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineProps")
							break
						}
						propsVariableName = name.Text()
					case "defineEmits":
						if emitsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineEmits")
							break
						}
						emitsVariableName = name.Text()
					case "defineSlots":
						if slotsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineSlots")
							break
						}
						slotsVariableName = name.Text()
					}
				}
			case ast.KindExpressionStatement:
				expr := statement.AsExpressionStatement().Expression
				if !ast.IsCallExpression(expr) {
					break
				}
				call := expr.AsCallExpression()
				callee := call.Expression
				if !ast.IsIdentifier(callee) {
					break
				}
				calleeName := callee.Text()
				switch calleeName {
				case "defineProps":
					if propsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineProps")
						break
					}
					propsVariableName = "__VLS_Props"
					c.mapText(lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Props = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					lastMappedPos = innerStart + statement.End()
				case "defineEmits":
					if emitsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineEmits")
						break
					}
					emitsVariableName = "__VLS_Emits"
					c.mapText(lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Emits = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					lastMappedPos = innerStart + statement.End()
				case "defineSlots":
					if slotsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineSlots")
						break
					}
					slotsVariableName = "__VLS_Slots"
					c.mapText(lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Slots = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					lastMappedPos = innerStart + statement.End()
				}
			case ast.KindFunctionDeclaration, ast.KindClassDeclaration, ast.KindEnumDeclaration:
				if name := statement.Name(); name != nil {
					bindingRanges = append(bindingRanges, name.Loc)
				}
			case ast.KindImportDeclaration:
				importRanges = append(importRanges, core.NewTextRange(innerStart+statement.Loc.Pos(), innerStart+statement.Loc.End()))
				if lastMappedPos != statement.Pos() {
					c.mapText(lastMappedPos, innerStart+statement.Pos())
				}
				lastMappedPos = innerStart + statement.End()

				importClause := statement.AsImportDeclaration().ImportClause
				if importClause != nil {
					if importClause.Name() != nil {
						bindingRanges = append(bindingRanges, importClause.Name().Loc)
					}

					namedBindings := importClause.AsImportClause().NamedBindings
					if namedBindings != nil {
						if ast.IsNamespaceImport(namedBindings) {
							bindingRanges = append(bindingRanges, namedBindings.Name().Loc)
						} else {
							for _, element := range namedBindings.Elements() {
								bindingRanges = append(bindingRanges, element.Name().Loc)
							}
						}
					}
				}
			}
		}
		c.mapText(lastMappedPos, text.Loc.End())
		c.serviceText.WriteByte('\n')

		c.serviceText.WriteString("type __VLS_SetupExposed = {\n")
		// TODO: proxy refs
		for _, binding := range bindingRanges {
			c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
			c.serviceText.WriteString(": typeof ")
			c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
			c.serviceText.WriteRune('\n')
		}
		c.serviceText.WriteString("}\n")

		c.serviceText.WriteString("const __VLS_Ctx = {\n")
		c.serviceText.WriteString("...{} as unknown as __VLS_SetupExposed,\n")
		if propsVariableName != "" {
			c.serviceText.WriteString("...{} as unknown as typeof ")
			c.serviceText.WriteString(propsVariableName)
			c.serviceText.WriteString(",\n")
		}
		if selfType != "" {
			c.serviceText.WriteString("...{} as unknown as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(", new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as unknown as import('vue').ComponentPublicInstance,\n")
		}
		if propsVariableName != "" {
			// TODO: model props and emit props
			c.serviceText.WriteString("...{} as unknown as { $props: typeof ")
			c.serviceText.WriteString(propsVariableName)
			c.serviceText.WriteString("},\n")
		}
		c.serviceText.WriteString("}\n")

		generateTemplate(c.codegenCtx, templateEl)

		c.serviceText.WriteString("\nconst __VLS_Base = __VLS_DefineComponent({\n")
		if propsVariableName != "" {
			c.serviceText.WriteString("__typeProps: {} as unknown as typeof ")
			c.serviceText.WriteString(propsVariableName)
			c.serviceText.WriteString(",\n")
		}
		if emitsVariableName != "" {
			c.serviceText.WriteString("__typeEmits: {} as unknown as typeof ")
			c.serviceText.WriteString(emitsVariableName)
			c.serviceText.WriteString(",\n")
		}
		c.serviceText.WriteString("})\n")

		if slotsVariableName == "" {
			c.serviceText.WriteString("return __VLS_Base\n")
		} else {
			c.serviceText.WriteString("return {} as unknown as __VLS_WithSlots<typeof __VLS_Base, typeof ")
			c.serviceText.WriteString(slotsVariableName)
			c.serviceText.WriteString(">\n")
		}

		c.serviceText.WriteString("\n})()\n")
		for _, loc := range importRanges {
			c.mapText(loc.Pos(), loc.End())
			c.serviceText.WriteString("\n")
		}

		if c.scriptEl == nil {
			c.serviceText.WriteString("export default {} as unknown as Awaited<typeof __VLS_Export>\n")
		}
	}

	if c.scriptEl == nil && c.scriptSetupEl == nil {
		generateTemplate(c.codegenCtx, templateEl)
	}
}
