// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package defermistake defines an Analyzer that reports common errors using defer.
// Specifically, this package is concerned with reporting errors when a defer statement
// is made with a function expression that is evaluated before the defer statements runs.
//
// For example, doing: defer observe(time.Since(time.Now())) will result in the observe
// function receiving a time.Duration that is near zero. In order to have the intended
// output, you can write: defer func() { observe(time.Since(now)) }()
//
// Currently the only function that is reported by this analysis pass is time.Since().
package defermistake

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const Doc = `report common mistakes evaluating functions at defer invocation

The defermistake analysis reports if a function is evaluated when the defer is invoked,
but it most likely that it is intended to be evaluated when the defer is executed.

time.Since is currently the only function that is checked in this way.`

// Analyzer is the defermistake analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "defermistake",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Doc:      Doc,
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// checkDeferCall walks an expression tree and reports if any expressions are time.Since.
	// FIXME: it does not walk into an ast.CallExpr, because FuncLit.Body is a BlockStmt.
	// This means we do not catch a case where a function is invoked in a function literal:
	//    defer x(func() time.Duration {time.Since(x)}())
	checkDeferCall := func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.CallExpr:
			// Useful print if you need to debug:
			//   pos := pass.Fset.Position(call.Pos())
			//   fmt.Printf("%v:%v\t| =>> fun=%T \n", pos.Line, pos.Column, call.Fun)
			fn, ok := typeutil.Callee(pass.TypesInfo, v).(*types.Func)
			if ok && isTimeSince(fn) {
				pass.Reportf(v.Pos(), "defer func should not evaluate time.Since")
				return true
			}

			return true
		case ast.Expr:
			return true
		}
		return false
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// FIXME: ast.Inspect is called on the defer call. This could instead
	// be constructed instead by creating a different function on inspect
	// that allowed you to traverse nodes by matching a pattern. I don't know
	// if this preferred to this more simplistic implementation.
	nodeFilter := []ast.Node{
		(*ast.DeferStmt)(nil),
	}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		d := n.(*ast.DeferStmt)
		ast.Inspect(d.Call, checkDeferCall)
	})

	return nil, nil
}

func isTimeSince(f *types.Func) bool {
	if f.Name() == "Since" && f.Pkg().Path() == "time" {
		return true
	}
	return false
}
