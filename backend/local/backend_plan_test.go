package local

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

func TestLocal_planBasic(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", planFixtureSchema())

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.PlanRefresh = true

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	if !p.PlanResourceChangeCalled {
		t.Fatal("PlanResourceChange should be called")
	}
}

func TestLocal_planInAutomation(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	TestLocalProvider(t, b, "test", planFixtureSchema())

	const msg = `You didn't specify an "-out" parameter`

	// When we're "in automation" we omit certain text from the
	// plan output. However, testing for the absense of text is
	// unreliable in the face of future copy changes, so we'll
	// mitigate that by running both with and without the flag
	// set so we can ensure that the expected messages _are_
	// included the first time.
	b.RunningInAutomation = false
	b.CLI = cli.NewMockUi()
	{
		op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
		defer configCleanup()
		op.PlanRefresh = true

		run, err := b.Operation(context.Background(), op)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		<-run.Done()
		if run.Result != backend.OperationSuccess {
			t.Fatalf("plan operation failed")
		}

		output := b.CLI.(*cli.MockUi).OutputWriter.String()
		if !strings.Contains(output, msg) {
			t.Fatalf("missing next-steps message when not in automation")
		}
	}

	// On the second run, we expect the next-steps messaging to be absent
	// since we're now "running in automation".
	b.RunningInAutomation = true
	b.CLI = cli.NewMockUi()
	{
		op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
		defer configCleanup()
		op.PlanRefresh = true

		run, err := b.Operation(context.Background(), op)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		<-run.Done()
		if run.Result != backend.OperationSuccess {
			t.Fatalf("plan operation failed")
		}

		output := b.CLI.(*cli.MockUi).OutputWriter.String()
		if strings.Contains(output, msg) {
			t.Fatalf("next-steps message present when in automation")
		}
	}

}

func TestLocal_planNoConfig(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	TestLocalProvider(t, b, "test", &terraform.ProviderSchema{})

	b.CLI = cli.NewMockUi()

	op, configCleanup := testOperationPlan(t, "./test-fixtures/empty")
	defer configCleanup()
	op.PlanRefresh = true

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()

	if run.Result == backend.OperationSuccess {
		t.Fatal("plan operation succeeded; want failure")
	}
	output := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if !strings.Contains(output, "configuration") {
		t.Fatalf("bad: %s", err)
	}
}

func TestLocal_planTainted(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState_tainted())
	b.CLI = cli.NewMockUi()
	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")
	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.PlanRefresh = true
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		// Just a placeholder so that we can generate a valid plan file.
		Type:   "local",
		Config: cfgRaw,
	}
	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}
	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}
	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	expectedOutput := `An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
-/+ destroy and then create replacement

Terraform will perform the following actions:

  # test_instance.foo is tainted, so must be replaced
-/+ resource "test_instance" "foo" {
        ami = "bar"

        network_interface {
            description  = "Main network interface"
            device_index = 0
        }
    }

Plan: 1 to add, 0 to change, 1 to destroy.`
	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("Unexpected output:\n%s", output)
	}
}

