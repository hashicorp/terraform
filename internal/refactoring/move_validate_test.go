// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring_test

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestValidateMoves(t *testing.T) {
	rootCfg, instances := loadRefactoringFixture(t, "testdata/move-validate-zoo")

	tests := map[string]struct {
		Statements []refactoring.MoveStatement
		WantError  string
	}{
		"no move statements": {
			Statements: nil,
			WantError:  ``,
		},
		"some valid statements": {
			Statements: []refactoring.MoveStatement{
				// This is just a grab bag of various valid cases that don't
				// generate any errors at all.
				makeTestMoveStmt(t,
					``,
					`test.nonexist1`,
					`test.target1`,
				),
				makeTestMoveStmt(t,
					`single`,
					`test.nonexist1`,
					`test.target1`,
				),
				makeTestMoveStmt(t,
					``,
					`test.nonexist2`,
					`module.nonexist.test.nonexist2`,
				),
				makeTestMoveStmt(t,
					``,
					`module.single.test.nonexist3`,
					`module.single.test.single`,
				),
				makeTestMoveStmt(t,
					``,
					`module.single.test.nonexist4`,
					`test.target2`,
				),
				makeTestMoveStmt(t,
					``,
					`test.single[0]`, // valid because test.single doesn't have "count" set
					`test.target3`,
				),
				makeTestMoveStmt(t,
					``,
					`test.zero_count[0]`, // valid because test.zero_count has count = 0
					`test.target4`,
				),
				makeTestMoveStmt(t,
					``,
					`test.zero_count[1]`, // valid because test.zero_count has count = 0
					`test.zero_count[0]`,
				),
				makeTestMoveStmt(t,
					``,
					`module.nonexist1`,
					`module.target3`,
				),
				makeTestMoveStmt(t,
					``,
					`module.nonexist1[0]`,
					`module.target4`,
				),
				makeTestMoveStmt(t,
					``,
					`module.single[0]`, // valid because module.single doesn't have "count" set
					`module.target5`,
				),
				makeTestMoveStmt(t,
					``,
					`module.for_each["nonexist1"]`,
					`module.for_each["a"]`,
				),
				makeTestMoveStmt(t,
					``,
					`module.for_each["nonexist2"]`,
					`module.nonexist.module.nonexist`,
				),
				makeTestMoveStmt(t,
					``,
					`module.for_each["nonexist3"].test.single`, // valid because module.for_each doesn't currently have a "nonexist3"
					`module.for_each["a"].test.single`,
				),
			},
			WantError: ``,
		},
		"two statements with the same endpoints": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.b`,
				),
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.b`,
				),
			},
			WantError: ``,
		},
		"moving nowhere": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.a`,
				),
			},
			WantError: `Redundant move statement: This statement declares a move from module.a to the same address, which is the same as not declaring this move at all.`,
		},
		"cyclic chain": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.b`,
				),
				makeTestMoveStmt(t,
					``,
					`module.b`,
					`module.c`,
				),
				makeTestMoveStmt(t,
					``,
					`module.c`,
					`module.a`,
				),
			},
			WantError: `Cyclic dependency in move statements: The following chained move statements form a cycle, and so there is no final location to move objects to:
  - test:1,1: module.a[*] → module.b[*]
  - test:1,1: module.b[*] → module.c[*]
  - test:1,1: module.c[*] → module.a[*]

A chain of move statements must end with an address that doesn't appear in any other statements, and which typically also refers to an object still declared in the configuration.`,
		},
		"module.single as a call still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single`,
					`module.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single, but that module call is still declared at testdata/move-validate-zoo/move-validate-root.tf:6,1.

Change your configuration so that this call will be declared as module.other instead.`,
		},
		"module.single as an instance still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single`,
					`module.other[0]`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single, but that module instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:6,1.

