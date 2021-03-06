package exprstress

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// TestExpression tries to evaluate the given test expression and returns
// errors if parsing or evaluation fail or if a successful result doesn't
// conform to the expression's expectations.
func TestExpression(expr Expression) []error {
	src := ExpressionSourceBytes(expr)
	expected := expr.ExpectedResult()
	return testExpression(src, expected)
}

// TestCase is a helper type to help with permanently capturing interesting
// test cases and their expected results as examples in hand-written unit tests.
//
// The "tfexprstress run" command will generate example values of this type
// for each failure it encounters, to help with setting up a reproducible
// test case for further debugging.
//
// You can use either method Run or global function TestCases as a convenient
// way to verify TestCase values within a normal Go test run.
type TestCase struct {
	ExprSrc  string
	Expected Expected
}

// Run executes the reciever as a single test against the given testing.T.
//
// Run generates errors on the given testing.T if the test fails, but it
// doesn't prevent further execution of subsequent test code. In case the
// outcome of the test case needs to conditionally block further test
// execution, TestCase returns true if it detected at least one error. Callers
// may ignore the return value if the test outcome is immeterial for any
// subsequent test code.
func (tc TestCase) Run(t *testing.T) bool {
	t.Helper()
	errs := testExpression([]byte(tc.ExprSrc), tc.Expected)
	for _, err := range errs {
		t.Error(err)
	}
	return len(errs) > 0
}

// TestCases is a helper for running a number of separate test cases at the
// same time inside a single test function.
//
// TestCases uses t.Run to generate a separate subtest for each of the test
// cases, and then calls method Run on each of the test cases to run them.
// As with t.Run, this might generate test errors but it will not prevent
// further execution in the event of test failures. However, it will return
// true if at least one of the given test cases fails, to allow for a caller
// to conditionally halt test execution if necessary.
func TestCases(t *testing.T, tests ...TestCase) bool {
	failure := false
	for _, test := range tests {
		t.Run(test.ExprSrc, func(t *testing.T) {
			if test.Run(t) {
				failure = true
			}
		})
	}
	return failure
}

func testExpression(src []byte, expected Expected) (errs []error) {
	defer func() {
		// Since expression evaluation is typically self-contained we'll
		// try to present panics as normal errors so that we can potentially
		// print out a useful reproduction case message and keep testing.
		if r := recover(); r != nil {
			errs = append(errs, errorForPanic(r))
		}
	}()

	expr, hclDiags := hclsyntax.ParseExpression(src, "", hcl.InitialPos)
	for _, diag := range hclDiags {
		errs = append(errs, diag)
	}
	if len(errs) > 0 {
		// If parsing failed then we won't even try evaluation
		return errs
	}

	scope := &lang.Scope{
		Data: testData,
	}

	v, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
	for _, diag := range diags {
		desc := diag.Description()
		var rng hcl.Range
		if subject := diag.Source().Subject; subject != nil {
			rng = subject.ToHCL()
		}
		errs = append(errs, fmt.Errorf("[%s] %s: %s", rng, desc.Summary, desc.Detail))
	}
	if len(errs) > 0 {
		// If evaluation failed then we won't check against the expected value
		return errs
	}

	if v == cty.NilVal {
		// NilVal is never a valid result for a successful evaluation
		errs = append(errs, fmt.Errorf("result is cty.NilVal"))
		return errs
	}

	if got, want := v.Type(), expected.Type; !want.Equals(got) {
		errs = append(errs, fmt.Errorf(
			"wrong result type\ngot:  %swant: %s",
			ctydebug.TypeString(got),
			ctydebug.TypeString(want),
		))
	}

	var gotMode ValueMode
	switch {
	case v.IsNull():
		gotMode = NullValue
	case !v.IsKnown():
		gotMode = UnknownValue
	default:
		gotMode = SpecifiedValue
	}
	if gotMode != expected.Mode {
		errs = append(errs, fmt.Errorf(
			"result has wrong mode\ngot:  %s\nwant: %s",
			gotMode, expected.Mode,
		))
	}

	if got, want := v.IsMarked(), expected.Sensitive; got != want {
		errs = append(errs, fmt.Errorf(
			"wrong result sensitivity\ngot:  %#v\nwant: %#v",
			got, want,
		))
	}

	if expected.SpecialNumber != NumberUninteresting {
		// The SpecialNumber field is primarily to help the expression
		// generator produce valid results, but since we've gone to the trouble
		// of populating it we'll also take the opportunity to do some
		// additional verification of the result.
		v, _ := v.Unmark()
		nv, err := convert.Convert(v, cty.Number)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"result cannot be a number\ngot:  %#v\nwant: any value convertable to a number",
				v,
			))
		} else {
			switch expected.SpecialNumber {
			case NumberZero:
				if got, want := nv, cty.Zero; !want.RawEquals(got) {
					errs = append(errs, fmt.Errorf(
						"wrong numeric result\ngot:  %#v\nwant: %v",
						got, want,
					))
				}
			case NumberOne:
				if got, want := nv, cty.NumberIntVal(1); !want.RawEquals(got) {
					errs = append(errs, fmt.Errorf(
						"wrong numeric result\ngot:  %#v\nwant: %v",
						got, want,
					))
				}
			case NumberInfinity:
				if !(cty.PositiveInfinity.RawEquals(nv) || cty.NegativeInfinity.RawEquals(nv)) {
					errs = append(errs, fmt.Errorf(
						"wrong numeric result\ngot:  %#v\nwant: cty.PositiveInfinity or cty.NegativeInfinity",
						nv,
					))
				}
			default:
				panic(fmt.Sprintf("unhandled %s", expected.SpecialNumber))
			}
		}
	}

	return errs
}

type panicError struct {
	Value interface{}
	Stack []byte
}

func errorForPanic(val interface{}) error {
	return panicError{
		Value: val,
		Stack: debug.Stack(),
	}
}

func (e panicError) Error() string {
	return fmt.Sprintf("panic during expression evaluation: %s\n%s", e.Value, e.Stack)
}
