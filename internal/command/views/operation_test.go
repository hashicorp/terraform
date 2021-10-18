package views

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
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

	tests := map[string]struct {
		plan     func(schemas *terraform.Schemas) *plans.Plan
		wantText string
	}{
		"nothing at all in normal mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				return &plans.Plan{
					UIMode:  plans.NormalMode,
					Changes: plans.NewChanges(),
				}
			},
			"no differences, so no changes are needed.",
		},
		"nothing at all in refresh-only mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				return &plans.Plan{
					UIMode:  plans.RefreshOnlyMode,
					Changes: plans.NewChanges(),
				}
			},
			"Terraform has checked that the real remote objects still match",
		},
		"nothing at all in destroy mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				return &plans.Plan{
					UIMode:  plans.DestroyMode,
					Changes: plans.NewChanges(),
				}
			},
			"No objects need to be destroyed.",
		},
		"drift detected in normal mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				addr := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_resource",
					Name: "somewhere",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
				schema, _ := schemas.ResourceTypeConfig(
					addrs.NewDefaultProvider("test"),
					addr.Resource.Resource.Mode,
					addr.Resource.Resource.Type,
				)
				ty := schema.ImpliedType()
				rc := &plans.ResourceInstanceChange{
					Addr:        addr,
					PrevRunAddr: addr,
					ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault(
						addrs.NewDefaultProvider("test"),
					),
					Change: plans.Change{
						Action: plans.Update,
						Before: cty.NullVal(ty),
						After: cty.ObjectVal(map[string]cty.Value{
							"id":  cty.StringVal("1234"),
							"foo": cty.StringVal("bar"),
						}),
					},
				}
				rcs, err := rc.Encode(ty)
				if err != nil {
					panic(err)
				}
				drs := []*plans.ResourceInstanceChangeSrc{rcs}
				return &plans.Plan{
					UIMode:           plans.NormalMode,
					Changes:          plans.NewChanges(),
					DriftedResources: drs,
				}
			},
			"to update the Terraform state to match, create and apply a refresh-only plan",
		},
		"drift detected in refresh-only mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				addr := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_resource",
					Name: "somewhere",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
				schema, _ := schemas.ResourceTypeConfig(
					addrs.NewDefaultProvider("test"),
					addr.Resource.Resource.Mode,
					addr.Resource.Resource.Type,
				)
				ty := schema.ImpliedType()
				rc := &plans.ResourceInstanceChange{
					Addr:        addr,
					PrevRunAddr: addr,
					ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault(
						addrs.NewDefaultProvider("test"),
					),
					Change: plans.Change{
						Action: plans.Update,
						Before: cty.NullVal(ty),
						After: cty.ObjectVal(map[string]cty.Value{
							"id":  cty.StringVal("1234"),
							"foo": cty.StringVal("bar"),
						}),
					},
				}
				rcs, err := rc.Encode(ty)
				if err != nil {
					panic(err)
				}
				drs := []*plans.ResourceInstanceChangeSrc{rcs}
				return &plans.Plan{
					UIMode:           plans.RefreshOnlyMode,
					Changes:          plans.NewChanges(),
					DriftedResources: drs,
				}
			},
			"If you were expecting these changes then you can apply this plan",
		},
		"move-only changes in refresh-only mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				addr := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_resource",
					Name: "somewhere",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
				addrPrev := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_resource",
					Name: "anywhere",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
				schema, _ := schemas.ResourceTypeConfig(
					addrs.NewDefaultProvider("test"),
					addr.Resource.Resource.Mode,
					addr.Resource.Resource.Type,
				)
				ty := schema.ImpliedType()
				rc := &plans.ResourceInstanceChange{
					Addr:        addr,
					PrevRunAddr: addrPrev,
					ProviderAddr: addrs.RootModuleInstance.ProviderConfigDefault(
						addrs.NewDefaultProvider("test"),
					),
					Change: plans.Change{
						Action: plans.NoOp,
						Before: cty.ObjectVal(map[string]cty.Value{
							"id":  cty.StringVal("1234"),
							"foo": cty.StringVal("bar"),
						}),
						After: cty.ObjectVal(map[string]cty.Value{
							"id":  cty.StringVal("1234"),
							"foo": cty.StringVal("bar"),
						}),
					},
				}
				rcs, err := rc.Encode(ty)
				if err != nil {
					panic(err)
				}
				drs := []*plans.ResourceInstanceChangeSrc{rcs}
				return &plans.Plan{
					UIMode:           plans.RefreshOnlyMode,
					Changes:          plans.NewChanges(),
					DriftedResources: drs,
				}
			},
			"test_resource.anywhere has moved to test_resource.somewhere",
		},
		"drift detected in destroy mode": {
			func(schemas *terraform.Schemas) *plans.Plan {
				return &plans.Plan{
					UIMode:  plans.DestroyMode,
					Changes: plans.NewChanges(),
					PrevRunState: states.BuildState(func(state *states.SyncState) {
						state.SetResourceInstanceCurrent(
							addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_resource",
								Name: "somewhere",
							}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
							&states.ResourceInstanceObjectSrc{
								Status:    states.ObjectReady,
								AttrsJSON: []byte(`{}`),
							},
							addrs.RootModuleInstance.ProviderConfigDefault(addrs.NewDefaultProvider("test")),
						)
					}),
					PriorState: states.NewState(),
				}
			},
			"No objects need to be destroyed.",
		},
	}

	schemas := testSchemas()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewOperation(arguments.ViewHuman, false, NewView(streams))
			plan := test.plan(schemas)
			v.Plan(plan, schemas)
			got := done(t).Stdout()
			if want := test.wantText; want != "" && !strings.Contains(got, want) {
				t.Errorf("missing expected message\ngot:\n%s\n\nwant substring: %s", got, want)
			}
		})
	}
}

