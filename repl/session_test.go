package repl

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

func TestSession_basicState(t *testing.T) {
	state := &terraform.State{
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"id": "bar",
							},
						},
					},
				},
			},

			&terraform.ModuleState{
				Path: []string{"root", "module"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"id": "bar",
							},
						},
					},
				},
			},
		},
	}

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
					Input:         "exit",
					Error:         true,
					ErrorContains: ErrSessionExit.Error(),
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
					ErrorContains: "'test_instance.bar' not found",
				},
			},
		})
	})
}

func testSession(t *testing.T, test testSessionTest) {
	// Build the TF context
	ctx, err := terraform.NewContext(&terraform.ContextOpts{
		State:  test.State,
		Module: module.NewEmptyTree(),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Build the session
	s := &Session{
		Interpolater: ctx.Interpolater(),
	}

	// Test the inputs. We purposely don't use subtests here because
	// the inputs don't recognize subtests, but a sequence of stateful
	// operations.
	for _, input := range test.Inputs {
		result, err := s.Handle(input.Input)
		if (err != nil) != input.Error {
			t.Fatalf("%q: err: %s", input.Input, err)
		}
		if err != nil {
			if input.ErrorContains != "" {
				if !strings.Contains(err.Error(), input.ErrorContains) {
					t.Fatalf(
						"%q: err should contain: %q\n\n%s",
						input.Input, input.ErrorContains, err)
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
	State  *terraform.State // State to use
	Module string           // Module name in test-fixtures to load

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
	ErrorContains  string
}
