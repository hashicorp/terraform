package e2etest

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/e2e"
)

// The tests in this file run through different scenarios recommended in our
// "Running Terraform in Automation" guide:
//     https://www.terraform.io/guides/running-terraform-in-automation.html

// TestPlanApplyInAutomation runs through the "main case" of init, plan, apply
// using the specific command line options suggested in the guide.
func TestPlanApplyInAutomation(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "full-workflow-null")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	// We advertise that _any_ non-empty value works, so we'll test something
	// unconventional here.
	tf.AddEnv("TF_IN_AUTOMATION=yes-please")

	//// INIT
	stdout, stderr, err := tf.Run("init", "-input=false")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	// Make sure we actually downloaded the plugins, rather than picking up
	// copies that might be already installed globally on the system.
	if !strings.Contains(stdout, "- Downloading plugin for provider \"template") {
		t.Errorf("template provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}
	if !strings.Contains(stdout, "- Downloading plugin for provider \"null") {
		t.Errorf("null provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	//// PLAN
	stdout, stderr, err = tf.Run("plan", "-out=tfplan", "-input=false")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "1 to add, 0 to change, 0 to destroy") {
		t.Errorf("incorrect plan tally; want 1 to add:\n%s", stdout)
	}

	// Because we're running with TF_IN_AUTOMATION set, we should not see
	// any mention of the plan file in the output.
	if strings.Contains(stdout, "tfplan") {
		t.Errorf("unwanted mention of \"tfplan\" file in plan output\n%s", stdout)
	}

	plan, err := tf.Plan("tfplan")
	if err != nil {
		t.Fatalf("failed to read plan file: %s", err)
	}

	// stateResources := plan.Changes.Resources
	diffResources := plan.Changes.Resources

	if len(diffResources) != 1 || diffResources[0].Addr.String() != "null_resource.test" {
		t.Errorf("incorrect number of resources in plan")
	}

	//// APPLY
	stdout, stderr, err = tf.Run("apply", "-input=false", "tfplan")
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
	for n, _ := range stateResources {
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
}

// TestAutoApplyInAutomation tests the scenario where the caller skips creating
// an explicit plan and instead forces automatic application of changes.
func TestAutoApplyInAutomation(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "full-workflow-null")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	// We advertise that _any_ non-empty value works, so we'll test something
	// unconventional here.
	tf.AddEnv("TF_IN_AUTOMATION=very-much-so")

	//// INIT
	stdout, stderr, err := tf.Run("init", "-input=false")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	// Make sure we actually downloaded the plugins, rather than picking up
	// copies that might be already installed globally on the system.
	if !strings.Contains(stdout, "- Downloading plugin for provider \"template") {
		t.Errorf("template provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}
	if !strings.Contains(stdout, "- Downloading plugin for provider \"null") {
		t.Errorf("null provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	//// APPLY
	stdout, stderr, err = tf.Run("apply", "-input=false", "-auto-approve")
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
}

// TestPlanOnlyInAutomation tests the scenario of creating a "throwaway" plan,
// which we recommend as a way to verify a pull request.
func TestPlanOnlyInAutomation(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "full-workflow-null")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	// We advertise that _any_ non-empty value works, so we'll test something
	// unconventional here.
	tf.AddEnv("TF_IN_AUTOMATION=verily")

	//// INIT
	stdout, stderr, err := tf.Run("init", "-input=false")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	// Make sure we actually downloaded the plugins, rather than picking up
	// copies that might be already installed globally on the system.
	if !strings.Contains(stdout, "- Downloading plugin for provider \"template") {
		t.Errorf("template provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}
	if !strings.Contains(stdout, "- Downloading plugin for provider \"null") {
		t.Errorf("null provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	//// PLAN
	stdout, stderr, err = tf.Run("plan", "-input=false")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "1 to add, 0 to change, 0 to destroy") {
		t.Errorf("incorrect plan tally; want 1 to add:\n%s", stdout)
	}

	// Because we're running with TF_IN_AUTOMATION set, we should not see
	// any mention of the the "terraform apply" command in the output.
	if strings.Contains(stdout, "terraform apply") {
		t.Errorf("unwanted mention of \"terraform apply\" in plan output\n%s", stdout)
	}

	if tf.FileExists("tfplan") {
		t.Error("plan file was created, but was not expected")
	}
}