func TestLocal_planDeposedOnly(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, states.BuildState(func(ss *states.SyncState) {
		ss.SetResourceInstanceDeposed(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			states.DeposedKey("00000000"),
			&states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady,
				AttrsJSON: []byte(`{
				"ami": "bar",
				"network_interface": [{
					"device_index": 0,
					"description": "Main network interface"
				}]
			}`),
			},
			addrs.ProviderConfig{
				Type: "test",
			}.Absolute(addrs.RootModuleInstance),
		)
	}))
	b.CLI = cli.NewMockUi()
	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")
	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.PlanRefresh = true
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		// Just a placeholder so that we can generate a valid plan file.
		Type:   "local",
		Config: cfgRaw,
	}
	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}
	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}
	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	// The deposed object and the current object are distinct, so our
	// plan includes separate actions for each of them. This strange situation
	// is not common: it should arise only if Terraform fails during
	// a create-before-destroy when the create hasn't completed yet but
	// in a severe way that prevents the previous object from being restored
	// as "current".
	//
	// However, that situation was more common in some earlier Terraform
	// versions where deposed objects were not managed properly, so this
	// can arise when upgrading from an older version with deposed objects
	// already in the state.
	//
	// This is one of the few cases where we expose the idea of "deposed" in
	// the UI, including the user-unfriendly "deposed key" (00000000 in this
	// case) just so that users can correlate this with what they might
	// see in `terraform show` and in the subsequent apply output, because
	// it's also possible for there to be _multiple_ deposed objects, in the
	// unlikely event that create_before_destroy _keeps_ crashing across
	// subsequent runs.
	expectedOutput := `An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create
  - destroy

Terraform will perform the following actions:

  # test_instance.foo will be created
  + resource "test_instance" "foo" {
      + ami = "bar"

      + network_interface {
          + description  = "Main network interface"
          + device_index = 0
        }
    }

  # test_instance.foo (deposed object 00000000) will be destroyed
  - resource "test_instance" "foo" {
      - ami = "bar" -> null

      - network_interface {
          - description  = "Main network interface" -> null
          - device_index = 0 -> null
        }
    }

Plan: 1 to add, 0 to change, 1 to destroy.`
	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("Unexpected output:\n%s\n\nwant output containing:\n%s", output, expectedOutput)
	}
}

func TestLocal_planTainted_createBeforeDestroy(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState_tainted())
	b.CLI = cli.NewMockUi()
	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")
	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan-cbd")
	defer configCleanup()
	op.PlanRefresh = true
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		// Just a placeholder so that we can generate a valid plan file.
		Type:   "local",
		Config: cfgRaw,
	}
	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}
	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}
	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	expectedOutput := `An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
+/- create replacement and then destroy

Terraform will perform the following actions:

  # test_instance.foo is tainted, so must be replaced
+/- resource "test_instance" "foo" {
        ami = "bar"

        network_interface {
            description  = "Main network interface"
            device_index = 0
        }
    }

Plan: 1 to add, 0 to change, 1 to destroy.`
	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("Unexpected output:\n%s", output)
	}
}

func TestLocal_planRefreshFalse(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()

	p := TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState())

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	if p.ReadResourceCalled {
		t.Fatal("ReadResource should not be called")
	}

	if !run.PlanEmpty {
		t.Fatal("plan should be empty")
	}
}

func TestLocal_planDestroy(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()

	p := TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState())

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.Destroy = true
	op.PlanRefresh = true
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		// Just a placeholder so that we can generate a valid plan file.
		Type:   "local",
		Config: cfgRaw,
	}

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}

	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	plan := testReadPlan(t, planPath)
	for _, r := range plan.Changes.Resources {
		if r.Action.String() != "Delete" {
			t.Fatalf("bad: %#v", r.Action.String())
		}
	}
}

func TestLocal_planDestroy_withDataSources(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()

	p := TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState_withDataSource())

	b.CLI = cli.NewMockUi()

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op, configCleanup := testOperationPlan(t, "./test-fixtures/destroy-with-ds")
	defer configCleanup()
	op.Destroy = true
	op.PlanRefresh = true
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		// Just a placeholder so that we can generate a valid plan file.
		Type:   "local",
		Config: cfgRaw,
	}

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	if !p.ReadResourceCalled {
		t.Fatal("ReadResource should be called")
	}

	if !p.ReadDataSourceCalled {
		t.Fatal("ReadDataSourceCalled should be called")
	}

	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	// Data source should still exist in the the plan file
	plan := testReadPlan(t, planPath)
	if len(plan.Changes.Resources) != 2 {
		t.Fatalf("Expected exactly 1 resource for destruction, %d given: %q",
			len(plan.Changes.Resources), getAddrs(plan.Changes.Resources))
	}

	// Data source should not be rendered in the output
	expectedOutput := `Terraform will perform the following actions:

  # test_instance.foo will be destroyed
  - resource "test_instance" "foo" {
      - ami = "bar" -> null

      - network_interface {
          - description  = "Main network interface" -> null
          - device_index = 0 -> null
        }
    }

Plan: 0 to add, 0 to change, 1 to destroy.`

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, expectedOutput) {
		t.Fatalf("Unexpected output (expected no data source):\n%s", output)
	}
}

