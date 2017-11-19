package testharness

import (
	"fmt"

	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/yuin/gopher-lua"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/gopherlua-cty/luacty"
)

// expect represents a call to either "expect" (within an "it") or "require"
// (within a "describe") that allows the user to make a statement about what
// is expected or required within a test.
//
// Expectations are used to decide if an "it" tests succeeds or fails.
// Requirements are used to decide whether or not a set of tests should be
// skipped.
//
// When expect/require appear in test spec files they are used by immediately
// calling a method to say what must be true about the value:
//
//     expect(var.protocol).to_equal("HTTP")
//
// Therefore when an "expect" is initially instantiated it lacks a result,
// with one being assigned only when one of the methods is called. If no
// method is ever called, the expectation never recieves a result and this
// should be reported to the user as an error. Likewise, if a method is
// called when a result is already present then this is also an error.
type expect struct {
	lstate *lua.LState

	// We accept any Lua value at instantiation time, but get more picky
	// once one of the methods is called.
	value lua.LValue

	result CheckResult
	diags  tfdiags.Diagnostics
	detail string

	defRange tfdiags.SourceRange
}

func newExpect(lstate *lua.LState, given lua.LValue, defRange tfdiags.SourceRange) *expect {
	return &expect{
		lstate:   lstate,
		value:    given,
		result:   Skipped,
		defRange: defRange,
	}
}

// Result returns the result of the expecatation, as either Success or Failure.
// Returns Skipped if no assertion method was ever called, or Error if
// error-level diagnostics prevented a result from being produced.
func (e *expect) Result() CheckResult {
	return e.result
}

// luaObject returns the value that should be returned from the expect/require
// call to make available the assertion methods.
func (e *expect) LuaObject() lua.LValue {
	L := e.lstate
	ret := L.NewTable()
	ret.RawSet(lua.LString("to_equal"), L.NewFunction(e.luaAssertEqual))
	return ret
}

func (e *expect) luaAssertEqual(L *lua.LState) int {
	if e.result != Skipped {
		e.diags = e.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Re-used expectation",
			Detail:   "Only one assertion call may be made per expectation.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}
	// If we exit any way other than falling out of the end of the function
	// then we'll assume we failed.
	e.result = Error

	if L.GetTop() != 1 {
		e.diags = e.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid \"to_equal\" assertion",
			Detail:   "A \"to_equal\" assertion must have one argument: the value to compare to.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}

	otherVal := L.CheckAny(1)

	// We use cty-style equality for this function, which requires that we
	// be given values that can be converted to that type system.
	conv := luacty.NewConverter(L)

	got, err := conv.ToCtyValue(e.value, cty.DynamicPseudoType)
	if err != nil {
		e.diags = e.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid value for \"to_equal\" assertion",
			Detail:   fmt.Sprintf("Invalid given value: %s.", err),
			Subject:  callingRange(L, 1),
		})
	}

	want, err := conv.ToCtyValue(otherVal, cty.DynamicPseudoType)
	if err != nil {
		e.diags = e.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid value for \"to_equal\" assertion",
			Detail:   fmt.Sprintf("Invalid wanted value: %s.", err),
			Subject:  callingRange(L, 1),
		})
	}

	if e.diags.HasErrors() {
		return 0
	}

	// We'll try to convert the given value into the same type as the
	// wanted value, since Terraform values don't always show up in the
	// type we ideally want.
	converted, err := convert.Convert(got, want.Type())
	if err == nil { // if not possible for any reason, just leave as-is and we'll fail the test below
		got = converted
	}

	result, err := stdlib.Equal(got, want)
	if err != nil {
		e.diags = e.diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid \"to_equal\" comparison",
			Detail:   fmt.Sprintf("Comparison failed: %s.", err),
			Subject:  callingRange(L, 1),
		})
	}

	if !result.IsKnown() {
		result = cty.False
	}

	if result.True() {
		e.result = Success
	} else {
		e.result = Failure
		e.detail = fmt.Sprintf("expected %q to be %q", e.value, otherVal)
	}

	return 0
}
