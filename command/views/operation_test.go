package views

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/plans"
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
		planMode plans.Mode
		want     string
	}{
		"apply": {
			planMode: plans.NormalMode,
			want:     "Apply cancelled.\n",
		},
		"destroy": {
			planMode: plans.DestroyMode,
			want:     "Destroy cancelled.\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOperation(arguments.ViewHuman, false, NewView(streams))

			v.Cancelled(tc.planMode)

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

// Test all the trivial OperationJSON methods together. Y'know, for brevity.
// This test is not a realistic stream of messages.
func TestOperationJSON_logs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	v.Cancelled(plans.NormalMode)
	v.Cancelled(plans.DestroyMode)
	v.Stopping()
	v.Interrupted()
	v.FatalInterrupt()

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "Apply cancelled",
			"@module":  "terraform.ui",
			"type":     "log",
		},
		{
			"@level":   "info",
			"@message": "Destroy cancelled",
			"@module":  "terraform.ui",
			"type":     "log",
		},
		{
			"@level":   "info",
			"@message": "Stopping operation...",
			"@module":  "terraform.ui",
			"type":     "log",
		},
		{
			"@level":   "info",
			"@message": interrupted,
			"@module":  "terraform.ui",
			"type":     "log",
		},
		{
			"@level":   "info",
			"@message": fatalInterrupt,
			"@module":  "terraform.ui",
			"type":     "log",
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

// This is a fairly circular test, but it's such a rarely executed code path
// that I think it's probably still worth having. We're not testing against
// a fixed state JSON output because this test ought not fail just because
// we upgrade state format in the future.
func TestOperationJSON_emergencyDumpState(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	stateFile := statefile.New(nil, "foo", 1)
	stateBuf := new(bytes.Buffer)
	err := statefile.Write(stateFile, stateBuf)
	if err != nil {
		t.Fatal(err)
	}
	var stateJSON map[string]interface{}
	err = json.Unmarshal(stateBuf.Bytes(), &stateJSON)
	if err != nil {
		t.Fatal(err)
	}

	err = v.EmergencyDumpState(stateFile)
	if err != nil {
		t.Fatalf("unexpected error dumping state: %s", err)
	}

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "Emergency state dump",
			"@module":  "terraform.ui",
			"type":     "log",
			"state":    stateJSON,
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestOperationJSON_planNoChanges(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	v.PlanNoChanges()

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "Plan: 0 to add, 0 to change, 0 to destroy.",
			"@module":  "terraform.ui",
			"type":     "change_summary",
			"changes": map[string]interface{}{
				"operation": "plan",
				"add":       float64(0),
				"change":    float64(0),
				"remove":    float64(0),
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestOperationJSON_plan(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	root := addrs.RootModuleInstance
	vpc, diags := addrs.ParseModuleInstanceStr("module.vpc")
	if len(diags) > 0 {
		t.Fatal(diags.Err())
	}
	boop := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_instance", Name: "boop"}
	beep := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_instance", Name: "beep"}
	derp := addrs.Resource{Mode: addrs.DataResourceMode, Type: "test_source", Name: "derp"}

	plan := &plans.Plan{
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr:      boop.Instance(addrs.IntKey(0)).Absolute(root),
					ChangeSrc: plans.ChangeSrc{Action: plans.CreateThenDelete},
				},
				{
					Addr:      boop.Instance(addrs.IntKey(1)).Absolute(root),
					ChangeSrc: plans.ChangeSrc{Action: plans.Create},
				},
				{
					Addr:      boop.Instance(addrs.IntKey(0)).Absolute(vpc),
					ChangeSrc: plans.ChangeSrc{Action: plans.Delete},
				},
				{
					Addr:      beep.Instance(addrs.NoKey).Absolute(root),
					ChangeSrc: plans.ChangeSrc{Action: plans.DeleteThenCreate},
				},
				{
					Addr:      beep.Instance(addrs.NoKey).Absolute(vpc),
					ChangeSrc: plans.ChangeSrc{Action: plans.Update},
				},
				// Data source deletion should not show up in the logs
				{
					Addr:      derp.Instance(addrs.NoKey).Absolute(root),
					ChangeSrc: plans.ChangeSrc{Action: plans.Delete},
				},
			},
		},
	}
	v.Plan(plan, nil, nil)

	want := []map[string]interface{}{
		// Create-then-delete should result in replace
		{
			"@level":   "info",
			"@message": "test_instance.boop[0]: Plan to replace",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "replace",
				"resource": map[string]interface{}{
					"addr":             `test_instance.boop[0]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_instance.boop[0]`,
					"resource_key":     float64(0),
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
			},
		},
		// Simple create
		{
			"@level":   "info",
			"@message": "test_instance.boop[1]: Plan to create",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "create",
				"resource": map[string]interface{}{
					"addr":             `test_instance.boop[1]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_instance.boop[1]`,
					"resource_key":     float64(1),
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
			},
		},
		// Simple delete
		{
			"@level":   "info",
			"@message": "module.vpc.test_instance.boop[0]: Plan to delete",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "delete",
				"resource": map[string]interface{}{
					"addr":             `module.vpc.test_instance.boop[0]`,
					"implied_provider": "test",
					"module":           "module.vpc",
					"resource":         `test_instance.boop[0]`,
					"resource_key":     float64(0),
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
			},
		},
		// Delete-then-create is also a replace
		{
			"@level":   "info",
			"@message": "test_instance.beep: Plan to replace",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "replace",
				"resource": map[string]interface{}{
					"addr":             `test_instance.beep`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_instance.beep`,
					"resource_key":     nil,
					"resource_name":    "beep",
					"resource_type":    "test_instance",
				},
			},
		},
		// Simple update
		{
			"@level":   "info",
			"@message": "module.vpc.test_instance.beep: Plan to update",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "update",
				"resource": map[string]interface{}{
					"addr":             `module.vpc.test_instance.beep`,
					"implied_provider": "test",
					"module":           "module.vpc",
					"resource":         `test_instance.beep`,
					"resource_key":     nil,
					"resource_name":    "beep",
					"resource_type":    "test_instance",
				},
			},
		},
		// These counts are 3 add/1 change/3 destroy because the replace
		// changes result in both add and destroy counts.
		{
			"@level":   "info",
			"@message": "Plan: 3 to add, 1 to change, 3 to destroy.",
			"@module":  "terraform.ui",
			"type":     "change_summary",
			"changes": map[string]interface{}{
				"operation": "plan",
				"add":       float64(3),
				"change":    float64(1),
				"remove":    float64(3),
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestOperationJSON_plannedChange(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	root := addrs.RootModuleInstance
	boop := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_instance", Name: "boop"}
	derp := addrs.Resource{Mode: addrs.DataResourceMode, Type: "test_source", Name: "derp"}

	// Replace requested by user
	v.PlannedChange(&plans.ResourceInstanceChangeSrc{
		Addr:         boop.Instance(addrs.IntKey(0)).Absolute(root),
		ChangeSrc:    plans.ChangeSrc{Action: plans.DeleteThenCreate},
		ActionReason: plans.ResourceInstanceReplaceByRequest,
	})

	// Simple create
	v.PlannedChange(&plans.ResourceInstanceChangeSrc{
		Addr:      boop.Instance(addrs.IntKey(1)).Absolute(root),
		ChangeSrc: plans.ChangeSrc{Action: plans.Create},
	})

	// Data source deletion
	v.PlannedChange(&plans.ResourceInstanceChangeSrc{
		Addr:      derp.Instance(addrs.NoKey).Absolute(root),
		ChangeSrc: plans.ChangeSrc{Action: plans.Delete},
	})

	// Expect only two messages, as the data source deletion should be a no-op
	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "test_instance.boop[0]: Plan to replace",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "replace",
				"reason": "requested",
				"resource": map[string]interface{}{
					"addr":             `test_instance.boop[0]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_instance.boop[0]`,
					"resource_key":     float64(0),
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop[1]: Plan to create",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "create",
				"resource": map[string]interface{}{
					"addr":             `test_instance.boop[1]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_instance.boop[1]`,
					"resource_key":     float64(1),
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}
