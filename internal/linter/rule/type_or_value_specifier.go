package rule

import (
	"slices"
	"strings"

	"github.com/auvred/golar/internal/utils"
	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/checker"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/core"
	"github.com/microsoft/typescript-go/pkg/module"
	"github.com/microsoft/typescript-go/pkg/modulespecifiers"
	"github.com/microsoft/typescript-go/pkg/tspath"
)

type TypeOrValueSpecifier struct {
	From     string   `json:"from"`
	Name     []string `json:"name"`
	FilePath string   `json:"filePath"`
	Package  string   `json:"package"`
}

func typeMatchesStringSpecifier(
	typeChecker *checker.Checker,
	t *checker.Type,
	names []string,
) bool {
	return utils.HasBaseType(typeChecker, t, func(t *checker.Type) bool {
		alias := t.Alias()
		var symbol *ast.Symbol
		if alias == nil {
			symbol = t.Symbol()
		} else {
			symbol = alias.Symbol()
		}

		if symbol != nil && slices.Contains(names, symbol.Name) {
			return true
		}

		if utils.IsTypeFlagSet(t, checker.TypeFlagsIntrinsic) && slices.Contains(names, t.AsIntrinsicType().IntrinsicName()) {
			return true
		}

		return false
	})
}

func typeDeclaredInFile(
	relativePath string,
	declarationFiles []*ast.SourceFile,
	program *compiler.Program,
) bool {
	cwd := program.Host().GetCurrentDirectory()
	if relativePath == "" {
		return core.Some(declarationFiles, func(f *ast.SourceFile) bool {
			return strings.HasPrefix(f.FileName(), cwd)
		})
	}
	absPath := tspath.GetNormalizedAbsolutePath(relativePath, cwd)
	return core.Some(declarationFiles, func(f *ast.SourceFile) bool {
		return f.FileName() == absPath
	})
}

func typeDeclaredInLib(
	declarationFiles []*ast.SourceFile,
	program *compiler.Program,
) bool {
	// Assertion: The type is not an error type.

	// Intrinsic type (i.e. string, number, boolean, etc) - Treat it as if it's from lib.
	if len(declarationFiles) == 0 {
		return true
	}
	return core.Some(declarationFiles, func(d *ast.SourceFile) bool {
		return utils.IsSourceFileDefaultLibrary(program, d)
	})
}

func findParentModuleDeclaration(
	node *ast.Node,
) *ast.ModuleDeclaration {
	switch node.Kind {
	case ast.KindModuleDeclaration:
		decl := node.AsModuleDeclaration()
		if decl.Keyword == ast.KindNamespaceKeyword {
			break
		}
		if ast.IsStringLiteral(decl.Name()) {
			return decl
		}
		return nil
	case ast.KindSourceFile:
		return nil
	}

	return findParentModuleDeclaration(node.Parent)
}

func typeDeclaredInDeclareModule(
	packageName string,
	declarations []*ast.Node,
) bool {
	return core.Some(declarations, func(d *ast.Node) bool {
		parentModule := findParentModuleDeclaration(d)
		return parentModule != nil && parentModule.Name().Text() == packageName
	})
}

func typeDeclaredInDeclarationFile(
	packageName string,
	declarationFiles []*ast.SourceFile,
	program *compiler.Program,
) bool {
	return core.Some(declarationFiles, func(declaration *ast.SourceFile) bool {
		if !program.IsSourceFileFromExternalLibrary(declaration) {
			return false
		}

		packageIdName := modulespecifiers.GetPackageNameFromDirectory(declaration.FileName())
		if packageIdName == "" {
			return false
		}

		if packageIdName == packageName {
			return true
		}

		return module.GetPackageNameFromTypesPackageName(packageIdName) == packageName
	})
}

func typeDeclaredInPackageDeclarationFile(
	packageName string,
	declarations []*ast.Node,
	declarationFiles []*ast.SourceFile,
	program *compiler.Program,
) bool {
	return typeDeclaredInDeclareModule(packageName, declarations) ||
		typeDeclaredInDeclarationFile(packageName, declarationFiles, program)
}

func typeMatchesSpecifier(
	program *compiler.Program,
	typeChecker *checker.Checker,
	t *checker.Type,
	specifier TypeOrValueSpecifier,
) bool {
	if utils.IsTypeFlagSet(t, checker.TypeFlagsUnion) {
		return core.Every(utils.UnionTypeParts(t), func(t *checker.Type) bool {
			return typeMatchesSpecifier(program, typeChecker, t, specifier)
		})
	}

	wholeTypeMatches := func() bool {
		if utils.IsIntrinsicErrorType(t) {
			return false
		}

		if !typeMatchesStringSpecifier(typeChecker, t, specifier.Name) {
			return false
		}

		symbol := t.Symbol()
		if symbol == nil {
			alias := t.Alias()
			if alias != nil {
				symbol = alias.Symbol()
			}
		}
		var declarations []*ast.Node
		if symbol != nil {
			declarations = symbol.Declarations
		}
		declarationFiles := core.Map(declarations, func(d *ast.Node) *ast.SourceFile {
			return ast.GetSourceFileOfNode(d)
		})

		switch specifier.From {
		case "file":
			return typeDeclaredInFile(specifier.FilePath, declarationFiles, program)
		case "lib":
			return typeDeclaredInLib(declarationFiles, program)
		case "package":
			return typeDeclaredInPackageDeclarationFile(specifier.Package, declarations, declarationFiles, program)
		default:
			panic("unknown specifier: " + specifier.From)
		}
	}()

	if wholeTypeMatches {
		return true
	}

	if utils.IsTypeFlagSet(t, checker.TypeFlagsIntersection) {
		return core.Some(utils.IntersectionTypeParts(t), func(t *checker.Type) bool {
			return typeMatchesSpecifier(program, typeChecker, t, specifier)
		})
	}

	return false
}

func TypeMatchesSomeSpecifier(
	program *compiler.Program,
	typeChecker *checker.Checker,
	t *checker.Type,
	specifiers []TypeOrValueSpecifier,
) bool {
	return core.Some(specifiers, func(s TypeOrValueSpecifier) bool {
		return typeMatchesSpecifier(program, typeChecker, t, s)
	})
}
