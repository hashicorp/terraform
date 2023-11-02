package stackeval

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
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

	sources, err := sourcebundle.OpenDir("testdata/sourcebundle")
	if err != nil {
		t.Fatalf("cannot open source bundle: %s", err)
	}

	ret, diags := stackconfig.LoadConfigDir(fakeSrc, sources)
	if diags.HasErrors() {
		t.Fatalf("configuration is invalid\n%s", diags.Err().Error())
	}
	return ret
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
}
