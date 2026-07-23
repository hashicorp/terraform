// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

// Test that users can do the full init-plan-apply workflow with pluggable state storage
// when the state storage provider is unmanaged by Terraform.
// As well as ensuring that the state store can be initialised ok, this tests that
// the state store's details can be stored in the plan file despite the fact it's unmanaged.
func TestPrimary_stateStore_unmanaged_separatePlan(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)

	reattachStr, provider := reattachedProviderForTest(t, addrs.NewDefaultProvider("simple6"), 6)
	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))

	// Required for the local state files to be written to the temp directory,
	// instead of the e2e directory in the repo.
	t.Chdir(tf.WorkDir())

	//// INIT
	t.Setenv("TF_ENABLE_PLUGGABLE_STATE_STORAGE", "1")
	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s\nstdout:\n%s", err, stderr, stdout)
	}
	if !provider.ReadStateBytesCalled() {
		t.Error("ReadStateBytes not called on un-managed provider")
	}
	provider.ResetReadStateBytesCalled()
	provider.ResetWriteStateBytesCalled()

	// Make sure we didn't download the binary
	if strings.Contains(stdout, "Installing hashicorp/simple6 v") {
		t.Errorf("test provider download message is present in init output:\n%s", stdout)
	}
	if tf.FileExists(filepath.Join(".terraform", "plugins", "registry.terraform.io", "hashicorp", "simple6")) {
		t.Errorf("test provider binary found in .terraform dir")
	}

	//// PLAN
	stdout, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s\nstdout:\n%s", err, stderr, stdout)
	}
	if !provider.ReadStateBytesCalled() {
		t.Error("ReadStateBytes not called on un-managed provider")
	}
	if provider.WriteStateBytesCalled() {
		t.Error("WriteStateBytes should not be called on un-managed provider during plan")
	}
	provider.ResetReadStateBytesCalled()
	provider.ResetWriteStateBytesCalled()

	//// APPLY
	stdout, stderr, err = tf.Run("apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s\nstdout:\n%s", err, stderr, stdout)
	}
	if !provider.ReadStateBytesCalled() {
		t.Error("ReadStateBytes not called on un-managed provider")
	}
	if !provider.WriteStateBytesCalled() {
		t.Error("WriteStateBytes not called on un-managed provider")
	}
	provider.ResetReadStateBytesCalled()
	provider.ResetWriteStateBytesCalled()

	// Check the apply process has made a state file as expected.
	stateFilePath := filepath.Join("states", "default", "terraform.tfstate")
	if !tf.FileExists(stateFilePath) {
		t.Fatalf("state file not found at expected path: %s", filepath.Join(tf.WorkDir(), stateFilePath))
	}

	//// DESTROY
	stdout, stderr, err = tf.Run("destroy", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected destroy error: %s\nstderr:\n%s\nstdout:\n%s", err, stderr, stdout)
	}
}

