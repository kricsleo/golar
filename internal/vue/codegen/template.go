package vue_codegen

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/auvred/golar/internal/collections"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/internal/vue/ast"
	"github.com/auvred/golar/internal/vue/diagnostics"
	"github.com/auvred/golar/internal/vue/parser"
	"github.com/microsoft/typescript-go/shim/ast"
)

type templateCodegenCtx struct {
	*codegenCtx
	scopes             []collections.Set[string]
	parentComponentVar string
	condChain          conditionalChain
}

func newTemplateCodegenCtx(base *codegenCtx) templateCodegenCtx {
	return templateCodegenCtx{
		codegenCtx: base,
	}
}

func generateTemplate(base *codegenCtx, el *vue_ast.ElementNode) {
	c := newTemplateCodegenCtx(base)
	if el != nil {
		c.visit(el.AsNode())
	}
}

func (c *templateCodegenCtx) enterScope() {
	c.scopes = append(c.scopes, collections.Set[string]{})
}
func (c *templateCodegenCtx) exitScope() {
	if len(c.scopes) > 0 {
		c.scopes = c.scopes[:len(c.scopes)-1]
	}
}
func (c *templateCodegenCtx) declareScopeVar(name string) {
	if len(c.scopes) > 0 {
		c.scopes[len(c.scopes)-1].Add(name)
	}
}

func (c *templateCodegenCtx) shouldPrefixIdentifier(identifier *ast.Node) bool {
	name := identifier.Text()

	for location := identifier; location != nil; location = location.Parent {
		locals := location.Locals()
		if _, ok := locals[name]; ok {
			return false
		}
	}

	for _, scope := range c.scopes {
		if scope.Has(name) {
			return false
		}
	}

	return true
}

type conditionalChain uint8

const (
	conditionalChainNone conditionalChain = iota
	conditionalChainValid
	conditionalChainBroken
)

