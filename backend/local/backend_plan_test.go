package local

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

func TestLocal_planBasic(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test")

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

	if !p.DiffCalled {
		t.Fatal("diff should be called")
	}
}

func TestLocal_planInAutomation(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	TestLocalProvider(t, b, "test")

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
	TestLocalProvider(t, b, "test")

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

func TestLocal_planRefreshFalse(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	op, configCleanup := testOperationPlan(t, "./test-fixtures/empty")
	defer configCleanup()

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	if p.RefreshCalled {
		t.Fatal("refresh should not be called")
	}

	if !run.PlanEmpty {
		t.Fatal("plan should be empty")
	}
}

func TestLocal_planDestroy(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	p := TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.Destroy = true
	op.PlanRefresh = true
	op.PlanOutPath = planPath

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	if !p.RefreshCalled {
		t.Fatal("refresh should be called")
	}

	if run.PlanEmpty {
		t.Fatal("plan should not be empty")
	}

	plan := testReadPlan(t, planPath)
	for _, m := range plan.Diff.Modules {
		for _, r := range m.Resources {
			if !r.Destroy {
				t.Fatalf("bad: %#v", r)
			}
		}
	}
}

func TestLocal_planOutPathNoChange(t *testing.T) {
	b, cleanup := TestLocal(t)
	defer cleanup()
	TestLocalProvider(t, b, "test")
	terraform.TestStateFile(t, b.StatePath, testPlanState())

	outDir := testTempDir(t)
	defer os.RemoveAll(outDir)
	planPath := filepath.Join(outDir, "plan.tfplan")

	op, configCleanup := testOperationPlan(t, "./test-fixtures/plan")
	defer configCleanup()
	op.PlanOutPath = planPath

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	<-run.Done()
	if run.Result != backend.OperationSuccess {
		t.Fatalf("plan operation failed")
	}

	plan := testReadPlan(t, planPath)
	if !plan.Diff.Empty() {
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
	TestLocalProvider(t, b, "test")
	state := &terraform.State{
		Version: 2,
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo.0": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
					"test_instance.foo.1": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
	terraform.TestStateFile(t, b.StatePath, state)

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

	_, configLoader, configCleanup := configload.MustLoadConfigForTests(t, configDir)

	return &backend.Operation{
		Type:         backend.OperationTypePlan,
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
	}, configCleanup
}

// testPlanState is just a common state that we use for testing refresh.
func testPlanState() *terraform.State {
	return &terraform.State{
		Version: 2,
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
			},
		},
	}
}

func testReadPlan(t *testing.T, path string) *terraform.Plan {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	p, err := terraform.ReadPlan(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return p
}