Change your configuration so that this instance will be declared as module.other[0] instead.`,
		},
		"module.count[0] still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.count[0]`,
					`module.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.count[0], but that module instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:12,12.

Change your configuration so that this instance will be declared as module.other instead.`,
		},
		`module.for_each["a"] still exists in configuration`: {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.for_each["a"]`,
					`module.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.for_each["a"], but that module instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:22,14.

Change your configuration so that this instance will be declared as module.other instead.`,
		},
		"test.single as a resource still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from test.single, but that resource is still declared at testdata/move-validate-zoo/move-validate-root.tf:27,1.

Change your configuration so that this resource will be declared as test.other instead.`,
		},
		"test.single as an instance still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.single`,
					`test.other[0]`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from test.single, but that resource instance is still declared at testdata/move-validate-zoo/move-validate-root.tf:27,1.

Change your configuration so that this instance will be declared as test.other[0] instead.`,
		},
		"module.single.test.single as a resource still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single.test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single.test.single, but that resource is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this resource will be declared as test.other instead.`,
		},
		"module.single.test.single as a resource declared in module.single still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					`single`,
					`test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single.test.single, but that resource is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this resource will be declared as module.single.test.other instead.`,
		},
		"module.single.test.single as an instance still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.single.test.single`,
					`test.other[0]`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.single.test.single, but that resource instance is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this instance will be declared as test.other[0] instead.`,
		},
		"module.count[0].test.single still exists in configuration": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.count[0].test.single`,
					`test.other`,
				),
			},
			WantError: `Moved object still exists: This statement declares a move from module.count[0].test.single, but that resource is still declared at testdata/move-validate-zoo/child/move-validate-child.tf:6,1.

Change your configuration so that this resource will be declared as test.other instead.`,
		},
		"two different moves from test.nonexist": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.nonexist`,
					`test.other1`,
				),
				makeTestMoveStmt(t,
					``,
					`test.nonexist`,
					`test.other2`,
				),
			},
			WantError: `Ambiguous move statements: A statement at test:1,1 declared that test.nonexist moved to test.other1, but this statement instead declares that it moved to test.other2.

Each resource can move to only one destination resource.`,
		},
		"two different moves to test.single": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.other1`,
					`test.single`,
				),
				makeTestMoveStmt(t,
					``,
					`test.other2`,
					`test.single`,
				),
			},
			WantError: `Ambiguous move statements: A statement at test:1,1 declared that test.other1 moved to test.single, but this statement instead declares that test.other2 moved there.

Each resource can have moved from only one source resource.`,
		},
		"two different moves to module.count[0].test.single across two modules": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.other1`,
					`module.count[0].test.single`,
				),
				makeTestMoveStmt(t,
					`count`,
					`test.other2`,
					`test.single`,
				),
			},
			WantError: `Ambiguous move statements: A statement at test:1,1 declared that test.other1 moved to module.count[0].test.single, but this statement instead declares that module.count[0].test.other2 moved there.