func (c *templateCodegenCtx) visit(el *vue_ast.Node) {
	switch el.Kind {
	case vue_ast.KindElement:
		elem := el.AsElement()

		var conditionalDirective *vue_ast.DirectiveNode
		var forDirective *vue_ast.ForParseResult
		var slotDirective *vue_ast.DirectiveNode
		var seenProps collections.Set[string]
		hasSeenConditionalDirective := false

		// TODO: unexpected props and directives, for example on <template>
		for _, p := range elem.Props {
			if p.Kind != vue_ast.KindDirective {
				attr := p.AsAttribute()
				if seenProps.Has(attr.Name) {
					c.reportDiagnostic(attr.NameLoc, vue_diagnostics.Elements_cannot_have_multiple_X_0_with_the_same_name, "attributes")
				} else {
					seenProps.Add(attr.Name)
				}
				continue
			}
			dir := p.AsDirective()
			if seenProps.Has(dir.RawName) {
				c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Elements_cannot_have_multiple_X_0_with_the_same_name, "directives")
				continue
			} else {
				seenProps.Add(dir.RawName)
			}
			switch dir.Name {
			case "if":
				if hasSeenConditionalDirective {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
					break
				}
				hasSeenConditionalDirective = true
				c.condChain = conditionalChainValid
				conditionalDirective = dir
			case "else-if":
				if hasSeenConditionalDirective {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
					break
				}
				hasSeenConditionalDirective = true
				switch c.condChain {
				case conditionalChainNone:
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else-if")
					c.condChain = conditionalChainBroken
				case conditionalChainValid:
					conditionalDirective = dir
				}
			case "else":
				if hasSeenConditionalDirective {
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.Multiple_conditional_directives_cannot_coexist_on_the_same_element)
					break
				}
				hasSeenConditionalDirective = true
				switch c.condChain {
				case conditionalChainNone:
					c.reportDiagnostic(dir.NameLoc, vue_diagnostics.X_0_has_no_adjacent_v_if_or_v_else_if, "v-else")
				case conditionalChainValid:
					c.condChain = conditionalChainNone
					conditionalDirective = dir
				}
			case "for":
				forDirective = dir.ForParseResult
			// TODO: #slot
			case "slot":
				slotDirective = dir
			}
		}
		if conditionalDirective != nil {
			switch conditionalDirective.Name {
			case "else-if":
				c.serviceText.WriteString("else ")
				fallthrough
			case "if":
				c.serviceText.WriteString("if (")
				if conditionalDirective.Expression != nil && conditionalDirective.Expression.Ast != nil {
					c.mapExpressionInNonBindingPosition(conditionalDirective.Expression)
				} else {
					c.reportDiagnostic(conditionalDirective.Loc, vue_diagnostics.X_0_is_missing_expression, conditionalDirective.RawName)
					c.serviceText.WriteString("1 as number")
				}
				c.serviceText.WriteString(") {\n")
			case "else":
				c.serviceText.WriteString("else {\n")
			}
		} else if !hasSeenConditionalDirective {
			c.condChain = conditionalChainNone
		}
		if forDirective != nil {
			c.enterScope()
			c.serviceText.WriteString("{\nconst [")
			if forDirective.Value != nil {
				c.mapExpressionInBindingPosition(forDirective.Value)
			}
			c.serviceText.WriteString(",")
			if forDirective.Key != nil {
				c.mapExpressionInBindingPosition(forDirective.Key)
			}
			c.serviceText.WriteString(",")
			if forDirective.Index != nil {
				c.mapExpressionInBindingPosition(forDirective.Index)
			}
			c.serviceText.WriteString("] = __VLS_vFor(")
			c.mapExpressionInNonBindingPosition(forDirective.Source)
			c.serviceText.WriteString(")\n")
		}

		// TODO: handle template and component
		isComponent := elem.Tag != "template" && elem.Tag != "component" && (isBuiltInComponent(elem.Tag) || !isNativeElement(elem.Tag))
		var ctxVar string
		// TODO: don't generate unused vars
		// TODO: expression components like foo.bar
		// TODO: global components
		// TODO: self component
		// TODO: component name casing
		// TODO: native elements
		if isComponent {
			ctxVar = c.newInternalVariable()
			propsVar := c.newInternalVariable()
			componentVar := c.newInternalVariable()
			vnodeVar := c.newInternalVariable()
			functionalVar := c.newInternalVariable()
			emitsVar := ""

			c.serviceText.WriteString("let ")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString("!: __VLS_ExtractComponentType<'")
			c.serviceText.WriteString(elem.Tag)
			c.serviceText.WriteString("', __VLS_SetupExposed, void, '")
			c.serviceText.WriteString(elem.Tag)
			c.serviceText.WriteString("'>['")
			// TODO: mapping?
			c.serviceText.WriteString(elem.Tag)
			c.serviceText.WriteString("']\n")

			c.serviceText.WriteString("const ")
			c.serviceText.WriteString(vnodeVar)
			c.serviceText.WriteString(" = ({} as unknown as typeof ")
			c.serviceText.WriteString(functionalVar)
			c.serviceText.WriteString(")")
			propsStart := c.serviceText.Len() + 1
			c.serviceText.WriteString("({\n")
			for _, prop := range elem.Props {
				switch prop.Kind {
				case vue_ast.KindAttribute:
					attr := prop.AsAttribute()
					propNameStart := c.serviceText.Len()
					c.serviceText.WriteByte('\'')
					camelize(attr.Name, &c.serviceText)
					propNameEnd := c.serviceText.Len() + 1
					c.mapRange(attr.Loc.Pos(), attr.Loc.Pos()+len(attr.Name), propNameStart, propNameEnd)
					if attr.Value == nil {
						c.serviceText.WriteString("': true,\n")
					} else {
						c.serviceText.WriteString("': '")
						// TODO: perf
						// TODO: escape
						for _, r := range attr.Value.Content {
							switch r {
							case '\\':
								c.serviceText.WriteString("\\x5c")
							case '\n':
								c.serviceText.WriteString("\\x0a")
							case '\'':
								c.serviceText.WriteString("\\x27")
							default:
								c.serviceText.WriteRune(r)
							}
						}
						c.serviceText.WriteString("',\n")
					}
				}
			}
			c.serviceText.WriteString("})\n")
			propsEnd := c.serviceText.Len() - 2
			// TODO: is this valid?
			tagStart := elem.Loc.Pos() + 1
			c.mapRange(tagStart, tagStart+len(elem.Tag), propsStart, propsEnd)

			// TODO: generic support
			c.serviceText.WriteString("const ")
			c.serviceText.WriteString(functionalVar)
			c.serviceText.WriteString(" = __VLS_AsFunctionalComponent(")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString(", new ")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString("({\n")
			// TODO: props here
			c.serviceText.WriteString("}))\n")

			// TODO: emits type mismatch mapping locations
			for _, prop := range elem.Props {
				if prop.Kind != vue_ast.KindDirective {
					continue
				}

				dir := prop.AsDirective()
				// TODO: model
				// TODO: dynamic event name
				if dir.Name != "on" || !dir.IsStatic {
					continue
				}
				if emitsVar == "" {
					emitsVar = c.newInternalVariable()
					c.serviceText.WriteString("var ")
					c.serviceText.WriteString(emitsVar)
					c.serviceText.WriteString("!: __VLS_ResolveEmits<typeof ")
					c.serviceText.WriteString(componentVar)
					c.serviceText.WriteString(", typeof ")
					c.serviceText.WriteString(ctxVar)
					c.serviceText.WriteString(".emit>\n")
				}

				// TODO: model & vue:
				c.serviceText.WriteString("const ")
				c.serviceText.WriteString(c.newInternalVariable())
				c.serviceText.WriteString(": __VLS_NormalizeComponentEvent<typeof ")
				c.serviceText.WriteString(propsVar)
				c.serviceText.WriteString(", typeof ")
				c.serviceText.WriteString(emitsVar)
				c.serviceText.WriteString(", '")
				camelize("on-"+dir.Arg, &c.serviceText) // propName
				c.serviceText.WriteString("', '")
				emitName := dir.Arg
				c.serviceText.WriteString(emitName)
				c.serviceText.WriteString("', '")
				camelize(emitName, &c.serviceText) // camelizedEmitName
				c.serviceText.WriteString("'> = {\n")
				emitNameStart := c.serviceText.Len()
				c.serviceText.WriteString("'on")
				// TODO(perf): no unnecessary allocations
				camelize(strings.Title(emitName), &c.serviceText)
				c.mapRange(dir.Loc.Pos(), dir.Loc.Pos()+len(dir.RawName), emitNameStart, c.serviceText.Len()+1)
				c.serviceText.WriteString("': ")
				if dir.Expression == nil || dir.Expression.Ast == nil {
					c.serviceText.WriteString("() => {}")
				} else {
					isCompound := true
					if len(dir.Expression.Ast.Statements.Nodes) == 0 {
						panic("Expected event listener AST to have at least one statement")
					}
					if len(dir.Expression.Ast.Statements.Nodes) == 1 {
						if ast.IsExpressionStatement(dir.Expression.Ast.Statements.Nodes[0]) {
							expr := ast.SkipParentheses(dir.Expression.Ast.Statements.Nodes[0].AsExpressionStatement().Expression)
							if ast.IsArrowFunction(expr) || ast.IsIdentifier(expr) || ast.IsPropertyAccessExpression(expr) || ast.IsFunctionExpression(expr) {
								isCompound = false
							}
						}
					}

					if isCompound {
						c.serviceText.WriteString("(...[$event]) => {\n")
						c.enterScope()
						c.declareScopeVar("$event")
						c.mapExpressionInNonBindingPosition(dir.Expression)
						c.exitScope()
						c.serviceText.WriteString("}\n")
						// TODO: condition guards
					} else {
						c.mapExpressionInNonBindingPosition(dir.Expression)
					}
				}
				c.serviceText.WriteString("\n}\n")
			}

			c.serviceText.WriteString("var ")
			c.serviceText.WriteString(ctxVar)
			c.serviceText.WriteString("!: __VLS_FunctionalComponentCtx<typeof ")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString(", typeof ")
			c.serviceText.WriteString(vnodeVar)
			c.serviceText.WriteString(">\n")

			c.serviceText.WriteString("var ")
			c.serviceText.WriteString(propsVar)
			c.serviceText.WriteString("!: __VLS_FunctionalComponentProps<typeof ")
			c.serviceText.WriteString(componentVar)
			c.serviceText.WriteString(", typeof ")
			c.serviceText.WriteString(vnodeVar)
			c.serviceText.WriteString(">\n")
		}

		// TODO: implicit default slot?
		// TODO: report duplicate slots
		if slotDirective != nil {
			parentComponentCtx := ctxVar
			if parentComponentCtx == "" {
				parentComponentCtx = c.parentComponentVar
			}
			if parentComponentCtx == "" {
				c.reportDiagnostic(slotDirective.Loc.WithEnd(slotDirective.Loc.Pos()+len(slotDirective.RawName)), vue_diagnostics.Slot_does_not_belong_to_the_parent_component)
			} else if slotDirective.Expression != nil {
				c.enterScope()
				slotVar := c.newInternalVariable()
				c.serviceText.WriteString("{\nconst { ")
				if slotDirective.Arg == "" {
					c.serviceText.WriteString("default: ")
				} else {
					// TODO: dynamic name
					c.serviceText.WriteByte('\'')
					c.serviceText.WriteString(slotDirective.Arg)
					c.serviceText.WriteString("': ")
				}
				c.serviceText.WriteString(slotVar)
				c.serviceText.WriteString("} = ")

				c.serviceText.WriteString(parentComponentCtx)
				c.serviceText.WriteString(".slots!\nconst [")
				c.mapExpressionInBindingPosition(slotDirective.Expression)
				c.serviceText.WriteString("] = __VLS_vSlot(")
				c.serviceText.WriteString(slotVar)
				c.serviceText.WriteString(")\n")
			}
		}

		currCondChain := c.condChain
		c.condChain = conditionalChainNone
		currParentComponentVar := c.parentComponentVar
		c.parentComponentVar = ctxVar
		for _, child := range elem.Children {
			c.visit(child)
		}
		c.parentComponentVar = currParentComponentVar
		c.condChain = currCondChain

		if slotDirective != nil && slotDirective.Expression != nil {
			c.exitScope()
			c.serviceText.WriteString("}\n")
		}
		if forDirective != nil {
			c.exitScope()
			c.serviceText.WriteString("}\n")
		}
		if conditionalDirective != nil {
			c.serviceText.WriteString("}\n")
		}
	case vue_ast.KindInterpolation:
		interpolation := el.AsInterpolation()
		c.serviceText.WriteString(";( ")
		c.mapExpressionInNonBindingPosition(interpolation.Content)
		c.serviceText.WriteString(" )\n")
	}
}

