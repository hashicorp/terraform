// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// Tests using `terraform workspace` commands in combination with pluggable state storage.
func TestPrimary_stateStore_workspaceCmd(t *testing.T) {
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
	fsMirrorPath := "cache/registry.terraform.io/hashicorp/simple6/0.0.1/"
	if err := os.MkdirAll(tf.Path(fsMirrorPath, platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(fsMirrorPath, platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	//// Init
	_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	fi, err := os.Stat(path.Join(tf.WorkDir(), workspaceDirName, "default", "terraform.tfstate"))
	if err != nil {
		t.Fatalf("failed to open default workspace's state file: %s", err)
	}
	if fi.Size() == 0 {
		t.Fatal("default workspace's state file should not have size 0 bytes")
	}

	//// Create Workspace: terraform workspace new
	newWorkspace := "foobar"
	stdout, stderr, err := tf.Run("workspace", "new", newWorkspace, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg := fmt.Sprintf("Created and switched to workspace %q!", newWorkspace)
	if !strings.Contains(stdout, expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}
	fi, err = os.Stat(path.Join(tf.WorkDir(), workspaceDirName, newWorkspace, "terraform.tfstate"))
	if err != nil {
		t.Fatalf("failed to open %s workspace's state file: %s", newWorkspace, err)
	}
	if fi.Size() == 0 {
		t.Fatalf("%s workspace's state file should not have size 0 bytes", newWorkspace)
	}

	//// List Workspaces: : terraform workspace list
	stdout, stderr, err = tf.Run("workspace", "list", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	if !strings.Contains(stdout, newWorkspace) {
		t.Errorf("unexpected output, expected the new %q workspace to be listed present, but it's missing. Got:\n%s", newWorkspace, stdout)
	}

	//// Select Workspace: terraform workspace select
	selectedWorkspace := "default"
	stdout, stderr, err = tf.Run("workspace", "select", selectedWorkspace, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = fmt.Sprintf("Switched to workspace %q.", selectedWorkspace)
	if !strings.Contains(stdout, expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}

	//// Show Workspace: terraform workspace show
	stdout, stderr, err = tf.Run("workspace", "show", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = fmt.Sprintf("%s\n", selectedWorkspace)
	if stdout != expectedMsg {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}

	//// Delete Workspace: terraform workspace delete
	stdout, stderr, err = tf.Run("workspace", "delete", newWorkspace, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = fmt.Sprintf("Deleted workspace %q!\n", newWorkspace)
	if stdout != expectedMsg {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}
}

// Tests using `terraform state` subcommands in combination with pluggable state storage:
// > `terraform state show`
// > `terraform state list`
func TestPrimary_stateStore_stateCmds(t *testing.T) {

	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	t.Setenv(e2e.TestExperimentFlag, "true")
	tfBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")

	fixturePath := filepath.Join("testdata", "initialized-directory-with-state-store-fs")
	tf := e2e.NewBinary(t, tfBin, fixturePath)

	workspaceDirName := "states" // see test fixture value for workspace_dir

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into the correct .terraform/providers/ directory
	// that will contain provider binaries in an initialized working directory.
	platform := getproviders.CurrentPlatform.String()
	providerCachePath := ".terraform/providers/registry.terraform.io/hashicorp/simple6/0.0.1/"
	if err := os.MkdirAll(tf.Path(providerCachePath, platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(providerCachePath, platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	// Assert that the test starts with the default state present from test fixtures
	defaultStateId := "default"
	fi, err := os.Stat(path.Join(tf.WorkDir(), workspaceDirName, defaultStateId, "terraform.tfstate"))
	if err != nil {
		t.Fatalf("failed to open default workspace's state file: %s", err)
	}
	if fi.Size() == 0 {
		t.Fatal("default workspace's state file should not have size 0 bytes")
	}

	//// List State: terraform state list
	expectedResourceAddr := "terraform_data.my-data"
	stdout, stderr, err := tf.Run("state", "list", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg := expectedResourceAddr + "\n" // This is the only resource instance in the test fixture state
	if stdout != expectedMsg {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}

	//// Show State: terraform state show
	stdout, stderr, err = tf.Run("state", "show", expectedResourceAddr, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	// show displays the state for the specified resource
	expectedMsg = `# terraform_data.my-data:
resource "terraform_data" "my-data" {
    id     = "d71fb368-2ba1-fb4c-5bd9-6a2b7f05d60c"
    input  = "hello world"
    output = "hello world"
}
`
	if diff := cmp.Diff(stdout, expectedMsg); diff != "" {
		t.Errorf("wrong result, diff:\n%s", diff)
	}
}

// Tests using the `terraform provider` subcommands in combination with pluggable state storage:
// > `terraform providers`
// > `terraform providers schema`
//
// Commands `terraform providers locks` and `terraform providers mirror` aren't tested as they
// don't interact with the backend.
func TestPrimary_stateStore_providerCmds(t *testing.T) {
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

	//// Init
	_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	fi, err := os.Stat(path.Join(tf.WorkDir(), workspaceDirName, "default", "terraform.tfstate"))
	if err != nil {
		t.Fatalf("failed to open default workspace's state file: %s", err)
	}
	if fi.Size() == 0 {
		t.Fatal("default workspace's state file should not have size 0 bytes")
	}

	//// Providers: `terraform providers`
	stdout, stderr, err := tf.Run("providers", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	expectedMsgs := []string{
		"├── provider[registry.terraform.io/hashicorp/simple6]",
		"└── provider[terraform.io/builtin/terraform]",
	}
	for _, msg := range expectedMsgs {
		if !strings.Contains(stdout, msg) {
			t.Errorf("unexpected output, expected %q, but got:\n%s", msg, stdout)
		}
	}

	//// Provider schemas: `terraform providers schema`
	stdout, stderr, err = tf.Run("providers", "schema", "-json", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	expectedMsgs = []string{
		`{"format_version":"1.0","provider_schemas":{`, // opening of JSON
		`"registry.terraform.io/hashicorp/simple6":{`,  // provider 1
		`"terraform.io/builtin/terraform":{`,           // provider 2
	}
	for _, msg := range expectedMsgs {
		if !strings.Contains(stdout, msg) {
			t.Errorf("unexpected output, expected %q, but got:\n%s", msg, stdout)
		}
	}
}
