// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/zclconf/go-cty/cty"
)

// This file contains tests for the "test-only globals" mechanism itself, and
// so if tests from this file fail at the same time as tests in other files
// in this package it's probably productive to address the failures in here
// first, in case they are indirectly causing failures in other files where
// unit tests are written to rely on this mechanism.

func TestTestOnlyGlobals_parseAndEval(t *testing.T) {
	// We'll do our best to try to attract the attention of the hypothetical
	// maintainer who is looking at a wall of test failures of many different
	// tests that were all depending on test-only globals to function
	// correctly.
	t.Log(`
--------------------------------------------------------------------------------------------------------------------------
NOTE: If any part of this test fails, the problem might also be the cause of other test failures elsewhere in this package
                                  (so maybe prioritize fixing this one first!)
--------------------------------------------------------------------------------------------------------------------------
`)

	// We're evaluating individual expressions in isolation here because
	// that allows us to focus as closely as possible on only testing the
	// test-only globals mechanism itself, without relying on any other
	// language features such as output values.
	//
	// This does mean that this test might miss some situations that can
	// arise only when a test only global is mentioned in a particular
	// evaluation context. If that arises later, consider adding another
	// test alongside this one which tests _that_ situation as tightly
	// as possible too; the goal of the tests in this file is to give a
	// clear signal if the test utilities themselves are malfunctioning,
	// so that maintainers can minimize time wasted trying to debug another
	// test that's relying on this utility.
	fooExpr, hclDiags := hclsyntax.ParseExpression([]byte("_test_only_global.foo"), "test", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("failed to parse expression: %s", hclDiags.Error())
	}
	barAttrExpr, hclDiags := hclsyntax.ParseExpression([]byte("_test_only_global.bar.attr"), "test", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("failed to parse expression: %s", hclDiags.Error())
	}
	nonExistExpr, hclDiags := hclsyntax.ParseExpression([]byte("_test_only_global.nonexist"), "test", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("failed to parse expression: %s", hclDiags.Error())
	}

	fakeConfig := testStackConfigEmpty(t)
	main := testEvaluator(t, testEvaluatorOpts{
		Config: fakeConfig,
		TestOnlyGlobals: map[string]cty.Value{
			"foo": cty.StringVal("foo value"),
			"bar": cty.ObjectVal(map[string]cty.Value{
				"attr": cty.StringVal("bar.attr value"),
			}),
		},
	})

	ctx := context.Background()
	mainStack := main.MainStack(ctx)

	t.Run("foo", func(t *testing.T) {
		got, diags := EvalExpr(ctx, fooExpr, InspectPhase, mainStack)
		if diags.HasErrors() {
			t.Errorf("unexpected errors: %s", diags.Err().Error())
		}
		want := cty.StringVal("foo value")
		if !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}

		t.Run("without test-only globals enabled", func(t *testing.T) {
			noGlobalsMain := NewForInspecting(fakeConfig, stackstate.NewState(), InspectOpts{})
			mainStack := noGlobalsMain.MainStack(ctx)
			got, diags := EvalExpr(ctx, fooExpr, InspectPhase, mainStack)
			if !diags.HasErrors() {
				t.Fatalf("unexpected success\ngot:  %#v\nwant: an error diagnostic", got)
			}
			if len(diags) != 1 {
				t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
			}
			// Without test-only globals enabled, we try our best to pretend
			// that test-only globals don't exist at all, since they are an
			// implementation detail as far as end-users are concerned.
			gotSummary := diags[0].Description().Summary
			wantSummary := `Reference to unknown symbol`
			if gotSummary != wantSummary {
				t.Errorf("unexpected diagnostic summary\ngot:  %s\nwant: %s", gotSummary, wantSummary)
			}
		})
	})
	t.Run("bar.attr", func(t *testing.T) {
		got, diags := EvalExpr(ctx, barAttrExpr, InspectPhase, mainStack)
		if diags.HasErrors() {
			t.Errorf("unexpected errors: %s", diags.Err().Error())
		}
		want := cty.StringVal("bar.attr value")
		if !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("nonexist", func(t *testing.T) {
		got, diags := EvalExpr(ctx, nonExistExpr, InspectPhase, mainStack)
		if !diags.HasErrors() {
			t.Fatalf("unexpected success\ngot:  %#v\nwant: an error diagnostic", got)
		}
		if len(diags) != 1 {
			t.Fatalf("unexpected diagnostics: %s", diags.Err().Error())
		}
		gotSummary := diags[0].Description().Summary
		wantSummary := `Reference to undefined test-only global`
		if gotSummary != wantSummary {
			t.Errorf("unexpected diagnostic summary\ngot:  %s\nwant: %s", gotSummary, wantSummary)
		}
	})
}
