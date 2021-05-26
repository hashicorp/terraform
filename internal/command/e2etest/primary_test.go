package e2etest

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
)

// The tests in this file are for the "primary workflow", which includes
// variants of the following sequence, with different details:
// terraform init
// terraform plan
// terraform apply
// terraform destroy

func TestPrimarySeparatePlan(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "full-workflow-null")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	//// INIT
	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	// Make sure we actually downloaded the plugins, rather than picking up
	// copies that might be already installed globally on the system.
	if !strings.Contains(stdout, "Installing hashicorp/template v") {
		t.Errorf("template provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}
	if !strings.Contains(stdout, "Installing hashicorp/null v") {
		t.Errorf("null provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	//// PLAN
	stdout, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "1 to add, 0 to change, 0 to destroy") {
		t.Errorf("incorrect plan tally; want 1 to add:\n%s", stdout)
	}

	if !strings.Contains(stdout, "Saved the plan to: tfplan") {
		t.Errorf("missing \"Saved the plan to...\" message in plan output\n%s", stdout)
	}
	if !strings.Contains(stdout, "terraform apply \"tfplan\"") {
		t.Errorf("missing next-step instruction in plan output\n%s", stdout)
	}

	plan, err := tf.Plan("tfplan")
	if err != nil {
		t.Fatalf("failed to read plan file: %s", err)
	}

	diffResources := plan.Changes.Resources
	if len(diffResources) != 1 {
		t.Errorf("incorrect number of resources in plan")
	}

	expected := map[string]plans.Action{
		"null_resource.test": plans.Create,
	}

	for _, r := range diffResources {
		expectedAction, ok := expected[r.Addr.String()]
		if !ok {
			t.Fatalf("unexpected change for %q", r.Addr)
		}
		if r.Action != expectedAction {
			t.Fatalf("unexpected action %q for %q", r.Action, r.Addr)
		}
	}

	//// APPLY
	stdout, stderr, err = tf.Run("apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 1 added, 0 changed, 0 destroyed") {
		t.Errorf("incorrect apply tally; want 1 added:\n%s", stdout)
	}

	state, err := tf.LocalState()
	if err != nil {
		t.Fatalf("failed to read state file: %s", err)
	}

	stateResources := state.RootModule().Resources
	var gotResources []string
	for n := range stateResources {
		gotResources = append(gotResources, n)
	}
	sort.Strings(gotResources)

	wantResources := []string{
		"data.template_file.test",
		"null_resource.test",
	}

	if !reflect.DeepEqual(gotResources, wantResources) {
		t.Errorf("wrong resources in state\ngot: %#v\nwant: %#v", gotResources, wantResources)
	}

	//// DESTROY
	stdout, stderr, err = tf.Run("destroy", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected destroy error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 1 destroyed") {
		t.Errorf("incorrect destroy tally; want 1 destroyed:\n%s", stdout)
	}

	state, err = tf.LocalState()
	if err != nil {
		t.Fatalf("failed to read state file after destroy: %s", err)
	}

	stateResources = state.RootModule().Resources
	if len(stateResources) != 0 {
		t.Errorf("wrong resources in state after destroy; want none, but still have:%s", spew.Sdump(stateResources))
	}

}

func TestPrimaryChdirOption(t *testing.T) {
	t.Parallel()

	// This test case does not include any provider dependencies, so it's
	// safe to run it even when network access is disallowed.

	fixturePath := filepath.Join("testdata", "chdir-option")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	//// INIT
	_, stderr, err := tf.Run("-chdir=subdir", "init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	stdout, stderr, err := tf.Run("-chdir=subdir", "plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	if want := "You can apply this plan to save these new output values"; !strings.Contains(stdout, want) {
		t.Errorf("missing expected message for an outputs-only plan\ngot:\n%s\n\nwant substring: %s", stdout, want)
	}

	if !strings.Contains(stdout, "Saved the plan to: tfplan") {
		t.Errorf("missing \"Saved the plan to...\" message in plan output\n%s", stdout)
	}
	if !strings.Contains(stdout, "terraform apply \"tfplan\"") {
		t.Errorf("missing next-step instruction in plan output\n%s", stdout)
	}

	// The saved plan is in the subdirectory because -chdir switched there
	plan, err := tf.Plan("subdir/tfplan")
	if err != nil {
		t.Fatalf("failed to read plan file: %s", err)
	}

	diffResources := plan.Changes.Resources
	if len(diffResources) != 0 {
		t.Errorf("incorrect diff in plan; want no resource changes, but have:\n%s", spew.Sdump(diffResources))
	}

	//// APPLY
	stdout, stderr, err = tf.Run("-chdir=subdir", "apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 0 added, 0 changed, 0 destroyed") {
		t.Errorf("incorrect apply tally; want 0 added:\n%s", stdout)
	}

	// The state file is in subdir because -chdir changed the current working directory.
	state, err := tf.StateFromFile("subdir/terraform.tfstate")
	if err != nil {
		t.Fatalf("failed to read state file: %s", err)
	}

	gotOutput := state.RootModule().OutputValues["cwd"]
	wantOutputValue := cty.StringVal(tf.Path()) // path.cwd returns the original path, because path.root is how we get the overridden path
	if gotOutput == nil || !wantOutputValue.RawEquals(gotOutput.Value) {
		t.Errorf("incorrect value for cwd output\ngot: %#v\nwant Value: %#v", gotOutput, wantOutputValue)
	}

	gotOutput = state.RootModule().OutputValues["root"]
	wantOutputValue = cty.StringVal(tf.Path("subdir")) // path.root is a relative path, but the text fixture uses abspath on it.
	if gotOutput == nil || !wantOutputValue.RawEquals(gotOutput.Value) {
		t.Errorf("incorrect value for root output\ngot: %#v\nwant Value: %#v", gotOutput, wantOutputValue)
	}

	if len(state.RootModule().Resources) != 0 {
		t.Errorf("unexpected resources in state")
	}

	//// DESTROY
	stdout, stderr, err = tf.Run("-chdir=subdir", "destroy", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected destroy error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 0 destroyed") {
		t.Errorf("incorrect destroy tally; want 0 destroyed:\n%s", stdout)
	}
}