Each resource can have moved from only one source resource.`,
		},
		"move from resource in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.fake_external.test.thing`,
					`test.thing`,
				),
			},
			WantError: ``,
		},
		"move to resource in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`test.thing`,
					`module.fake_external.test.thing`,
				),
			},
			WantError: ``,
		},
		"move from module call in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.fake_external.module.a`,
					`module.b`,
				),
			},
			WantError: ``,
		},
		"move to module call in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.a`,
					`module.fake_external.module.b`,
				),
			},
			WantError: ``,
		},
		"implied move from resource in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestImpliedMoveStmt(t,
					``,
					`module.fake_external.test.thing`,
					`test.thing`,
				),
			},
			// Implied move statements are not subject to the cross-package restriction
			WantError: ``,
		},
		"implied move to resource in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestImpliedMoveStmt(t,
					``,
					`test.thing`,
					`module.fake_external.test.thing`,
				),
			},
			// Implied move statements are not subject to the cross-package restriction
			WantError: ``,
		},
		"implied move from module call in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestImpliedMoveStmt(t,
					``,
					`module.fake_external.module.a`,
					`module.b`,
				),
			},
			// Implied move statements are not subject to the cross-package restriction
			WantError: ``,
		},
		"implied move to module call in another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestImpliedMoveStmt(t,
					``,
					`module.a`,
					`module.fake_external.module.b`,
				),
			},
			// Implied move statements are not subject to the cross-package restriction
			WantError: ``,
		},
		"move to a call that refers to another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.nonexist`,
					`module.fake_external`,
				),
			},
			WantError: ``, // This is okay because the call itself is not considered to be inside the package it refers to
		},
		"move to instance of a call that refers to another module package": {
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t,
					``,
					`module.nonexist`,
					`module.fake_external[0]`,
				),
			},
			WantError: ``, // This is okay because the call itself is not considered to be inside the package it refers to
		},
		"crossing nested statements": {
			// overlapping nested moves will result in a cycle.
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t, ``,
					`module.nonexist.test.single`,
					`module.count[0].test.count[0]`,
				),
				makeTestMoveStmt(t, ``,
					`module.nonexist`,
					`module.count[0]`,
				),
			},
			WantError: `Cyclic dependency in move statements: The following chained move statements form a cycle, and so there is no final location to move objects to:
  - test:1,1: module.nonexist → module.count[0]
  - test:1,1: module.nonexist.test.single → module.count[0].test.count[0]

A chain of move statements must end with an address that doesn't appear in any other statements, and which typically also refers to an object still declared in the configuration.`,
		},
		"fully contained nested statements": {
			// we have to avoid a cycle because the nested moves appear in both
			// the from and to address of the parent when only the module index
			// is changing.
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t, `count`,
					`test.count`,
					`test.count[0]`,
				),
				makeTestMoveStmt(t, ``,
					`module.count`,
					`module.count[0]`,
				),
			},
		},
		"double fully contained nested statements": {
			// we have to avoid a cycle because the nested moves appear in both
			// the from and to address of the parent when only the module index
			// is changing.
			Statements: []refactoring.MoveStatement{
				makeTestMoveStmt(t, `count`,
					`module.count`,
					`module.count[0]`,
				),
				makeTestMoveStmt(t, `count.count`,
					`test.count`,
					`test.count[0]`,
				),
				makeTestMoveStmt(t, ``,
					`module.count`,
					`module.count[0]`,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotDiags := refactoring.ValidateMoves(test.Statements, rootCfg, instances)

			switch {
			case test.WantError != "":
				if !gotDiags.HasErrors() {
					t.Fatalf("unexpected success\nwant error: %s", test.WantError)
				}
				if got, want := gotDiags.Err().Error(), test.WantError; got != want {
					t.Fatalf("wrong error\ngot error:  %s\nwant error: %s", got, want)
				}
			default:
				if gotDiags.HasErrors() {
					t.Fatalf("unexpected error\ngot error: %s", gotDiags.Err().Error())
				}
			}
		})
	}
}

func makeTestMoveStmt(t *testing.T, moduleStr, fromStr, toStr string) refactoring.MoveStatement {
	t.Helper()

	module := addrs.RootModule
	if moduleStr != "" {
		module = addrs.Module(strings.Split(moduleStr, "."))
	}

	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(fromStr), "", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid from address: %s", hclDiags.Error())
	}
	fromEP, diags := addrs.ParseMoveEndpoint(traversal)
	if diags.HasErrors() {
		t.Fatalf("invalid from address: %s", diags.Err().Error())
	}

	traversal, hclDiags = hclsyntax.ParseTraversalAbs([]byte(toStr), "", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid to address: %s", hclDiags.Error())
	}
	toEP, diags := addrs.ParseMoveEndpoint(traversal)
	if diags.HasErrors() {
		t.Fatalf("invalid to address: %s", diags.Err().Error())
	}

	fromInModule, toInModule := addrs.UnifyMoveEndpoints(module, fromEP, toEP)
	if fromInModule == nil || toInModule == nil {
		t.Fatalf("incompatible move endpoints")
	}

	return refactoring.MoveStatement{
		From: fromInModule,
		To:   toInModule,
		DeclRange: tfdiags.SourceRange{
			Filename: "test",
			Start:    tfdiags.SourcePos{Line: 1, Column: 1},
			End:      tfdiags.SourcePos{Line: 1, Column: 1},
		},
	}
}

func makeTestImpliedMoveStmt(t *testing.T, moduleStr, fromStr, toStr string) refactoring.MoveStatement {
	t.Helper()
	ret := makeTestMoveStmt(t, moduleStr, fromStr, toStr)
	ret.Implied = true
	return ret
}