// Tests using `terraform workspace` commands in combination with pluggable state storage.
func TestPrimary_stateStore_workspaceCmd(t *testing.T) {
	t.Parallel()
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")
	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)
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
	_, err = os.Stat(path.Join(tf.WorkDir(), workspaceDirName, "default", "terraform.tfstate"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expected default workspace's state file to not exist, but it exists")
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
	fi, err := os.Stat(path.Join(tf.WorkDir(), workspaceDirName, newWorkspace, "terraform.tfstate"))
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
	stdout, stderr, err = tf.Run("workspace", "select", "-or-create", selectedWorkspace, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = fmt.Sprintf("Created and switched to workspace %q!", selectedWorkspace)
	if !strings.Contains(stdout, expectedMsg) {
		t.Errorf("unexpected output, expected %s, but got:\n%s", expectedMsg, stdout)
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
	t.Parallel()
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "initialized-directory-with-state-store-fs")
	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)

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

// Test using `terraform state migrate` subcommand
func TestPrimary_stateStore_stateMigrateCmd_upgrade(t *testing.T) {
	t.Parallel()
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "state-migrate-upgrade")

	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)

	// setup FS mirror
	tmpDir := t.TempDir()
	mirrorPath := filepath.Join(tmpDir, "mirror")
	cliCfgFilePath := filepath.Join(tmpDir, "test.tfrc")
	cfgBody := fmt.Sprintf(`provider_installation {
  filesystem_mirror {
    path    = %q
    include = ["registry.terraform.io/hashicorp/simple6"]
  }
  direct {
    exclude = ["registry.terraform.io/hashicorp/simple6"]
  }
}
`, mirrorPath)
	os.WriteFile(cliCfgFilePath, []byte(cfgBody), 0o700)
	tf.AddEnv("TF_CLI_CONFIG_FILE=" + cliCfgFilePath)

	// In order to test integration with PSS we need two provider plugins implementing a state store
	// which we can tell apart to be able to verify successful upgrade between them.
	platform := getproviders.CurrentPlatform.String()
	// Build v1.0.0 plugin
	simpleProviderv1 := filepath.Join(t.TempDir(), "terraform-provider-simple6")
	simpleProviderv1Exe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main",
		simpleProviderv1, "-ldflags", "-X 'main.fsStatesDir=v1.tfstate.d'")
	providerv1MirrorPath := filepath.Join(mirrorPath, "registry.terraform.io", "hashicorp", "simple6", "1.0.0")
	if err := os.MkdirAll(filepath.Join(providerv1MirrorPath, platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simpleProviderv1Exe, filepath.Join(providerv1MirrorPath, platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}
	// Build v2.0.0 plugin
	simpleProviderv2 := filepath.Join(t.TempDir(), "terraform-provider-simple6")
	simpleProviderv2Exe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main",
		simpleProviderv2, "-ldflags", "-X 'main.fsStatesDir=v2.tfstate.d'")
	providerv2MirrorPath := filepath.Join(mirrorPath, "registry.terraform.io", "hashicorp", "simple6", "2.0.0")
	if err := os.MkdirAll(filepath.Join(providerv2MirrorPath, platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simpleProviderv2Exe, filepath.Join(providerv2MirrorPath, platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := tf.Run("state", "migrate", "-upgrade", "-input=false", "-force-copy", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%q", err, stderr)
	}

	expectedMsg := []string{
		`Initializing provider hashicorp/simple6 (1.0.0) for state store "simple6_fs"...
- Reusing version 1.0.0 of hashicorp/simple6 from the dependency lock file
- Installing hashicorp/simple6 v1.0.0...`, // version from lockfile or tfmigrate
		`Initializing provider hashicorp/simple6 (2.0.0) for state store "simple6_fs"...
- Finding hashicorp/simple6 versions matching "2.0.0"...
- Installing hashicorp/simple6 v2.0.0...`, // version from tf config ONLY
		// read state from 1.0.0, retain in memory
		// write state to 2.0.0

		// TODO: add provider versions to the output
		// `Migrating state from state store "simple6_fs" (hashicorp/simple6 1.0.0) to state store "simple6_fs" (hashicorp/simple6 2.0.0)...`,
		// `Finished migrating state from state store "simple6_fs" (hashicorp/simple6 1.0.0) to state store "simple6_fs" (hashicorp/simple6 2.0.0).`,
		// `Finished upgrade of hashicorp/simple6 from 1.0.0 to 2.0.0`, // updated lockfile
	}
	for _, expectedMsg := range expectedMsg {
		if !strings.Contains(stdout, expectedMsg) {
			t.Fatalf("expected output %q, got %q", expectedMsg, stdout)
		}
	}

	// verify state exists in the new location
	newStatePath := filepath.Join(tf.WorkDir(), "v2.tfstate.d", "default", "terraform.tfstate")
	f, err := os.Open(newStatePath)
	t.Cleanup(func() { f.Close() })
	if err != nil {
		t.Fatal(err)
	}
	// b, err := io.ReadAll(f)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// TODO: verify contents
	// TODO: verify lockfile was updated
}

// Tests using the `terraform output` command in combination with pluggable state storage:
// > `terraform output`
// > `terraform output <name>`
func TestPrimary_stateStore_outputCmd(t *testing.T) {
	t.Parallel()
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "initialized-directory-with-state-store-fs")
	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)

	workspaceDirName := "states" // see test fixture value for workspace_dir

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into the correct .terraform/providers/ directory
	// that will contain provider binaries in an initialized working directory.
	platform := getproviders.CurrentPlatform.String()
	if err := os.MkdirAll(tf.Path(".terraform/providers/registry.terraform.io/hashicorp/simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(".terraform/providers/registry.terraform.io/hashicorp/simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
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

	//// List all outputs: terraform output
	stdout, stderr, err := tf.Run("output", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg := "greeting = \"hello world\"\n" // See the test fixture files
	if stdout != expectedMsg {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}

	//// View a specific output: terraform output <name>
	outputName := "greeting"
	stdout, stderr, err = tf.Run("output", outputName, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = "\"hello world\"\n" // Only the value is outputted, no name present
	if stdout != expectedMsg {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}
}

// Tests using the `terraform show` command in combination with pluggable state storage
// > `terraform show`
// > `terraform show <path-to-state-file>`
// > `terraform show <path-to-plan-file>`
func TestPrimary_stateStore_showCmd(t *testing.T) {
	t.Parallel()
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "initialized-directory-with-state-store-fs")
	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)

	workspaceDirName := "states" // see test fixture value for workspace_dir

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into the correct .terraform/providers/ directory
	// that will contain provider binaries in an initialized working directory.
	platform := getproviders.CurrentPlatform.String()
	if err := os.MkdirAll(tf.Path(".terraform/providers/registry.terraform.io/hashicorp/simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(".terraform/providers/registry.terraform.io/hashicorp/simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
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

	//// Show state: terraform state
	stdout, stderr, err := tf.Run("show", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg := `# terraform_data.my-data:
resource "terraform_data" "my-data" {
    id     = "d71fb368-2ba1-fb4c-5bd9-6a2b7f05d60c"
    input  = "hello world"
    output = "hello world"
}


Outputs:

greeting = "hello world"
` // See the test fixture folder's state file

	if diff := cmp.Diff(stdout, expectedMsg); diff != "" {
		t.Errorf("wrong result, diff:\n%s", diff)
	}

	//// Show state: terraform show <path to state file>
	path := fmt.Sprintf("./%s/%s/terraform.tfstate", workspaceDirName, defaultStateId)
	stdout, stderr, err = tf.Run("show", path, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	if diff := cmp.Diff(stdout, expectedMsg); diff != "" {
		t.Errorf("wrong result, diff:\n%s", diff)
	}

	//// Show state: terraform show <path to plan file>

	// 1. Create a plan file via plan command
	newOutput := `output "replacement" {
  value = resource.terraform_data.my-data.output
}`
	if err := os.WriteFile(filepath.Join(tf.WorkDir(), "outputs.tf"), []byte(newOutput), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	planFile := "tfplan"
	stdout, stderr, err = tf.Run("plan", fmt.Sprintf("-out=%s", planFile), "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = "Changes to Outputs"
	if !strings.Contains(stdout, expectedMsg) {
		t.Errorf("wrong result, expected the plan command to create a plan file but that hasn't happened, got:\n%s",
			stdout,
		)
	}

	// 2. Inspect plan file
	stdout, stderr, err = tf.Run("show", planFile, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	expectedMsg = `
Changes to Outputs:
  - greeting    = "hello world" -> null
  + replacement = "hello world"

You can apply this plan to save these new output values to the Terraform
state, without changing any real infrastructure.
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
//
// The test `TestProvidersSchema` has test coverage showing that state store schemas are present
// in the command's outputs. _This_ test is intended to assert that the command is able to read and use
// state via a state store ok, and is able to detect providers required only by the state.
func TestPrimary_stateStore_providerCmds(t *testing.T) {
	t.Parallel()
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")
	tf := e2e.NewBinary(t, experimentalTerraformBin, fixturePath)
	workspaceDirName := "states" // See workspace_dir value in the configuration

	// Add a state file describing a resource from the simple (v5) provider, so
	// we can test that the state is read and used to get all the provider schemas
	fakeState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "simple_resource",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("simple"),
				Module:   addrs.RootModule,
			},
		)
	})
	fakeStateFile := &statefile.File{
		Lineage:          "boop",
		Serial:           4,
		TerraformVersion: version.Must(version.NewVersion("1.0.0")),
		State:            fakeState,
	}
	var fakeStateBuf bytes.Buffer
	err := statefile.WriteForTest(fakeStateFile, &fakeStateBuf)
	if err != nil {
		t.Error(err)
	}
	fakeStateBytes := fakeStateBuf.Bytes()

	if err := os.MkdirAll(tf.Path(workspaceDirName, "default"), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tf.Path(workspaceDirName, "default", "terraform.tfstate"), fakeStateBytes, 0644); err != nil {
		t.Fatal(err)
	}

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which will be used for PSS.
	// The simple (v5) provider is also built, as that provider will be present in the state and therefore
	// needed for creating the schema output.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	simpleProvider := filepath.Join(tf.WorkDir(), "terraform-provider-simple")
	simpleProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", simpleProvider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	fsMirrorPathV6 := "cache/registry.terraform.io/hashicorp/simple6/0.0.1/"
	if err := os.MkdirAll(tf.Path(fsMirrorPathV6, platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(fsMirrorPathV6, platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	fsMirrorPathV5 := "cache/registry.terraform.io/hashicorp/simple/0.0.1/"
	if err := os.MkdirAll(tf.Path(fsMirrorPathV5, platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simpleProviderExe, tf.Path(fsMirrorPathV5, platform, "terraform-provider-simple")); err != nil {
		t.Fatal(err)
	}

	//// Init
	_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	// Note: The default state was already created earlier in the test

	//// Providers: `terraform providers`
	stdout, stderr, err := tf.Run("providers", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	// We expect the command to be able to use the state store to
	// detect providers that come from only the state.
	expectedMsgs := []string{
		"Providers required by configuration:",
		"provider[registry.terraform.io/hashicorp/simple6]",
		"provider[terraform.io/builtin/terraform]",
		"Providers required by state:",
		"provider[registry.terraform.io/hashicorp/simple]",
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
		`"registry.terraform.io/hashicorp/simple6"`, // provider used for PSS
		`"terraform.io/builtin/terraform"`,          // provider used for resources
		`"registry.terraform.io/hashicorp/simple"`,  // provider present only in the state
	}
	for _, msg := range expectedMsgs {
		if !strings.Contains(stdout, msg) {
			t.Errorf("unexpected output, expected %q, but got:\n%s", msg, stdout)
		}
	}

	// More thorough checking of the JSON output is in `TestProvidersSchema`.
	// This test just asserts that `terraform providers schema` can read state
	// via the state store, and therefore detects all 3 providers needed for the output.
}
