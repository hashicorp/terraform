// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package repl

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestSession_basicState(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"bar"}`),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("module", addrs.NoKey)),
			&states.ResourceInstanceObjectSrc{
				Status:    states.ObjectReady,
				AttrsJSON: []byte(`{"id":"bar"}`),
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	t.Run("basic", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:  "test_instance.foo.id",
					Output: `"bar"`,
				},
			},
		})
	})

	t.Run("missing resource", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:         "test_instance.bar.id",
					Error:         true,
					ErrorContains: `A managed resource "test_instance" "bar" has not been declared`,
				},
			},
		})
	})

	t.Run("missing module", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:         "module.child",
					Error:         true,
					ErrorContains: `No module call named "child" is declared in the root module.`,
				},
			},
		})
	})

	t.Run("missing module referencing just one output", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:         "module.child.foo",
					Error:         true,
					ErrorContains: `No module call named "child" is declared in the root module.`,
				},
			},
		})
	})

	t.Run("missing module output", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:         "module.module.foo",
					Error:         true,
					ErrorContains: `Unsupported attribute: This object does not have an attribute named "foo"`,
				},
			},
		})
	})

	t.Run("type function", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input: "type(test_instance.foo)",
					Output: `object({
    id: string,
})`,
				},
			},
		})
	})
}

func TestSession_stateless(t *testing.T) {
	t.Run("exit", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input: "exit",
					Exit:  true,
				},
			},
		})
	})

	t.Run("help", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:          "help",
					OutputContains: "allows you to",
				},
			},
		})
	})

	t.Run("help with spaces", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:          "help   ",
					OutputContains: "allows you to",
				},
			},
		})
	})

	t.Run("basic math", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:  "1 + 5",
					Output: "6",
				},
			},
		})
	})

	t.Run("missing resource", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:         "test_instance.bar.id",
					Error:         true,
					ErrorContains: `resource "test_instance" "bar" has not been declared`,
				},
			},
		})
	})

	t.Run("type function", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:  `type("foo")`,
					Output: "string",
				},
			},
		})
	})

	t.Run("type type is type", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:  `type(type("foo"))`,
					Output: "type",
				},
			},
		})
	})

	t.Run("interpolating type with strings is not possible", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:         `"quin${type([])}"`,
					Error:         true,
					ErrorContains: "Invalid template interpolation value",
				},
			},
		})
	})

	t.Run("type function cannot be used in expressions", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:         `[for i in [1, "two", true]: type(i)]`,
					Output:        "",
					Error:         true,
					ErrorContains: "Invalid use of type function",
				},
			},
		})
	})

	t.Run("type equality checks are not permitted", func(t *testing.T) {
		testSession(t, testSessionTest{
			Inputs: []testSessionInput{
				{
					Input:         `type("foo") == type("bar")`,
					Output:        "",
					Error:         true,
					ErrorContains: "Invalid use of type function",
				},
			},
		})
	})
}

func testSession(t *testing.T, test testSessionTest) {
	t.Helper()

	p := &testing_provider.MockProvider{}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Computed: true},
					},
				},
			},
		},
	}

	config, _, cleanup, configDiags := initwd.LoadConfigForTests(t, "testdata/config-fixture", "tests")
	defer cleanup()
	if configDiags.HasErrors() {
		t.Fatalf("unexpected problems loading config: %s", configDiags.Err())
	}

	// Build the TF context
	ctx, diags := terraform.NewContext(&terraform.ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): providers.FactoryFixed(p),
		},
	})
	if diags.HasErrors() {
		t.Fatalf("failed to create context: %s", diags.Err())
	}

	state := test.State
	if state == nil {
		state = states.NewState()
	}
	scope, diags := ctx.Eval(config, state, addrs.RootModuleInstance, &terraform.EvalOpts{})
	if diags.HasErrors() {
		t.Fatalf("failed to create scope: %s", diags.Err())
	}

	// Ensure that any console-only functions are available
	scope.ConsoleMode = true

	// Build the session
	s := &Session{
		Scope: scope,
	}

	// Test the inputs. We purposely don't use subtests here because
	// the inputs don't represent subtests, but a sequence of stateful
	// operations.
	for _, input := range test.Inputs {
		result, exit, diags := s.Handle(input.Input)
		if exit != input.Exit {
			t.Fatalf("incorrect 'exit' result %t; want %t", exit, input.Exit)
		}
		if (diags.HasErrors()) != input.Error {
			t.Fatalf("%q: unexpected errors: %s", input.Input, diags.Err())
		}
		if diags.HasErrors() {
			if input.ErrorContains != "" {
				if !strings.Contains(diags.Err().Error(), input.ErrorContains) {
					t.Fatalf(
						"%q: diagnostics should contain: %q\n\n%s",
						input.Input, input.ErrorContains, diags.Err(),
					)
				}
			}

			continue
		}

		if input.Output != "" && result != input.Output {
			t.Fatalf(
				"%q: expected:\n\n%s\n\ngot:\n\n%s",
				input.Input, input.Output, result)
		}

		if input.OutputContains != "" && !strings.Contains(result, input.OutputContains) {
			t.Fatalf(
				"%q: expected contains:\n\n%s\n\ngot:\n\n%s",
				input.Input, input.OutputContains, result)
		}
	}
}

type testSessionTest struct {
	State  *states.State // State to use
	Module string        // Module name in testdata to load

	// Inputs are the list of test inputs that are run in order.
	// Each input can test the output of each step.
	Inputs []testSessionInput
}

// testSessionInput is a single input to test for a session.
type testSessionInput struct {
	Input          string // Input string
	Output         string // Exact output string to check
	OutputContains string
	Error          bool // Error is true if error is expected
	Exit           bool // Exit is true if exiting is expected
	ErrorContains  string
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  string
	}{
		// Primititves
		{
			cty.StringVal("a"),
			"string",
		},
		{
			cty.NumberIntVal(42),
			"number",
		},
		{
			cty.BoolVal(true),
			"bool",
		},
		// Collections
		{
			cty.EmptyObjectVal,
			`object({})`,
		},
		{
			cty.EmptyTupleVal,
			`tuple([])`,
		},
		{
			cty.ListValEmpty(cty.String),
			`list(string)`,
		},
		{
			cty.MapValEmpty(cty.String),
			`map(string)`,
		},
		{
			cty.SetValEmpty(cty.String),
			`set(string)`,
		},
		{
			cty.ListVal([]cty.Value{cty.StringVal("a")}),
			`list(string)`,
		},
		{
			cty.ListVal([]cty.Value{cty.ListVal([]cty.Value{cty.NumberIntVal(42)})}),
			`list(list(number))`,
		},
		{
			cty.ListVal([]cty.Value{cty.MapValEmpty(cty.String)}),
			`list(map(string))`,
		},
		{
			cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			})}),
			"list(\n    object({\n        foo: string,\n    }),\n)",
		},
		// Unknowns and Nulls
		{
			cty.UnknownVal(cty.String),
			"string",
		},
		{
			cty.NullVal(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			"object({\n    foo: string,\n})",
		},
		{ // irrelevant marks do nothing
			cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar").Mark("ignore me"),
			})}),
			"list(\n    object({\n        foo: string,\n    }),\n)",
		},
	}
	for _, test := range tests {
		got := typeString(test.Input.Type())
		if got != test.Want {
			t.Errorf("wrong result:\n%s", cmp.Diff(got, test.Want))
		}
	}
}
