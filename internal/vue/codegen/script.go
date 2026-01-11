package vue_codegen

import (
	"github.com/auvred/golar/internal/collections"
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
	lastMappedPos int

	seenDefineModels        collections.Set[string]
	modelPropsVariableNames []string
	modelEmitsVariableNames []string
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
	if c.scriptEl != nil {
		if len(c.scriptEl.Children) != 1 {
			panic("TODO: len of <script> children != 1")
		}

		// innerStart := c.scriptEl.InnerLoc.Pos()
		text := c.scriptEl.Children[0].AsText()

		c.lastMappedPos = text.Loc.Pos()

		// for _, statement := range c.scriptEl.Ast.Statements.Nodes {
		// 	if !ast.IsExportAssignment(statement) {
		// 		continue
		// 	}
		// 	if c.scriptSetupEl != nil {
		// 	}
		// 	// TODO: report export equals? (export = ...)
		//
		// 	export := statement.AsExportAssignment()
		// 	c.mapText(c.lastMappedPos, innerStart+export.Expression.Pos())
		// 	c.serviceText.WriteString(" {} as unknown as typeof __VLS_Export\n")
		// 	if c.scriptSetupEl == nil {
		// 		c.serviceText.WriteString("const __VLS_Export = ")
		// 		selfType = "__VLS_Export"
		// 	} else {
		// 		c.serviceText.WriteString("const __VLS_Self = ")
		// 		selfType = "__VLS_Self"
		// 	}
		// 	c.lastMappedPos = innerStart + export.Expression.Pos()
		//
		// 	break
		// }

		c.mapText(c.lastMappedPos, text.Loc.End())
		c.serviceText.WriteString("\n\n")

		// if !hasExportDefault {
		// 	c.serviceText.WriteString("const __VLS_Export = __VLS_DefineComponent({})\nexport default __VLS_Export\n")
		// }

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

		c.lastMappedPos = text.Loc.Pos()

		var (
			propsVariableName string
			emitsVariableName string
			slotsVariableName string
		)

		// TODO: report nested compiler macros (vue compiler errors on them)
		// TODO: report incorrect compiler macros arguments
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

					nameIsIdentifier := ast.IsIdentifier(name)
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
						// TODO: report props destructuring?
						if !nameIsIdentifier {
							break
						}
						if propsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineProps")
							break
						}
						propsVariableName = name.Text()
					case "defineEmits":
						// TODO: can there be destructuring
						if !nameIsIdentifier {
							break
						}
						if emitsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineEmits")
							break
						}
						emitsVariableName = name.Text()
					case "defineSlots":
						// TODO: can there be destructuring
						if !nameIsIdentifier {
							break
						}
						if !c.options.Version.supportsDefineSlots() {
							break
						}
						if slotsVariableName != "" {
							calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
							c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineSlots")
							break
						}
						slotsVariableName = name.Text()
					case "defineModel":
						if !c.options.Version.supportsDefineModel() {
							break
						}
						modelVariableName := c.newInternalVariable()
						callLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, call.AsNode())
						c.mapText(c.lastMappedPos, innerStart+callLoc.Pos())
						c.lastMappedPos = innerStart + callLoc.Pos()
						c.serviceText.WriteString("{} as unknown as typeof ")
						c.serviceText.WriteString(modelVariableName)
						c.processDefineModel(innerStart, call, callLoc, modelVariableName)
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
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Props = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
				case "defineEmits":
					if emitsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineEmits")
						break
					}
					emitsVariableName = "__VLS_Emits"
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Emits = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
				case "defineSlots":
					if !c.options.Version.supportsDefineSlots() {
						break
					}
					if slotsVariableName != "" {
						calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
						c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_X_0_call, "defineSlots")
						break
					}
					slotsVariableName = "__VLS_Slots"
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
					c.serviceText.WriteString("\nconst __VLS_Slots = ")
					c.mapText(innerStart+statement.Pos(), innerStart+statement.End())
					c.lastMappedPos = innerStart + statement.End()
				case "defineModel":
					modelVariableName := c.newInternalVariable()
					callLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, call.AsNode())
					c.mapText(c.lastMappedPos, innerStart+callLoc.Pos())
					c.lastMappedPos = innerStart + callLoc.Pos()
					c.processDefineModel(innerStart, call, callLoc, modelVariableName)
				}
			case ast.KindFunctionDeclaration, ast.KindClassDeclaration, ast.KindEnumDeclaration:
				if name := statement.Name(); name != nil {
					bindingRanges = append(bindingRanges, name.Loc)
				}
			case ast.KindImportDeclaration:
				importRanges = append(importRanges, core.NewTextRange(innerStart+statement.Loc.Pos(), innerStart+statement.Loc.End()))
				if c.lastMappedPos != statement.Pos() {
					c.mapText(c.lastMappedPos, innerStart+statement.Pos())
				}
				c.lastMappedPos = innerStart + statement.End()

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
		c.mapText(c.lastMappedPos, text.Loc.End())
		c.serviceText.WriteByte('\n')

		c.serviceText.WriteString("type __VLS_SetupExposed = import('vue').ShallowUnwrapRef<{\n")
		for _, binding := range bindingRanges {
			c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
			c.serviceText.WriteString(": typeof ")
			c.serviceText.WriteString(c.sourceText[innerStart+binding.Pos() : innerStart+binding.End()])
			c.serviceText.WriteRune('\n')
		}
		c.serviceText.WriteString("}>\n")

		publicPropsStarted := false
		startPublicProps := func() bool {
			if publicPropsStarted {
				return false
			}
			c.serviceText.WriteString("\ntype __VLS_PublicProps = ")
			publicPropsStarted = true
			return true
		}
		if propsVariableName != "" {
			startPublicProps()
			c.serviceText.WriteString("typeof ")
			c.serviceText.WriteString(propsVariableName)
		}
		for i, varName := range c.modelPropsVariableNames {
			if i > 0 || !startPublicProps() {
				c.serviceText.WriteString(" & ")
			}
			c.serviceText.WriteString(varName)
		}

		publicEmitsStarted := false
		startPublicEmits := func() bool {
			if publicEmitsStarted {
				return false
			}
			c.serviceText.WriteString("\ntype __VLS_PublicEmits = ")
			publicEmitsStarted = true
			return true
		}
		if emitsVariableName != "" {
			startPublicEmits()
			c.serviceText.WriteString("typeof ")
			c.serviceText.WriteString(emitsVariableName)
		}
		for i, varName := range c.modelEmitsVariableNames {
			if i > 0 || !startPublicEmits() {
				c.serviceText.WriteString(" & ")
			}
			c.serviceText.WriteString("typeof ")
			c.serviceText.WriteString(varName)
		}

		c.serviceText.WriteString("\nconst __VLS_Ctx = {\n")
		if selfType != "" {
			c.serviceText.WriteString("...{} as unknown as InstanceType<__VLS_PickNotAny<typeof ")
			c.serviceText.WriteString(selfType)
			c.serviceText.WriteString(", new () => {}>>,\n")
		} else {
			c.serviceText.WriteString("...{} as unknown as import('vue').ComponentPublicInstance,\n")
		}
		c.serviceText.WriteString("...{} as unknown as __VLS_SetupExposed,\n")
		if publicPropsStarted {
			c.serviceText.WriteString("...{} as unknown as __VLS_PublicProps,\n")
			// TODO: $emits and other $s
			c.serviceText.WriteString("...{} as unknown as { $props: __VLS_PublicProps },\n")
		}
		c.serviceText.WriteString("}\n")

		generateTemplate(c.codegenCtx, templateEl)

		c.serviceText.WriteString("\nconst __VLS_Base = __VLS_DefineComponent({\n")
		// TODO: withDefaults
		// TODO: defineProps(arg)
		if publicPropsStarted {
			if c.options.Version.supportsTypeProps() {
				c.serviceText.WriteString("__typeProps: {} as unknown as __VLS_PublicProps,\n")
			} else {
				c.serviceText.WriteString("props: {} as unknown as __VLS_TypePropsToOption<__VLS_PublicProps>,\n")
			}
		}
		if publicEmitsStarted {
			if c.options.Version.supportsTypeEmits() {
				c.serviceText.WriteString("__typeEmits: {} as unknown as __VLS_PublicEmits,\n")
			} else {
				c.serviceText.WriteString("emits: {} as unknown as __VLS_NormalizeEmits<__VLS_PublicEmits>,\n")
			}
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

// TODO: reconcile with vuejs/core/packages/compiler-sfc/src/script/defineModel.ts
func (c *scriptCodegenCtx) parseDefineModel(expr *ast.CallExpression) string {
	var name string
	if len(expr.Arguments.Nodes) >= 1 {
		if ast.IsStringLiteral(expr.Arguments.Nodes[0]) {
			name = expr.Arguments.Nodes[0].AsStringLiteral().Text
		}
	}

	return name
}

func (c *scriptCodegenCtx) processDefineModel(innerStart int, call *ast.CallExpression, callLoc core.TextRange, modelVariableName string) {
	modelName := c.parseDefineModel(call)
	c.serviceText.WriteString("\nconst ")
	c.serviceText.WriteString(modelVariableName)
	c.serviceText.WriteString(" = ")
	c.mapText(innerStart+callLoc.Pos(), innerStart+callLoc.End())
	c.lastMappedPos = innerStart + callLoc.End()
	modelTypesVariableName := c.newInternalVariable()
	c.serviceText.WriteString("\ntype ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString(" = typeof ")
	c.serviceText.WriteString(modelVariableName)
	c.serviceText.WriteString(" extends import('vue').ModelRef<infer T, infer M extends string | number | symbol")
	if c.options.Version.modelRefHasGetterAndSetter() {
		c.serviceText.WriteString(", any, any")
	}
	c.serviceText.WriteString("> ? [T, M] : never\n")
	modelPropTypeVariableName := c.newInternalVariable()
	c.modelPropsVariableNames = append(c.modelPropsVariableNames, modelPropTypeVariableName)
	c.serviceText.WriteString("type ")
	c.serviceText.WriteString(modelPropTypeVariableName)
	c.serviceText.WriteString(" = (undefined extends ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0] ? { '")
	// TODO: don't use quotes when not needed
	// TODO: escape
	camelizedModelName := "modelValue"
	camelizedModelNameForModifiers := "model"
	if modelName == "" {
		c.serviceText.WriteString(camelizedModelName)
	} else {
		camelizedModelName = camelize(modelName, &c.serviceText)
		camelizedModelNameForModifiers = camelizedModelName
	}
	c.serviceText.WriteString("'?: ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0] } : { '")
	c.serviceText.WriteString(camelizedModelName)
	c.serviceText.WriteString("': ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0] }) & { '")
	c.serviceText.WriteString(camelizedModelNameForModifiers)
	c.serviceText.WriteString("Modifiers'?: Partial<Record<")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[1], true>> }\n")

	modelEmitTypeVariableName := c.newInternalVariable()
	c.modelEmitsVariableNames = append(c.modelEmitsVariableNames, modelEmitTypeVariableName)
	c.serviceText.WriteString("const ")
	c.serviceText.WriteString(modelEmitTypeVariableName)
	// TODO: escape?
	c.serviceText.WriteString(" = defineEmits<{ 'update:")
	c.serviceText.WriteString(camelizedModelName)
	c.serviceText.WriteString("': [value: ")
	c.serviceText.WriteString(modelTypesVariableName)
	c.serviceText.WriteString("[0]] }>()\n")

	if c.seenDefineModels.Has(camelizedModelName) {
		callee := call.Expression
		calleeLoc := utils.TrimNodeTextRange(c.scriptSetupEl.Ast, callee)
		c.reportDiagnostic(core.NewTextRange(innerStart+calleeLoc.Pos(), innerStart+calleeLoc.End()), vue_diagnostics.Duplicate_model_name_X_0, camelizedModelName)
	} else {
		c.seenDefineModels.Add(camelizedModelName)
	}
}