type expressionMapper struct {
	*templateCodegenCtx
	expr          *vue_ast.SimpleExpressionNode
	innerStart    int
	lastMappedPos int
	typeOnly      bool
}

func newExpressionMapper(c *templateCodegenCtx, expr *vue_ast.SimpleExpressionNode) expressionMapper {
	return expressionMapper{
		templateCodegenCtx: c,
		expr:               expr,
		innerStart:         expr.Loc.Pos() - expr.PrefixLen,
		lastMappedPos:      expr.Loc.Pos(),
	}
}

func (m *expressionMapper) mapTextToNodePos(pos int) {
	pos += m.innerStart
	m.mapText(m.lastMappedPos, pos)
	m.lastMappedPos = pos
}

func (m *expressionMapper) shouldPrefixIdentifier(identifier *ast.Node) bool {
	if m.typeOnly {
		return false
	}
	return m.templateCodegenCtx.shouldPrefixIdentifier(identifier)
}

func (c *templateCodegenCtx) mapExpressionInNonBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		// TODO: report non-binding cases
		if ast.IsExpressionStatement(firstStmt) {
			expr := firstStmt.AsExpressionStatement().Expression
			if ast.IsParenthesizedExpression(expr) {
				m.mapInNonBindingPosition(expr.AsParenthesizedExpression().Expression)
				goto FinalizeMapping
			}
		}

		// TODO: iter statements?
		m.mapInNonBindingPosition(firstStmt)
	}
