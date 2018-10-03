package repl

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

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
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(addrs.RootModuleInstance),
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
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(addrs.RootModuleInstance),
		)
	})

	t.Run("basic", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:  "test_instance.foo.id",
					Output: "bar",
				},
			},
		})
	})

	t.Run("resource count", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:  "test_instance.foo.count",
					Output: "1",
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
					ErrorContains: "'test_instance.bar' not found",
				},
			},
		})
	})

	t.Run("missing module", func(t *testing.T) {
		testSession(t, testSessionTest{
			State: state,
			Inputs: []testSessionInput{
				{
					Input:         "module.child.foo",
					Error:         true,
					ErrorContains: "Couldn't find module \"child\"",
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
					ErrorContains: "Couldn't find output \"foo\"",
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
	p := &terraform.MockProvider{}
	p.GetSchemaReturn = &terraform.ProviderSchema{}

	// Build the TF context
	ctx, diags := terraform.NewContext(&terraform.ContextOpts{
		State: test.State,
		ProviderResolver: providers.ResolverFixed(map[string]providers.Factory{
			"aws": providers.FactoryFixed(p),
		}),
		Config: configs.NewEmptyConfig(),
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
	Module string        // Module name in test-fixtures to load

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
