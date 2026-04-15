// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/command"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	"github.com/hashicorp/terraform/internal/plans"
	tfplugin "github.com/hashicorp/terraform/internal/plugin6"
	simple "github.com/hashicorp/terraform/internal/provider-simple-v6"
	"github.com/hashicorp/terraform/internal/states/statefile"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
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
	_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	// No separate plan step; this test lets the apply make a plan.

	//// APPLY
	stdout, stderr, err := tf.Run("apply", "-auto-approve", "-no-color")
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
	_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	planFile := "testplan"
	_, stderr, err = tf.Run("plan", "-out="+planFile, "-no-color")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	//// APPLY
	stdout, stderr, err := tf.Run("apply", "-auto-approve", "-no-color", planFile)
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

// Characterize what happens when the state store is supplied through different methods between init and plan/apply.
// Outcomes are influenced by whether the init command produces a lock file entry for the PSS provider or not. Only managed
// providers are recorded in the dependency lock file.
func TestPrimary_stateStore_swapProviderSupplyMode_betweenInitAndPlanApply(t *testing.T) {
	// Swapping between different 'unmanaged' provider supply modes doesn't trigger a prompt to migrate state because
	// that change doesn't impact the hash of the state store. The hash is impacted by the Version data, and all unmanaged
	// providers used for PSS will have null version data.
	//
	// In contrast, swapping between a managed provider and any of reattached/dev_override/builtin WILL trigger a hash mismatch
	// because the version data will change.
	t.Run("users are NOT prompted to migrate state if an unmanaged provider used for PSS provider swaps supply mode (e.g. swap from reattached to dev_override) between init and plan+apply", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

		t.Setenv(e2e.TestExperimentFlag, "true")
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		reattachCh := make(chan *plugin.ReattachConfig)
		closeCh := make(chan struct{})
		provider := &providerServer{
			ProviderServer: grpcwrap.Provider6(simple.Provider()),
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		go plugin.Serve(&plugin.ServeConfig{
			Logger: hclog.New(&hclog.LoggerOptions{
				Name:   "plugintest",
				Level:  hclog.Trace,
				Output: io.Discard,
			}),
			Test: &plugin.ServeTestConfig{
				Context:          ctx,
				ReattachConfigCh: reattachCh,
				CloseCh:          closeCh,
			},
			GRPCServer: plugin.DefaultGRPCServer,
			VersionedPlugins: map[int]plugin.PluginSet{
				6: {
					"provider": &tfplugin.GRPCProviderPlugin{
						GRPCProvider: func() proto.ProviderServer {
							return provider
						},
					},
				},
			},
		})
		config := <-reattachCh
		if config == nil {
			t.Fatalf("no reattach config received")
		}
		reattachStr, err := json.Marshal(map[string]reattachConfig{
			"hashicorp/simple6": {
				Protocol:        string(config.Protocol),
				ProtocolVersion: 6,
				Pid:             config.Pid,
				Test:            true,
				Addr: reattachConfigAddr{
					Network: config.Addr.Network(),
					String:  config.Addr.String(),
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))

		//// INIT - using reattached provider.
		_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-no-color")
		if err != nil {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}

		// Assert backend state file says the provider is a reattached
		statePath := filepath.Join(tf.WorkDir(), ".terraform", command.DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil || s.StateStore == nil {
			t.Fatal("expected backend state file to be created and include state store details, but it was missing.")
		}
		if s.StateStore.ProviderSupplyMode != getproviders.Reattached {
			t.Fatalf("expected state store provider supply mode to be 'reattached', got '%s'", s.StateStore.ProviderSupplyMode)
		}

		//// PLAN - using same provider but supplied via dev_override instead of reattach config.

		// No longer using reattached providers.
		tf.RemoveEnv("TF_REATTACH_PROVIDERS")

		// Build the provider binary and direct Terraform to use it via dev_override, which should cause Terraform to treat it as a dev_override in a CLI configuration file.
		simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		if err := os.Rename(simple6ProviderExe, simple6Provider); err != nil {
			t.Fatal(err)
		}
		cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, tf.WorkDir())
		if err := os.WriteFile(tf.Path("dev_override.tfrc"), []byte(cliCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path("dev_override.tfrc"))

		planFile := "testplan"
		stdout, stderr, err := tf.Run("plan", "-out="+planFile, "-no-color")
		if err != nil {
			t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stdout, "Warning: Provider development overrides are in effect") {
			t.Fatalf("expected warning about provider development overrides being in effect, but it was missing from output:\n%s", stdout)
		}

		//// APPLY
		_, stderr, err = tf.Run("apply", "-auto-approve", "-no-color", planFile)
		if err != nil {
			t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
		}
	})

	t.Run("users are prompted to migrate state when they use an unmanaged provider (dev_override) for plan and apply, after initializing a project with a managed provider", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

		t.Setenv(e2e.TestExperimentFlag, "true")
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Build provider binaries that will be used via a filesystem mirror/-plugin-dir flag.
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

		//// INIT - using managed provider.
		_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
		if err != nil {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}

		// Assert backend state file says the provider is a managed provider
		statePath := filepath.Join(tf.WorkDir(), ".terraform", command.DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil || s.StateStore == nil {
			t.Fatal("expected backend state file to be created and include state store details, but it was missing.")
		}
		if s.StateStore.ProviderSupplyMode != getproviders.ManagedByTerraform {
			t.Fatalf("expected state store provider supply mode to be 'managed_by_terraform', got '%s'", s.StateStore.ProviderSupplyMode)
		}

		//// PLAN - using same provider but dev_overrides now.

		// Delete the cache directory, to ensure that's no longer in use.
		if err := os.RemoveAll(tf.Path("cache")); err != nil {
			t.Fatal(err)
		}

		// Build a new provider binary and direct Terraform to use it via CLI configuration file.
		simple6Provider = filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe = e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		if err := os.Rename(simple6ProviderExe, simple6Provider); err != nil {
			t.Fatal(err)
		}
		cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, tf.WorkDir())
		if err := os.WriteFile(tf.Path("dev_override.tfrc"), []byte(cliCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path("dev_override.tfrc"))

		planFile := "testplan"
		stdout, stderr, err := tf.Run("plan", "-out="+planFile, "-no-color")
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
		}
		devOverrideMsg := "Warning: Provider development overrides are in effect"
		if !strings.Contains(stdout, devOverrideMsg) {
			t.Fatalf("expected output to include %q, but it was missing from output:\n%s", devOverrideMsg, stdout)
		}
		initErrorMsg := "Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\""
		if !strings.Contains(stderr, initErrorMsg) {
			t.Fatalf("expected error output to include %q, but it was missing from output:\n%s", initErrorMsg, stderr)
		}
	})

	t.Run("users are prompted to migrate state when using a managed provider for plan and apply, after initializing a project with an unmanaged provider (dev_override) for PSS", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

		t.Setenv(e2e.TestExperimentFlag, "true")
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Build a new provider binary and direct Terraform to use it via CLI configuration file.
		simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		if err := os.Rename(simple6ProviderExe, simple6Provider); err != nil {
			t.Fatal(err)
		}
		cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, tf.WorkDir())
		if err := os.WriteFile(tf.Path("dev_override.tfrc"), []byte(cliCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path("dev_override.tfrc"))

		// INIT - using dev_override provider.

		stdout, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-no-color")
		if err != nil {
			t.Fatalf("unexpected error during init: %s\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stdout, "Warning: Provider development overrides are in effect") {
			t.Fatalf("expected warning about provider development overrides being in effect, but it was missing from output:\n%s", stdout)
		}

		// Assert backend state file says the provider is a dev_override provider
		statePath := filepath.Join(tf.WorkDir(), ".terraform", command.DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil || s.StateStore == nil {
			t.Fatal("expected backend state file to be created and include state store details, but it was missing.")
		}
		if s.StateStore.ProviderSupplyMode != getproviders.DevOverride {
			t.Fatalf("expected state store provider supply mode to be 'dev_override', got '%s'", s.StateStore.ProviderSupplyMode)
		}

		// PLAN - using same provider but now it's managed by Terraform.

		// Delete the old binary and CLI configuration file, to ensure that's no longer in use.
		if err := os.RemoveAll(simple6Provider); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(tf.Path("dev_override.tfrc")); err != nil {
			t.Fatal(err)
		}
		tf.RemoveEnv("TF_CLI_CONFIG_FILE")

		// Build provider binaries that will be used via a filesystem mirror/-plugin-dir flag.
		simple6Provider = filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe = e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

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

		_, stderr, err = tf.Run("plan", "-no-color")
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stderr, "Error: Inconsistent dependency lock file") {
			t.Fatalf("expected error to mention inconsistent dependency lock file, got: %s", stderr)
		}
	})
}

// Characterize what happens when the state store is supplied through different methods when the working directory is
// initialised and then re-initialised.
func TestPrimary_stateStore_swapProviderSupplyMode_betweenSuccessiveInits(t *testing.T) {
	// Swapping between different 'unmanaged' provider supply modes doesn't trigger a prompt to migrate state because
	// that change doesn't impact the hash of the state store. The hash is impacted by the Version data, and all unmanaged
	// providers used for PSS will have null version data.
	//
	// In contrast, swapping between a managed provider and any of reattached/dev_override/builtin WILL trigger a hash mismatch
	// because the version data will change.
	t.Run("users are NOT prompted to migrate state if an unmanaged provider used for PSS provider swaps supply mode (e.g. swap from reattached to dev_override) between init and plan+apply", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

		t.Setenv(e2e.TestExperimentFlag, "true")
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		reattachCh := make(chan *plugin.ReattachConfig)
		closeCh := make(chan struct{})
		provider := &providerServer{
			ProviderServer: grpcwrap.Provider6(simple.Provider()),
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		go plugin.Serve(&plugin.ServeConfig{
			Logger: hclog.New(&hclog.LoggerOptions{
				Name:   "plugintest",
				Level:  hclog.Trace,
				Output: io.Discard,
			}),
			Test: &plugin.ServeTestConfig{
				Context:          ctx,
				ReattachConfigCh: reattachCh,
				CloseCh:          closeCh,
			},
			GRPCServer: plugin.DefaultGRPCServer,
			VersionedPlugins: map[int]plugin.PluginSet{
				6: {
					"provider": &tfplugin.GRPCProviderPlugin{
						GRPCProvider: func() proto.ProviderServer {
							return provider
						},
					},
				},
			},
		})
		config := <-reattachCh
		if config == nil {
			t.Fatalf("no reattach config received")
		}
		reattachStr, err := json.Marshal(map[string]reattachConfig{
			"hashicorp/simple6": {
				Protocol:        string(config.Protocol),
				ProtocolVersion: 6,
				Pid:             config.Pid,
				Test:            true,
				Addr: reattachConfigAddr{
					Network: config.Addr.Network(),
					String:  config.Addr.String(),
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))

		//// INIT 1 - using reattached provider.
		_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-no-color")
		if err != nil {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}

		// Assert backend state file says the provider is a reattached
		statePath := filepath.Join(tf.WorkDir(), ".terraform", command.DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil || s.StateStore == nil {
			t.Fatal("expected backend state file to be created and include state store details, but it was missing.")
		}
		if s.StateStore.ProviderSupplyMode != getproviders.Reattached {
			t.Fatalf("expected state store provider supply mode to be 'reattached', got '%s'", s.StateStore.ProviderSupplyMode)
		}

		//// INIT 2 - using same provider but supplied via dev_override instead of reattach config.

		// No longer using reattached providers.
		tf.RemoveEnv("TF_REATTACH_PROVIDERS")

		// Build the provider binary and direct Terraform to use it via dev_override, which should cause Terraform to treat it as a dev_override in a CLI configuration file.
		simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		if err := os.Rename(simple6ProviderExe, simple6Provider); err != nil {
			t.Fatal(err)
		}
		cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, tf.WorkDir())
		if err := os.WriteFile(tf.Path("dev_override.tfrc"), []byte(cliCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path("dev_override.tfrc"))

		stdout, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-no-color")
		if err != nil {
			t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
		}
		expectedMessage := "Terraform has been successfully initialized!"
		if !strings.Contains(stdout, expectedMessage) {
			t.Fatalf("expected %q, but got: %s", expectedMessage, stdout)
		}
	})

	t.Run("users are prompted to migrate state when they init a project with a managed provider for PSS and re-init using an unmanaged provider (dev_override)", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

		t.Setenv(e2e.TestExperimentFlag, "true")
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Build provider binaries that will be used via a filesystem mirror/-plugin-dir flag.
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

		//// INIT 1 - using managed provider.
		_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
		if err != nil {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}

		// Assert backend state file says the provider is a managed provider
		statePath := filepath.Join(tf.WorkDir(), ".terraform", command.DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil || s.StateStore == nil {
			t.Fatal("expected backend state file to be created and include state store details, but it was missing.")
		}
		if s.StateStore.ProviderSupplyMode != getproviders.ManagedByTerraform {
			t.Fatalf("expected state store provider supply mode to be 'managed_by_terraform', got '%s'", s.StateStore.ProviderSupplyMode)
		}

		//// INIT 2 - using same provider but dev_overrides now.

		// Delete the cache directory, to ensure that's no longer in use.
		if err := os.RemoveAll(tf.Path("cache")); err != nil {
			t.Fatal(err)
		}

		// Build a new provider binary and direct Terraform to use it via CLI configuration file.
		simple6Provider = filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe = e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		if err := os.Rename(simple6ProviderExe, simple6Provider); err != nil {
			t.Fatal(err)
		}
		cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, tf.WorkDir())
		if err := os.WriteFile(tf.Path("dev_override.tfrc"), []byte(cliCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path("dev_override.tfrc"))

		_, stderr, err = tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-no-color")
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stderr, "Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"") {
			t.Fatalf("expected error about state store configuration changing, but got:\n%s", stderr)
		}
	})

	t.Run("users are prompted to migrate state when they init a project with an unmanaged provider (dev_override) for PSS and re-init using a managed provider", func(t *testing.T) {
		if !canRunGoBuild {
			// We're running in a separate-build-then-run context, so we can't
			// currently execute this test which depends on being able to build
			// new executable at runtime.
			//
			// (See the comment on canRunGoBuild's declaration for more information.)
			t.Skip("can't run without building a new provider executable")
		}

		fixturePath := filepath.Join("testdata", "full-workflow-with-state-store-fs")

		t.Setenv(e2e.TestExperimentFlag, "true")
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Build a new provider binary and direct Terraform to use it via CLI configuration file.
		simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
		if err := os.Rename(simple6ProviderExe, simple6Provider); err != nil {
			t.Fatal(err)
		}
		cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, tf.WorkDir())
		if err := os.WriteFile(tf.Path("dev_override.tfrc"), []byte(cliCfg), 0644); err != nil {
			t.Fatalf("err: %s", err)
		}
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path("dev_override.tfrc"))

		// INIT 1 - using dev_override provider.

		stdout, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-no-color")
		if err != nil {
			t.Fatalf("unexpected error during init: %s\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stdout, "Warning: Provider development overrides are in effect") {
			t.Fatalf("expected warning about provider development overrides being in effect, but it was missing from output:\n%s", stdout)
		}

		// Assert backend state file says the provider is a dev_override provider
		statePath := filepath.Join(tf.WorkDir(), ".terraform", command.DefaultStateFilename)
		sMgr := &clistate.LocalState{Path: statePath}
		if err := sMgr.RefreshState(); err != nil {
			t.Fatal("Failed to load state:", err)
		}
		s := sMgr.State()
		if s == nil || s.StateStore == nil {
			t.Fatal("expected backend state file to be created and include state store details, but it was missing.")
		}
		if s.StateStore.ProviderSupplyMode != getproviders.DevOverride {
			t.Fatalf("expected state store provider supply mode to be 'dev_override', got '%s'", s.StateStore.ProviderSupplyMode)
		}

		// INIT 2 - using same provider but now it's managed by Terraform.

		// Delete the old binary and CLI configuration file, to ensure that's no longer in use.
		if err := os.RemoveAll(simple6Provider); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(tf.Path("dev_override.tfrc")); err != nil {
			t.Fatal(err)
		}
		tf.RemoveEnv("TF_CLI_CONFIG_FILE")

		// Build provider binaries that will be used via a filesystem mirror/-plugin-dir flag.
		simple6Provider = filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
		simple6ProviderExe = e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

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

		_, stderr, err = tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
		if err.Error() != "exit status 1" {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stderr, "Error: State store initialization required, please run \"terraform state migrate\" or \"terraform init -reconfigure\"") {
			t.Fatalf("expected error about state store configuration changing, but got:\n%s", stderr)
		}
	})
}