FinalizeMapping:
	m.mapTextToNodePos(expr.Ast.End() - expr.SuffixLen)
}
func (c *templateCodegenCtx) mapExpressionInBindingPosition(expr *vue_ast.SimpleExpressionNode) {
	m := newExpressionMapper(c, expr)
	if len(expr.Ast.Statements.Nodes) > 0 {
		firstStmt := expr.Ast.Statements.Nodes[0]
		// TODO: report non-binding cases
		if ast.IsExpressionStatement(firstStmt) {
			expr := firstStmt.AsExpressionStatement().Expression
			if ast.IsArrowFunction(expr) {
				fn := expr.AsArrowFunction()
				if len(fn.Parameters.Nodes) == 1 && ast.IsParameter(fn.Parameters.Nodes[0]) {
					m.mapInBindingPosition(fn.Parameters.Nodes[0].AsParameterDeclaration().Name())
				}
			}
		}
	}
	m.mapTextToNodePos(expr.Ast.End() - expr.SuffixLen)
}

func (m *expressionMapper) mapInBindingPosition(node *ast.BindingName) bool {
	switch node.Kind {
	case ast.KindIdentifier:
		m.declareScopeVar(node.AsIdentifier().Text)
	case ast.KindArrayBindingPattern, ast.KindObjectBindingPattern:
		for _, elem := range node.AsBindingPattern().Elements.Nodes {
			bindingElem := elem.AsBindingElement()
			if visit(m.mapInNonBindingPositionIfNotIdentifier, bindingElem.PropertyName) ||
				visit(m.mapInBindingPosition, bindingElem.Name()) ||
				visit(m.mapInNonBindingPosition, bindingElem.Initializer) {
				return true
			}
		}
	}
	return false
}

