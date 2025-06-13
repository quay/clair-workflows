package findurls

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "findurls",
	Doc:      "check that URL-like strings have an annotation comment",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var prefix = regexp.MustCompile("^[`\"]https?://")

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	scope := pass.Pkg.Scope()
	for cur := range inspect.Root().Preorder((*ast.File)(nil)) {
		f := pass.Fset.File(cur.Node().Pos())
		if strings.HasSuffix(f.Name(), `_test.go`) {
			continue
		}

	Decl:
		for cur := range cur.Preorder((*ast.GenDecl)(nil)) {
			decl := cur.Node().(*ast.GenDecl)
			if s := scope.Innermost(decl.Pos()).Parent(); s != scope {
				continue
			}
			switch decl.Tok {
			case token.VAR:
			case token.CONST:
			default:
				continue
			}

			// Check that it has a url-like start.
			urlLike := false
			for c := range cur.Preorder((*ast.BasicLit)(nil)) {
				lit := c.Node().(*ast.BasicLit)
				if lit.Kind != token.STRING {
					continue
				}
				if prefix.MatchString(lit.Value) {
					urlLike = true
					break
				}
			}
			if !urlLike {
				continue
			}

			// Check for annotation.
			if doc := decl.Doc; doc != nil {
				for _, c := range doc.List {
					for _, l := range strings.Split(c.Text, "\n") {
						if strings.HasPrefix(l[2:], `doc:`) {
							continue Decl
						}
					}
				}
			}

			b, err := pass.ReadFile(f.Name())
			if err != nil {
				return nil, err
			}
			pass.Report(analysis.Diagnostic{
				Pos: decl.Pos(),
				End: decl.End(),
				Message: fmt.Sprintf("found URL-like string without annotation: %+#q",
					string(b[f.Position(decl.Pos()).Offset:f.Position(decl.End()).Offset]),
				),
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: `Add a "//doc:<subject>" annotation`,
					},
				},
			})
		}
	}
	return nil, nil
}
