package stackeval

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// This file contains some general test utilities that many of our other
// _test.go files rely on. It doesn't actually contain any tests itself.

// testStackConfig loads a stack configuration from the source bundle in this
// package's testdata directory.
//
// "collection" is the name of one of the synthetic source packages that's
// declared in the source bundle, and "subPath" is the path within that
// package.
func testStackConfig(t *testing.T, collection string, subPath string) *stackconfig.Config {
	t.Helper()

	// Our collection of test configurations is laid out like a source
	// bundle that was installed from some source addresses that don't
	// really exist, and so we'll construct a suitable fake source
	// address following that scheme.
	fakeSrcStr := fmt.Sprintf("https://testing.invalid/%s.tar.gz//%s", collection, subPath)
	fakeSrc, err := sourceaddrs.ParseRemoteSource(fakeSrcStr)
	if err != nil {
		t.Fatalf("artificial source address string %q is invalid: %s", fakeSrcStr, err)
	}

	sources := testSourceBundle(t)
	ret, diags := stackconfig.LoadConfigDir(fakeSrc, sources)
	if diags.HasErrors() {
		t.Fatalf("configuration is invalid\n%s", diags.Err().Error())
	}
	return ret
}

func testStackConfigEmpty(t *testing.T) *stackconfig.Config {
	t.Helper()
	sources := testSourceBundle(t)
	fakeAddr := sourceaddrs.MustParseSource("https://testing.invalid/nonexist.tar.gz").(sourceaddrs.RemoteSource)
	return stackconfig.NewEmptyConfig(fakeAddr, sources)
}

func testSourceBundle(t *testing.T) *sourcebundle.Bundle {
	t.Helper()
	sources, err := sourcebundle.OpenDir("testdata/sourcebundle")
	if err != nil {
		t.Fatalf("cannot open source bundle: %s", err)
	}
	return sources
}

// testEvaluator constructs a [Main] that's configured for [InspectPhase] using
// the given configuration, state, and other options.
//
// This evaluator is suitable for tests that focus only on evaluation logic
// within this package, but will not be suitable for all situations. Some
// tests should instantiate [Main] directly, particularly if they intend to
// exercise phase-specific functionality like planning or applying component
// instances.
func testEvaluator(t *testing.T, opts testEvaluatorOpts) *Main {
	t.Helper()
	if opts.Config == nil {
		t.Fatal("Config field must not be nil")
	}
	if opts.State == nil {
		opts.State = stackstate.NewState()
	}

	inputVals := make(map[stackaddrs.InputVariable]ExternalInputValue, len(opts.InputVariableValues))
	for name, val := range opts.InputVariableValues {
		inputVals[stackaddrs.InputVariable{Name: name}] = ExternalInputValue{
			Value: val,
			DefRange: tfdiags.SourceRange{
				Filename: "<test-input>",
				Start: tfdiags.SourcePos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: tfdiags.SourcePos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
			},
		}
	}

	return NewForInspecting(opts.Config, opts.State, InspectOpts{
		InputVariableValues: inputVals,
		ProviderFactories:   opts.ProviderFactories,
		TestOnlyGlobals:     opts.TestOnlyGlobals,
	})
}

type testEvaluatorOpts struct {
	// Config is required.
	Config *stackconfig.Config

	// State is optional; testEvaluator will use an empty state if this is nil.
	State *stackstate.State

	// InputVariableValues is optional and if set will provide the values
	// for the root stack input variables. Any variable not defined here
	// will evaluate to an unknown value of the configured type.
	InputVariableValues map[string]cty.Value

	// ProviderFactories is optional and if set provides factory functions
	// for provider types that the test can use. If not set then any attempt
	// to use provider configurations will lead to some sort of error.
	ProviderFactories ProviderFactories

	// TestOnlyGlobals is optional and if set makes it possible to use
	// references like _test_only_global.name to refer to values from this
	// map from anywhere in the entire stack configuration.
	//
	// This is intended as a kind of "test double" so that we can write more
	// minimal unit tests that can avoid relying on too many language features
	// all at once, so that hopefully future maintenance will not require
	// making broad changes across many different tests at once, which would
	// then risk inadvertently treating a regression as expected behavior.
	//
	// Configurations that refer to test-only globals are not valid for use
	// outside of the test suite of this package.
	TestOnlyGlobals map[string]cty.Value
}

// SetTestOnlyGlobals assigns the test-only globals map for the receiving
// main evaluator.
//
// This may be used only from unit tests in this package and must be called
// before performing any other operations against the reciever. It's invalid
// to change the test-only globals after some evaluation has already been
// performed, because the evaluator expects its input to be immutable and
// caches values derived from that input, and there's no mechanism to
// invalidate those caches.
//
// This is intentionally defined in a _test.go file to prevent it from
// being used from non-test code, despite being named as if it's exported.
// It's named as if exported to help differentiate it from unexported
// methods that are intended only as internal API, since it's a public API
// from the perspective of a test caller even though it's not public to
// other callers.
func (m *Main) SetTestOnlyGlobals(t *testing.T, vals map[string]cty.Value) {
	m.testOnlyGlobals = vals
}

func assertNoDiags(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics\n%s", diags.Err())
	}
}

func assertMatchingDiag(t *testing.T, diags tfdiags.Diagnostics, check func(diag tfdiags.Diagnostic) bool) {
	t.Helper()
	for _, diag := range diags {
		if check(diag) {
			return
		}
	}
	t.Fatalf("none of the diagnostics is the one we are expecting\n%s", diags.Err())
}

// inPromisingTask is a helper for conveniently running some code in the context
// of a [promising.MainTask], with automatic promise error checking. This
// makes it valid to call functions that expect to run only as part of a
// promising task, which is true of essentially every method in this package
// that takes a [context.Context] as its first argument.
//
// Specifically, if the function encounters any direct promise-related failures,
// such as failure to resolve a promise before returning, this function will
// halt the test with an error message.
func inPromisingTask(t *testing.T, f func(ctx context.Context, t *testing.T)) {
	t.Helper()
	_, err := promising.MainTask(context.Background(), func(ctx context.Context) (struct{}, error) {
		t.Helper()
		f(ctx, t)
		return struct{}{}, nil
	})
	if err != nil {
		// We could get here if the test produces any self-references or
		// if it creates any promises that are left unresolved once it exits.
		t.Fatalf("promise resolution failure: %s", err)
	}
}

// subtestInPromisingTask compiles [testing.T.Run] with [inPromisingTask] as
// a convenience wrapper for running an entire subtest as a [promising.MainTask].
func subtestInPromisingTask(t *testing.T, name string, f func(ctx context.Context, t *testing.T)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		inPromisingTask(t, f)
	})
}