func visit(v ast.Visitor, node *ast.Node) bool {
	if node != nil {
		return v(node)
	}
	return false
}
func visitNodeList(v ast.Visitor, nodeList *ast.NodeList) bool {
	if nodeList == nil {
		return false
	}
	return slices.ContainsFunc(nodeList.Nodes, v)
}

func (m *expressionMapper) withTypeOnlyVisit(fn func() bool) bool {
	before := m.typeOnly
	m.typeOnly = true
	res := fn()
	m.typeOnly = before
	return res
}
func (m *expressionMapper) typeOnlyVisit(node *ast.Node) bool {
	return m.withTypeOnlyVisit(func() bool {
		return visit(m.mapInNonBindingPosition, node)
	})
}
func (m *expressionMapper) valueOnlyVisit(node *ast.Node) bool {
	before := m.typeOnly
	m.typeOnly = false
	res := visit(m.mapInNonBindingPosition, node)
	m.typeOnly = before
	return res
}
func (m *expressionMapper) typeOnlyNodeListVisit(nodeList *ast.NodeList) bool {
	if nodeList == nil {
		return false
	}
	return m.withTypeOnlyVisit(func() bool {
		for _, n := range nodeList.Nodes {
			if visit(m.mapInNonBindingPosition, n) {
				return true
			}
		}
		return false
	})
}

func (m *expressionMapper) mapInNonBindingPositionIfNotIdentifier(node *ast.Node) bool {
	return !ast.IsIdentifier(node) && m.mapInNonBindingPosition(node)
}

