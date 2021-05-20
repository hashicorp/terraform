package repl

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/providers"
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
}

func testSession(t *testing.T, test testSessionTest) {
	t.Helper()

	p := &terraform.MockProvider{}
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

	config, _, cleanup, configDiags := initwd.LoadConfigForTests(t, "testdata/config-fixture")
	defer cleanup()
	if configDiags.HasErrors() {
		t.Fatalf("unexpected problems loading config: %s", configDiags.Err())
	}

	// Build the TF context
	ctx, diags := terraform.NewContext(&terraform.ContextOpts{
		State: test.State,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): providers.FactoryFixed(p),
		},
		Config: config,
	})
	if diags.HasErrors() {
		t.Fatalf("failed to create context: %s", diags.Err())
	}

	scope, diags := ctx.Eval(addrs.RootModuleInstance)
	if diags.HasErrors() {
		t.Fatalf("failed to create scope: %s", diags.Err())
	}

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
