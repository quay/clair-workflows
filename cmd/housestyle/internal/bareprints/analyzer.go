package bareprints

import (
	"fmt"
	"go/ast"
	"go/types"

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
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.CallExpr)(nil),
	}
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok || call == nil {
			return
		}

		obj := typeutil.Callee(pass.TypesInfo, call)
		switch f := obj.(type) {
		case *types.Builtin:
			switch f.Name() {
			case "print":
			case "println":
			default:
				return
			}
		case *types.Func:
			f = f.Origin()
			if f.Pkg().Path() != "fmt" {
				return
			}
			switch f.Name() {
			case "Print":
			case "Printf":
			case "Println":
			default:
				return
			}
		default:
			return
		}

		f := pass.Fset.File(n.Pos())
		ln := f.Line(n.Pos())
		start := f.LineStart(ln)
		// This should always hold, because it should be impossible to have the
		// very last line of a file be a function call.
		end := f.LineStart(ln + 1)
		b, err := pass.ReadFile(f.Name())
		if err != nil {
			panic(err)
		}

		pass.Report(analysis.Diagnostic{
			Pos: call.Pos(),
			End: call.End(),
			Message: fmt.Sprintf("found print to stdout: %+#q",
				string(b[f.Position(n.Pos()).Offset:f.Position(n.End()).Offset]),
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
	})
	return nil, nil
}