func (m *expressionMapper) mapInNonBindingPosition(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindIdentifier:
		if m.shouldPrefixIdentifier(node) {
			// TODO: perf
			p := utils.TrimNodeTextRange(m.expr.Ast, node)
			m.mapTextToNodePos(p.Pos())
			m.serviceText.WriteString("__VLS_Ctx.")
			m.mapTextToNodePos(p.End())
		}
		return false
	case ast.KindShorthandPropertyAssignment:
		name := node.Name()
		if m.shouldPrefixIdentifier(name) {
			m.mapTextToNodePos(node.Pos())
			m.serviceText.WriteString(name.Text())
			m.serviceText.WriteString(": __VLS_Ctx.")
			m.mapTextToNodePos(node.End())
		}
		return false
	case ast.KindPropertyAccessExpression:
		n := node.AsPropertyAccessExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name())
	case ast.KindQualifiedName:
		n := node.AsQualifiedName()
		return visit(m.mapInNonBindingPosition, n.Left) || visit(m.mapInNonBindingPositionIfNotIdentifier, n.Right)
	case ast.KindEnumMember:
		n := node.AsEnumMember()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindPropertyDeclaration:
		n := node.AsPropertyDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyVisit(n.Type) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindPropertyAssignment:
		n := node.AsPropertyAssignment()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindGetAccessor:
		n := node.AsGetAccessorDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindSetAccessor:
		n := node.AsSetAccessorDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindVariableDeclaration:
		decl := node.AsVariableDeclaration()
		return visit(m.mapInBindingPosition, decl.Name()) || m.typeOnlyVisit(decl.Type) || visit(m.mapInNonBindingPosition, decl.Initializer)
	case ast.KindBreakStatement,
		ast.KindContinueStatement,
		ast.KindLabeledStatement,
		ast.KindModuleDeclaration:
		return false
	case ast.KindFunctionDeclaration:
		n := node.AsFunctionDeclaration()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindArrowFunction:
		n := node.AsArrowFunction()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindFunctionExpression:
		n := node.AsFunctionExpression()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindClassDeclaration:
		n := node.ClassLikeData()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.HeritageClauses) || visitNodeList(m.mapInNonBindingPosition, n.Members)
	case ast.KindConstructor:
		n := node.AsConstructorDeclaration()
		return m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindMethodDeclaration:
		n := node.AsMethodDeclaration()
		return visit(m.mapInNonBindingPositionIfNotIdentifier, n.Name()) || m.typeOnlyNodeListVisit(n.TypeParameters) || visitNodeList(m.mapInNonBindingPosition, n.Parameters) || m.typeOnlyVisit(n.Type) || m.typeOnlyVisit(n.FullSignature) || visit(m.mapInNonBindingPosition, n.Body)
	case ast.KindHeritageClause:
		n := node.AsHeritageClause()
		if n.Token == ast.KindImplementsKeyword {
			return m.withTypeOnlyVisit(func() bool {
				return node.ForEachChild(m.mapInNonBindingPosition)
			})
		}
	case ast.KindExpressionWithTypeArguments:
		n := node.AsExpressionWithTypeArguments()
		return visit(m.mapInNonBindingPosition, n.Expression) || m.typeOnlyNodeListVisit(n.TypeArguments)
	case ast.KindParameter:
		n := node.AsParameterDeclaration()
		return visit(m.mapInNonBindingPosition, n.Name()) || m.typeOnlyVisit(n.Type) || visit(m.mapInNonBindingPosition, n.Initializer)
	case ast.KindAsExpression:
		n := node.AsAsExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || m.typeOnlyVisit(n.Type)
	case ast.KindCallExpression:
		n := node.AsCallExpression()
		return visit(m.mapInNonBindingPosition, n.Expression) || m.typeOnlyNodeListVisit(n.TypeArguments) || visitNodeList(m.mapInNonBindingPosition, n.Arguments)
	case ast.KindTypeQuery:
		n := node.AsTypeQueryNode()
		return m.valueOnlyVisit(n.ExprName) || m.typeOnlyNodeListVisit(n.TypeArguments)
	case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
		return m.withTypeOnlyVisit(func() bool {
			return node.ForEachChild(m.mapInNonBindingPosition)
		})
	}
	// TODO: JSX

	return node.ForEachChild(m.mapInNonBindingPosition)
}

func camelize(str string, buf *strings.Builder) {
	hadDash := false
	lastWritten := 0
	for i, r := range str {
		if r == '-' {
			hadDash = true
			// TODO: what if double dash, like foo--bar
			continue
		}

		if hadDash {
			hadDash = false
			buf.WriteString(str[lastWritten : i-1])
			// TODO(perf): fast path for ascii, also ToUpper allocates internally
			buf.WriteString(strings.ToUpper(string(r)))
			lastWritten = i + utf8.RuneLen(r)
		}
	}
	buf.WriteString(str[lastWritten:])
}

func isBuiltInComponent(name string) bool {
	switch name {
	case "Teleport",
		"teleport",
		"Suspense",
		"suspense",
		"KeepAlive",
		"keep-alive",
		"BaseTransition",
		"base-transition",
		"Transition",
		"transition",
		"TransitionGroup",
		"transition-group":
		return true
	default:
		return false
	}
}

func isNativeElement(name string) bool {
	_, ok := vue_parser.NativeTags[name]
	return ok
}
