// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package tfdiagsappendcheck

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// DiagsAppendAnalyzer reports ignored return values from tfdiags.Diagnostics.Append.
var DiagsAppendAnalyzer = &analysis.Analyzer{
	Name: "Check tfdiags.Diagnostics.Append usage",
	Doc:  "reports ignored return values from tfdiags.Diagnostics.Append",
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
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

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Append" {
			return
		}

		selInfo, ok := pass.TypesInfo.Selections[sel]
		if !ok || selInfo.Kind() != types.MethodVal {
			return
		}

		if !isTfdiagsDiagnostics(selInfo.Recv()) {
			// Ignore calls to other Append methods that aren't tfdiags.Diagnostics.Append.
			return
		}

		pass.Reportf(sel.Sel.Pos(), "ignored return value from tfdiags.Diagnostics.Append")
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
