package views

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

func TestOperation_stopping(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOperation(arguments.ViewHuman, false, NewView(streams))

	v.Stopping()

	if got, want := done(t).Stdout(), "Stopping operation...\n"; got != want {
		t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
	}
}

func TestOperation_cancelled(t *testing.T) {
	testCases := map[string]struct {
		destroy bool
		want    string
	}{
		"apply": {
			destroy: false,
			want:    "Apply cancelled.\n",
		},
		"destroy": {
			destroy: true,
			want:    "Destroy cancelled.\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOperation(arguments.ViewHuman, false, NewView(streams))

			v.Cancelled(tc.destroy)

			if got, want := done(t).Stdout(), tc.want; got != want {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
			}
		})
	}
}

func TestOperation_emergencyDumpState(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOperation(arguments.ViewHuman, false, NewView(streams))

	stateFile := statefile.New(nil, "foo", 1)

	err := v.EmergencyDumpState(stateFile)
	if err != nil {
		t.Fatalf("unexpected error dumping state: %s", err)
	}

	// Check that the result (on stderr) looks like JSON state
	raw := done(t).Stderr()
	var state map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatalf("unexpected error parsing dumped state: %s\nraw:\n%s", err, raw)
	}
}

func TestOperation_planNoChanges(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOperation(arguments.ViewHuman, false, NewView(streams))

	v.PlanNoChanges()

	if got, want := done(t).Stdout(), "No changes. Infrastructure is up-to-date."; !strings.Contains(got, want) {
		t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
	}
}

func TestOperation_plan(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOperation(arguments.ViewHuman, true, NewView(streams))

	plan := testPlan(t)
	state := states.NewState()
	schemas := testSchemas()
	v.Plan(plan, state, schemas)

	want := `
Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # test_resource.foo will be created
  + resource "test_resource" "foo" {
      + foo = "bar"
      + id  = (known after apply)
    }

Plan: 1 to add, 0 to change, 0 to destroy.
`

	if got := done(t).Stdout(); got != want {
		t.Errorf("unexpected output\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestOperation_planNextStep(t *testing.T) {
	testCases := map[string]struct {
		path string
		want string
	}{
		"no state path": {
			path: "",
			want: "You didn't use the -out option",
		},
		"state path": {
			path: "good plan.tfplan",
			want: `terraform apply "good plan.tfplan"`,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOperation(arguments.ViewHuman, false, NewView(streams))

			v.PlanNextStep(tc.path)

			if got := done(t).Stdout(); !strings.Contains(got, tc.want) {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// The in-automation state is on the view itself, so testing it separately is
// clearer.
func TestOperation_planNextStepInAutomation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOperation(arguments.ViewHuman, true, NewView(streams))

	v.PlanNextStep("")

	if got := done(t).Stdout(); got != "" {
		t.Errorf("unexpected output\ngot: %q", got)
	}
}