func getAddrs(resources []*plans.ResourceInstanceChangeSrc) []string {
	addrs := make([]string, len(resources), len(resources))
	for i, r := range resources {
		addrs[i] = r.Addr.String()
	}
	return addrs
}

func TestLocal_planOutPathNoChange(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState())

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.PlanOutPath = planPath
	cfg := cty.ObjectVal(map[string]cty.Value{
		"path": cty.StringVal(b.StatePath),
	})
	cfgRaw, err := plans.NewDynamicValue(cfg, cfg.Type())
	if err != nil {
		t.Fatal(err)
	}
	op.PlanOutBackend = &plans.Backend{
		// Just a placeholder so that we can generate a valid plan file.
		Type:   "local",
		Config: cfgRaw,
	}

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	plan := testReadPlan(t, planPath)

	if !plan.Changes.Empty() {
		t.Fatalf("expected empty plan to be written")
	}
}

// TestLocal_planScaleOutNoDupeCount tests a Refresh/Plan sequence when a
// resource count is scaled out. The scaled out node needs to exist in the
// graph and run through a plan-style sequence during the refresh phase, but
// can conflate the count if its post-diff count hooks are not skipped. This
// checks to make sure the correct resource count is ultimately given to the
// UI.
func TestLocal_planScaleOutNoDupeCount(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	TestLocalProvider(t, b, "test", planFixtureSchema())
	testStateFile(t, b.StatePath, testPlanState())

	actual := new(CountHook)
	b.ContextOpts.Hooks = append(b.ContextOpts.Hooks, actual)

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan-scaleout")
	defer configCleanup()
	op.PlanRefresh = true

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	expected := new(CountHook)
	expected.ToAdd = 1
	expected.ToChange = 0
	expected.ToRemoveAndAdd = 0
	expected.ToRemove = 0

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected %#v, got %#v instead.",
			expected, actual)
	}
}

func testOperationPlan(t *testing.T, configDir string) (*backend.Operation, func()) {
	t.Helper()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)

	return &backend.Operation{
		Type:         backend.OperationTypePlan,
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
	}, configCleanup
}

// testPlanState is just a common state that we use for testing plan.
func testPlanState() *states.State {
	state := states.NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status: states.ObjectReady,
			AttrsJSON: []byte(`{
				"ami": "bar",
				"network_interface": [{
					"device_index": 0,
					"description": "Main network interface"
				}]
			}`),
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)
	return state
}

func testPlanState_withDataSource() *states.State {
	state := states.NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status: states.ObjectReady,
			AttrsJSON: []byte(`{
				"ami": "bar",
				"network_interface": [{
					"device_index": 0,
					"description": "Main network interface"
				}]
			}`),
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "test_ds",
			Name: "bar",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status: states.ObjectReady,
			AttrsJSON: []byte(`{
				"filter": "foo"
			}`),
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)
	return state
}

func testPlanState_tainted() *states.State {
	state := states.NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status: states.ObjectTainted,
			AttrsJSON: []byte(`{
				"ami": "bar",
				"network_interface": [{
					"device_index": 0,
					"description": "Main network interface"
				}]
			}`),
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)
	return state
}

func testReadPlan(t *testing.T, path string) *plans.Plan {
	t.Helper()

	p, err := planfile.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer p.Close()

	plan, err := p.ReadPlan()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return plan
}

// planFixtureSchema returns a schema suitable for processing the
// configuration in test-fixtures/plan . This schema should be
// assigned to a mock provider named "test".
func planFixtureSchema() *terraform.ProviderSchema {
	return &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"network_interface": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"device_index": {Type: cty.Number, Optional: true},
								"description":  {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"test_ds": {
				Attributes: map[string]*configschema.Attribute{
					"filter": {Type: cty.String, Required: true},
				},
			},
		},
	}
}
