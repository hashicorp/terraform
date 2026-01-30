// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statefile"
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
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

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
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

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

	gotOutput := state.RootOutputValues["cwd"]
	wantOutputValue := cty.StringVal(filepath.ToSlash(tf.Path())) // path.cwd returns the original path, because path.root is how we get the overridden path
	if gotOutput == nil || !wantOutputValue.RawEquals(gotOutput.Value) {
		t.Errorf("incorrect value for cwd output\ngot: %#v\nwant Value: %#v", gotOutput, wantOutputValue)
	}

	gotOutput = state.RootOutputValues["root"]
	wantOutputValue = cty.StringVal(filepath.ToSlash(tf.Path("subdir"))) // path.root is a relative path, but the text fixture uses abspath on it.
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

func TestPrimary_stateStore(t *testing.T) {

	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	t.Setenv(e2e.TestExperimentFlag, "true")
	terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")

	fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)
	workspaceDirName := "states" // See workspace_dir value in the configuration

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	//// INIT
	stdout, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Terraform created an empty state file for the default workspace") {
		t.Errorf("notice about creating the default workspace is missing from init output:\n%s", stdout)
	}

	//// PLAN
	// No separate plan step; this test lets the apply make a plan.

	//// APPLY
	stdout, stderr, err = tf.Run("apply", "-auto-approve", "-no-color")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 1 added, 0 changed, 0 destroyed") {
		t.Errorf("incorrect apply tally; want 1 added:\n%s", stdout)
	}

	// Check the statefile saved by the fs state store.
	path := fmt.Sprintf("%s/default/terraform.tfstate", workspaceDirName)
	f, err := tf.OpenFile(path)
	if err != nil {
		t.Fatalf("unexpected error opening state file %s: %s\nstderr:\n%s", path, err, stderr)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("unexpected error reading statefile %s: %s\nstderr:\n%s", path, err, stderr)
	}

	r := stateFile.State.RootModule().Resources
	if len(r) != 1 {
		t.Fatalf("expected state to include one resource, but got %d", len(r))
	}
	if _, ok := r["terraform_data.my-data"]; !ok {
		t.Fatalf("expected state to include terraform_data.my-data but it's missing")
	}
}

func TestPrimary_stateStore_planFile(t *testing.T) {

	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	t.Setenv(e2e.TestExperimentFlag, "true")
	terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")

	fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	//// INIT
	stdout, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Terraform created an empty state file for the default workspace") {
		t.Errorf("notice about creating the default workspace is missing from init output:\n%s", stdout)
	}

	//// PLAN
	planFile := "testplan"
	_, stderr, err = tf.Run("plan", "-out="+planFile, "-no-color")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	//// APPLY
	stdout, stderr, err = tf.Run("apply", "-auto-approve", "-no-color", planFile)
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 1 added, 0 changed, 0 destroyed") {
		t.Errorf("incorrect apply tally; want 1 added:\n%s", stdout)
	}

	// Check the statefile saved by the fs state store.
	path := "states/default/terraform.tfstate"
	f, err := tf.OpenFile(path)
	if err != nil {
		t.Fatalf("unexpected error opening state file %s: %s\nstderr:\n%s", path, err, stderr)
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("unexpected error reading statefile %s: %s\nstderr:\n%s", path, err, stderr)
	}

	r := stateFile.State.RootModule().Resources
	if len(r) != 1 {
		t.Fatalf("expected state to include one resource, but got %d", len(r))
	}
	if _, ok := r["terraform_data.my-data"]; !ok {
		t.Fatalf("expected state to include terraform_data.my-data but it's missing")
	}
}

func TestPrimary_stateStore_inMem(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	t.Setenv(e2e.TestExperimentFlag, "true")
	terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")

	fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-inmem")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	//// INIT
	//
	// Note - the inmem PSS implementation means that the default workspace state created during init
	// is lost as soon as the command completes.
	stdout, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Terraform created an empty state file for the default workspace") {
		t.Errorf("notice about creating the default workspace is missing from init output:\n%s", stdout)
	}

	//// PLAN
	// No separate plan step; this test lets the apply make a plan.

	//// APPLY
	//
	// Note - the inmem PSS implementation means that writing to the default workspace during apply
	// is creating the default state file for the first time.
	stdout, stderr, err = tf.Run("apply", "-auto-approve", "-no-color")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 1 added, 0 changed, 0 destroyed") {
		t.Errorf("incorrect apply tally; want 1 added:\n%s", stdout)
	}

	// We cannot inspect state or perform a destroy here, as the state isn't persisted between steps
	// when we use the simple6_inmem state store.
}
