package bareprints

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

var Analyzer = &analysis.Analyzer{
	Name:     "bareprints",
	Doc:      "check for bare prints to stdout",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

Walk:
	for cur := range inspect.Root().Preorder((*ast.CallExpr)(nil)) {
		call := cur.Node().(*ast.CallExpr)

		obj := typeutil.Callee(pass.TypesInfo, call)
		switch f := obj.(type) {
		case *types.Builtin:
			switch f.Name() {
			case "print":
			case "println":
			default:
				continue
			}
		case *types.Func:
			f = f.Origin()
			pkg := f.Pkg()
			if pkg == nil { // ???
				continue
			}
			if pkg.Path() != "fmt" {
				continue
			}
			switch f.Name() {
			case "Print":
			case "Printf":
			case "Println":
			default:
				continue
			}
		default:
			continue
		}

		f := pass.Fset.File(call.Pos())
		// This is a bare print, but allow it if in an example.
		for p := range cur.Enclosing((*ast.FuncDecl)(nil)) {
			decl := p.Node().(*ast.FuncDecl)
			if strings.HasPrefix(decl.Name.Name, `Example`) &&
				strings.HasSuffix(f.Name(), `_test.go`) {
				continue Walk
			}
		}

		ln := f.Line(call.Pos())
		start := f.LineStart(ln)
		// This should always hold, because it should be impossible to have the
		// very last line of a file be a function call.
		end := f.LineStart(ln + 1)
		b, err := pass.ReadFile(f.Name())
		if err != nil {
			return nil, err
		}

		pass.Report(analysis.Diagnostic{
			Pos: call.Pos(),
			End: call.End(),
			Message: fmt.Sprintf("found print to stdout: %+#q",
				string(b[f.Position(call.Pos()).Offset:f.Position(call.End()).Offset]),
			),
			SuggestedFixes: []analysis.SuggestedFix{
				{
					Message: "Remove the line",
					TextEdits: []analysis.TextEdit{
						{
							Pos: start,
							End: end,
						},
					},
				},
			},
		})
	}

	return nil, nil
}
