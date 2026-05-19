// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package defectdetector

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// IgnoredReturnedDiagsAnalyzer reports ignored return values with type tfdiags.Diagnostics.
// It emits a more specific message for ignored tfdiags.Diagnostics.Append calls.
var IgnoredReturnedDiagsAnalyzer = &analysis.Analyzer{
	Name: "ignored_diag_returns",
	Doc:  "reports ignored tfdiags.Diagnostics return values",
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	Run: ignoredDiagReturns,
}

func ignoredDiagReturns(pass *analysis.Pass) (interface{}, error) {
	ins, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.ExprStmt)(nil)}
	ins.Preorder(nodeFilter, func(n ast.Node) {
		exprStmt := n.(*ast.ExprStmt)
		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			return
		}

		// Report a specialized message for ignored tfdiags.Diagnostics.Append calls.
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == "Append" {
			selInfo, ok := pass.TypesInfo.Selections[sel]
			if ok && selInfo.Kind() == types.MethodVal && isTfdiagsDiagnostics(selInfo.Recv()) {
				pass.Reportf(sel.Sel.Pos(), "ignored return value from tfdiags.Diagnostics.Append")
				return
			}
		}

		callType := pass.TypesInfo.TypeOf(call)
		if callType == nil {
			return
		}

		diagsReturn := false
		switch t := callType.(type) {
		case *types.Tuple:
			if t.Len() != 1 {
				return
			}
			diagsReturn = isTfdiagsDiagnostics(t.At(0).Type())
		default:
			diagsReturn = isTfdiagsDiagnostics(t)
		}

		if !diagsReturn {
			return
		}

		pass.Reportf(call.Pos(), "ignored return value with type tfdiags.Diagnostics")
	})

	return nil, nil
}

// Check that the receiver type is tfdiags.Diagnostics or *tfdiags.Diagnostics
func isTfdiagsDiagnostics(t types.Type) bool {
	if t == nil {
		return false
	}

	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Name() != "Diagnostics" {
		return false
	}

	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return pkg.Path() == "github.com/hashicorp/terraform/internal/tfdiags"
}