func TestOperation_plan(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewOperation(arguments.ViewHuman, true, NewView(streams))

	plan := testPlan(t)
	schemas := testSchemas()
	v.Plan(plan, schemas)

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

	plan := &plans.Plan{
		Changes: plans.NewChanges(),
	}
	v.Plan(plan, nil)

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
	boop := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "boop"}
	beep := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "beep"}
	derp := addrs.Resource{Mode: addrs.DataResourceMode, Type: "test_source", Name: "derp"}

	plan := &plans.Plan{
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr:        boop.Instance(addrs.IntKey(0)).Absolute(root),
					PrevRunAddr: boop.Instance(addrs.IntKey(0)).Absolute(root),
					ChangeSrc:   plans.ChangeSrc{Action: plans.CreateThenDelete},
				},
				{
					Addr:        boop.Instance(addrs.IntKey(1)).Absolute(root),
					PrevRunAddr: boop.Instance(addrs.IntKey(1)).Absolute(root),
					ChangeSrc:   plans.ChangeSrc{Action: plans.Create},
				},
				{
					Addr:        boop.Instance(addrs.IntKey(0)).Absolute(vpc),
					PrevRunAddr: boop.Instance(addrs.IntKey(0)).Absolute(vpc),
					ChangeSrc:   plans.ChangeSrc{Action: plans.Delete},
				},
				{
					Addr:        beep.Instance(addrs.NoKey).Absolute(root),
					PrevRunAddr: beep.Instance(addrs.NoKey).Absolute(root),
					ChangeSrc:   plans.ChangeSrc{Action: plans.DeleteThenCreate},
				},
				{
					Addr:        beep.Instance(addrs.NoKey).Absolute(vpc),
					PrevRunAddr: beep.Instance(addrs.NoKey).Absolute(vpc),
					ChangeSrc:   plans.ChangeSrc{Action: plans.Update},
				},
				// Data source deletion should not show up in the logs
				{
					Addr:        derp.Instance(addrs.NoKey).Absolute(root),
					PrevRunAddr: derp.Instance(addrs.NoKey).Absolute(root),
					ChangeSrc:   plans.ChangeSrc{Action: plans.Delete},
				},
			},
		},
	}
	v.Plan(plan, testSchemas())

	want := []map[string]interface{}{
		// Create-then-delete should result in replace
		{
			"@level":   "info",
			"@message": "test_resource.boop[0]: Plan to replace",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "replace",
				"resource": map[string]interface{}{
					"addr":             `test_resource.boop[0]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.boop[0]`,
					"resource_key":     float64(0),
					"resource_name":    "boop",
					"resource_type":    "test_resource",
				},
			},
		},
		// Simple create
		{
			"@level":   "info",
			"@message": "test_resource.boop[1]: Plan to create",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "create",
				"resource": map[string]interface{}{
					"addr":             `test_resource.boop[1]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.boop[1]`,
					"resource_key":     float64(1),
					"resource_name":    "boop",
					"resource_type":    "test_resource",
				},
			},
		},
		// Simple delete
		{
			"@level":   "info",
			"@message": "module.vpc.test_resource.boop[0]: Plan to delete",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "delete",
				"resource": map[string]interface{}{
					"addr":             `module.vpc.test_resource.boop[0]`,
					"implied_provider": "test",
					"module":           "module.vpc",
					"resource":         `test_resource.boop[0]`,
					"resource_key":     float64(0),
					"resource_name":    "boop",
					"resource_type":    "test_resource",
				},
			},
		},
		// Delete-then-create is also a replace
		{
			"@level":   "info",
			"@message": "test_resource.beep: Plan to replace",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "replace",
				"resource": map[string]interface{}{
					"addr":             `test_resource.beep`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.beep`,
					"resource_key":     nil,
					"resource_name":    "beep",
					"resource_type":    "test_resource",
				},
			},
		},
		// Simple update
		{
			"@level":   "info",
			"@message": "module.vpc.test_resource.beep: Plan to update",
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "update",
				"resource": map[string]interface{}{
					"addr":             `module.vpc.test_resource.beep`,
					"implied_provider": "test",
					"module":           "module.vpc",
					"resource":         `test_resource.beep`,
					"resource_key":     nil,
					"resource_name":    "beep",
					"resource_type":    "test_resource",
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

func TestOperationJSON_planDriftWithMove(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	root := addrs.RootModuleInstance
	boop := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "boop"}
	beep := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "beep"}
	blep := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "blep"}
	honk := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "honk"}

	plan := &plans.Plan{
		UIMode: plans.NormalMode,
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{
				{
					Addr:        honk.Instance(addrs.StringKey("bonk")).Absolute(root),
					PrevRunAddr: honk.Instance(addrs.IntKey(0)).Absolute(root),
					ChangeSrc:   plans.ChangeSrc{Action: plans.NoOp},
				},
			},
		},
		DriftedResources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr:        beep.Instance(addrs.NoKey).Absolute(root),
				PrevRunAddr: beep.Instance(addrs.NoKey).Absolute(root),
				ChangeSrc:   plans.ChangeSrc{Action: plans.Delete},
			},
			{
				Addr:        boop.Instance(addrs.NoKey).Absolute(root),
				PrevRunAddr: blep.Instance(addrs.NoKey).Absolute(root),
				ChangeSrc:   plans.ChangeSrc{Action: plans.Update},
			},
			// Move-only resource drift should not be present in normal mode plans
			{
				Addr:        honk.Instance(addrs.StringKey("bonk")).Absolute(root),
				PrevRunAddr: honk.Instance(addrs.IntKey(0)).Absolute(root),
				ChangeSrc:   plans.ChangeSrc{Action: plans.NoOp},
			},
		},
	}
	v.Plan(plan, testSchemas())

	want := []map[string]interface{}{
		// Drift detected: delete
		{
			"@level":   "info",
			"@message": "test_resource.beep: Drift detected (delete)",
			"@module":  "terraform.ui",
			"type":     "resource_drift",
			"change": map[string]interface{}{
				"action": "delete",
				"resource": map[string]interface{}{
					"addr":             "test_resource.beep",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_resource.beep",
					"resource_key":     nil,
					"resource_name":    "beep",
					"resource_type":    "test_resource",
				},
			},
		},
		// Drift detected: update with move
		{
			"@level":   "info",
			"@message": "test_resource.boop: Drift detected (update)",
			"@module":  "terraform.ui",
			"type":     "resource_drift",
			"change": map[string]interface{}{
				"action": "update",
				"resource": map[string]interface{}{
					"addr":             "test_resource.boop",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_resource.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_resource",
				},
				"previous_resource": map[string]interface{}{
					"addr":             "test_resource.blep",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_resource.blep",
					"resource_key":     nil,
					"resource_name":    "blep",
					"resource_type":    "test_resource",
				},
			},
		},
		// Move-only change
		{
			"@level":   "info",
			"@message": `test_resource.honk["bonk"]: Plan to move`,
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "move",
				"resource": map[string]interface{}{
					"addr":             `test_resource.honk["bonk"]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.honk["bonk"]`,
					"resource_key":     "bonk",
					"resource_name":    "honk",
					"resource_type":    "test_resource",
				},
				"previous_resource": map[string]interface{}{
					"addr":             `test_resource.honk[0]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.honk[0]`,
					"resource_key":     float64(0),
					"resource_name":    "honk",
					"resource_type":    "test_resource",
				},
			},
		},
		// No changes
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

func TestOperationJSON_planDriftWithMoveRefreshOnly(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	root := addrs.RootModuleInstance
	boop := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "boop"}
	beep := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "beep"}
	blep := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "blep"}
	honk := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "honk"}

	plan := &plans.Plan{
		UIMode: plans.RefreshOnlyMode,
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{},
		},
		DriftedResources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr:        beep.Instance(addrs.NoKey).Absolute(root),
				PrevRunAddr: beep.Instance(addrs.NoKey).Absolute(root),
				ChangeSrc:   plans.ChangeSrc{Action: plans.Delete},
			},
			{
				Addr:        boop.Instance(addrs.NoKey).Absolute(root),
				PrevRunAddr: blep.Instance(addrs.NoKey).Absolute(root),
				ChangeSrc:   plans.ChangeSrc{Action: plans.Update},
			},
			// Move-only resource drift should be present in refresh-only plans
			{
				Addr:        honk.Instance(addrs.StringKey("bonk")).Absolute(root),
				PrevRunAddr: honk.Instance(addrs.IntKey(0)).Absolute(root),
				ChangeSrc:   plans.ChangeSrc{Action: plans.NoOp},
			},
		},
	}
	v.Plan(plan, testSchemas())

	want := []map[string]interface{}{
		// Drift detected: delete
		{
			"@level":   "info",
			"@message": "test_resource.beep: Drift detected (delete)",
			"@module":  "terraform.ui",
			"type":     "resource_drift",
			"change": map[string]interface{}{
				"action": "delete",
				"resource": map[string]interface{}{
					"addr":             "test_resource.beep",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_resource.beep",
					"resource_key":     nil,
					"resource_name":    "beep",
					"resource_type":    "test_resource",
				},
			},
		},
		// Drift detected: update
		{
			"@level":   "info",
			"@message": "test_resource.boop: Drift detected (update)",
			"@module":  "terraform.ui",
			"type":     "resource_drift",
			"change": map[string]interface{}{
				"action": "update",
				"resource": map[string]interface{}{
					"addr":             "test_resource.boop",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_resource.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_resource",
				},
				"previous_resource": map[string]interface{}{
					"addr":             "test_resource.blep",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_resource.blep",
					"resource_key":     nil,
					"resource_name":    "blep",
					"resource_type":    "test_resource",
				},
			},
		},
		// Drift detected: Move-only change
		{
			"@level":   "info",
			"@message": `test_resource.honk["bonk"]: Drift detected (move)`,
			"@module":  "terraform.ui",
			"type":     "resource_drift",
			"change": map[string]interface{}{
				"action": "move",
				"resource": map[string]interface{}{
					"addr":             `test_resource.honk["bonk"]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.honk["bonk"]`,
					"resource_key":     "bonk",
					"resource_name":    "honk",
					"resource_type":    "test_resource",
				},
				"previous_resource": map[string]interface{}{
					"addr":             `test_resource.honk[0]`,
					"implied_provider": "test",
					"module":           "",
					"resource":         `test_resource.honk[0]`,
					"resource_key":     float64(0),
					"resource_name":    "honk",
					"resource_type":    "test_resource",
				},
			},
		},
		// No changes
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

func TestOperationJSON_planOutputChanges(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := &OperationJSON{view: NewJSONView(NewView(streams))}

	root := addrs.RootModuleInstance

	plan := &plans.Plan{
		Changes: &plans.Changes{
			Resources: []*plans.ResourceInstanceChangeSrc{},
			Outputs: []*plans.OutputChangeSrc{
				{
					Addr: root.OutputValue("boop"),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.NoOp,
					},
				},
				{
					Addr: root.OutputValue("beep"),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
				{
					Addr: root.OutputValue("bonk"),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Delete,
					},
				},
				{
					Addr: root.OutputValue("honk"),
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Update,
					},
					Sensitive: true,
				},
			},
		},
	}
	v.Plan(plan, testSchemas())

	want := []map[string]interface{}{
		// No resource changes
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
		// Output changes
		{
			"@level":   "info",
			"@message": "Outputs: 4",
			"@module":  "terraform.ui",
			"type":     "outputs",
			"outputs": map[string]interface{}{
				"boop": map[string]interface{}{
					"action":    "noop",
					"sensitive": false,
				},
				"beep": map[string]interface{}{
					"action":    "create",
					"sensitive": false,
				},
				"bonk": map[string]interface{}{
					"action":    "delete",
					"sensitive": false,
				},
				"honk": map[string]interface{}{
					"action":    "update",
					"sensitive": true,
				},
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
		PrevRunAddr:  boop.Instance(addrs.IntKey(0)).Absolute(root),
		ChangeSrc:    plans.ChangeSrc{Action: plans.DeleteThenCreate},
		ActionReason: plans.ResourceInstanceReplaceByRequest,
	})

	// Simple create
	v.PlannedChange(&plans.ResourceInstanceChangeSrc{
		Addr:        boop.Instance(addrs.IntKey(1)).Absolute(root),
		PrevRunAddr: boop.Instance(addrs.IntKey(1)).Absolute(root),
		ChangeSrc:   plans.ChangeSrc{Action: plans.Create},
	})

	// Data source deletion
	v.PlannedChange(&plans.ResourceInstanceChangeSrc{
		Addr:        derp.Instance(addrs.NoKey).Absolute(root),
		PrevRunAddr: derp.Instance(addrs.NoKey).Absolute(root),
		ChangeSrc:   plans.ChangeSrc{Action: plans.Delete},
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
