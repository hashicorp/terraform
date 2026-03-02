// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestInit2_dynamicSourceErrors(t *testing.T) {
	tests := map[string]struct {
		fixture   string
		args      []string
		wantError string
	}{
		"version constraint added to previously unversioned module": {
			fixture:   "add-version-constraint",
			args:      []string{"-get=false"},
			wantError: "Module version requirements have changed",
		},
		"invalid registry source with version argument": {
			fixture:   "invalid-registry-source-with-module",
			wantError: "Invalid registry module source address",
		},
		"local source with version argument": {
			fixture:   "local-source-with-version",
			wantError: "Invalid registry module source address",
		},
		"non-const variable in module source": {
			fixture:   "local-source-with-non-const-variable",
			args:      []string{"-var", "module_name=example"},
			wantError: "Invalid module source",
		},
		"resource reference in module source": {
			fixture:   "source-with-resource-reference",
			wantError: "Invalid module source",
		},
		"module output reference in module source": {
			fixture:   "source-with-module-output-reference",
			wantError: "Invalid module source",
		},
		"each.key in module source": {
			fixture:   "each-in-module-source",
			wantError: "Invalid module source",
		},
		"count.index in module source": {
			fixture:   "count-in-module-source",
			wantError: "Invalid module source",
		},
		"terraform.workspace in module source": {
			fixture:   "terraform-attr-in-module-source",
			wantError: "Invalid module source",
		},
		"required const variable not set": {
			fixture:   "local-source-with-variable",
			wantError: "No value for required variable",
		},
		"override default with nonexistent module": {
			fixture:   "local-source-with-variable-default",
			args:      []string{"-var", "module_name=nonexistent"},
			wantError: "", // any error; the module directory doesn't exist
		},
		"version mismatch with dynamic constraint": {
			fixture:   "plan-with-version-mismatch",
			args:      []string{"-get=false", "-var", "module_version=0.0.2"},
			wantError: "Module version requirements have changed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", tc.fixture)), td)
			t.Chdir(td)

			ui := new(cli.MockUi)
			view, done := testView(t)
			c := &InitCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
				},
			}

			code := c.Run(tc.args)
			testOutput := done(t)
			if code != 1 {
				t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
			}

			if tc.wantError != "" {
				got := testOutput.All()
				if !strings.Contains(got, tc.wantError) {
					t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, tc.wantError)
				}
			}
		})
	}
}

func TestInit2_dynamicSourceSuccess(t *testing.T) {
	tests := map[string]struct {
		fixture string
		args    []string
	}{
		"const variable via -var": {
			fixture: "local-source-with-variable",
			args:    []string{"-var", "module_name=example"},
		},
		"const variable with default value": {
			fixture: "local-source-with-variable-default",
		},
		"local value referencing const variable": {
			fixture: "local-source-with-local-value",
			args:    []string{"-var", "module_name=example"},
		},
		"nested module with variable passed through parent": {
			fixture: "nested-module-with-variable-source",
			args:    []string{"-var", "child_name=child"},
		},
		"const variable from tfvars file": {
			fixture: "local-source-with-varsfile",
			args:    []string{"-var-file", "test.tfvars"},
		},
		"path.module in module source": {
			fixture: "path-attr-in-module-source",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", tc.fixture)), td)
			t.Chdir(td)

			ui := new(cli.MockUi)
			view, done := testView(t)
			c := &InitCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(testProvider()),
					Ui:               ui,
					View:             view,
				},
			}

			code := c.Run(tc.args)
			testOutput := done(t)
			if code != 0 {
				t.Fatalf("got exit status %d; want 0\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
			}
		})
	}
}

func TestInit2_getFalseWithDynamicSource(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "get-false-with-dynamic-source")), td)
	t.Chdir(td)

	// First, run init normally to install the module
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{"-var", "module_name=example"}
	code := c.Run(args)
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("first init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}

	// Now run init with -get=false; should succeed since modules are already installed
	ui2 := new(cli.MockUi)
	view2, done2 := testView(t)
	c2 := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui2,
			View:             view2,
		},
	}

	args2 := []string{"-get=false", "-var", "module_name=example"}
	code = c2.Run(args2)
	testOutput2 := done2(t)
	if code != 0 {
		t.Fatalf("init -get=false failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, testOutput2.Stderr(), testOutput2.Stdout())
	}
}

func TestInit2_getFalseWithDynamicSourceNotInstalled(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "get-false-with-dynamic-source")), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// Run init with -get=false without having installed modules first
	args := []string{"-get=false", "-var", "module_name=example"}
	code := c.Run(args)
	testOutput := done(t)
	if code != 1 {
		t.Fatalf("got exit status %d; want 1\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}
}

func TestInit2_reinitWithDifferentVariable(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "local-source-with-variable-default")), td)
	t.Chdir(td)

	// First init with default variable (example)
	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	code := c.Run([]string{})
	testOutput := done(t)
	if code != 0 {
		t.Fatalf("first init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}

	// Re-init with different variable
	ui2 := new(cli.MockUi)
	view2, done2 := testView(t)
	c2 := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui2,
			View:             view2,
		},
	}

	code = c2.Run([]string{"-var", "module_name=alternate"})
	testOutput2 := done2(t)
	if code != 0 {
		t.Fatalf("second init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, testOutput2.Stderr(), testOutput2.Stdout())
	}
}

func TestInit2_fromModuleWithDynamicSource(t *testing.T) {
	// TODO: -from-module currently panics when the copied configuration
	// contains a dynamic module source (e.g. "./modules/${var.module_name}").
	t.Skip("skipping: -from-module panics on dynamic module sources (see TODO in from_module.go)")

	// Create an empty target directory for -from-module to copy into
	td := t.TempDir()
	t.Chdir(td)

	ui := new(cli.MockUi)
	view, done := testView(t)
	c := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
			View:             view,
		},
	}

	// Use -from-module to copy the source module (which has a dynamic source)
	// into the empty working directory. This should copy the files but the
	// nested dynamic module won't be resolved by -from-module itself.
	srcDir := testFixturePath(filepath.Join("dynamic-module-sources", "from-module-with-dynamic-source", "source-module"))
	args := []string{"-from-module=" + srcDir}
	code := c.Run(args)
	testOutput := done(t)

	// -from-module should succeed in copying. The dynamic module source
	// within the copied configuration won't be resolved yet — that requires
	// a separate init with the variable value.
	if code != 0 {
		t.Fatalf("init -from-module failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, testOutput.Stderr(), testOutput.Stdout())
	}

	// Verify the main.tf was copied
	if _, err := os.Stat(filepath.Join(td, "main.tf")); os.IsNotExist(err) {
		t.Fatal("main.tf was not copied from the source module")
	}
}

func TestPlan_dynamicModuleSource(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "plan-with-dynamic-source")), td)
	t.Chdir(td)

	p := planFixtureProvider()
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	args := []string{"-var", "module_name=example"}

	initUi := new(cli.MockUi)
	initView, initDone := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               initUi,
			View:             initView,
			ProviderSource:   providerSource,
		},
	}

	initCode := initCmd.Run(args)
	initOutput := initDone(t)
	if initCode != 0 {
		t.Fatalf("init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", initCode, initOutput.Stderr(), initOutput.Stdout())
	}

	// Now run plan
	planUi := new(cli.MockUi)
	planView, planDone := testView(t)
	planCmd := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               planUi,
			View:             planView,
		},
	}

	planCode := planCmd.Run(args)
	planOutput := planDone(t)
	if planCode != 0 {
		t.Fatalf("plan failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", planCode, planOutput.Stderr(), planOutput.Stdout())
	}

	output := planOutput.Stdout()
	if !strings.Contains(output, "1 to add") {
		t.Fatalf("expected plan to show 1 resource to add, got:\n%s", output)
	}
}

func TestPlan_dynamicModuleSourceMismatch(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "plan-with-dynamic-source")), td)
	t.Chdir(td)

	p := planFixtureProvider()
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()
	args := []string{"-var", "module_name=example"}

	initUi := new(cli.MockUi)
	initView, initDone := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               initUi,
			View:             initView,
			ProviderSource:   providerSource,
		},
	}

	initCode := initCmd.Run(args)
	initOutput := initDone(t)
	if initCode != 0 {
		t.Fatalf("init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", initCode, initOutput.Stderr(), initOutput.Stdout())
	}

	// Now run plan with a different variable value
	planUi := new(cli.MockUi)
	planView, planDone := testView(t)
	planCmd := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               planUi,
			View:             planView,
		},
	}

	planArgs := []string{"-var", "module_name=nonexistent"}
	code := planCmd.Run(planArgs)
	planOutput := planDone(t)
	if code == 0 {
		t.Fatalf("expected plan to fail, but got exit status 0\nstdout:\n%s", planOutput.Stdout())
	}
}

func TestApply_dynamicModuleSource(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "apply-with-dynamic-source")), td)
	t.Chdir(td)

	p := planFixtureProvider()
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()
	args := []string{"-var", "module_name=example"}

	initUi := new(cli.MockUi)
	initView, initDone := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               initUi,
			View:             initView,
			ProviderSource:   providerSource,
		},
	}

	initCode := initCmd.Run(args)
	initOutput := initDone(t)
	if initCode != 0 {
		t.Fatalf("init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", initCode, initOutput.Stderr(), initOutput.Stdout())
	}

	applyUi := new(cli.MockUi)
	applyView, applyDone := testView(t)
	applyCmd := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               applyUi,
			View:             applyView,
		},
	}

	applyArgs := []string{"-auto-approve", "-var", "module_name=example"}
	code := applyCmd.Run(applyArgs)
	applyOutput := applyDone(t)
	if code != 0 {
		t.Fatalf("apply failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, applyOutput.Stderr(), applyOutput.Stdout())
	}

	output := applyOutput.Stdout()
	if !strings.Contains(output, "Apply complete!") {
		t.Fatalf("expected apply to succeed, got:\n%s", output)
	}
}

func TestApply_dynamicModuleSourceWithDefaultPlanFile(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "apply-plan-with-dynamic-source")), td)
	t.Chdir(td)

	p := planFixtureProvider()
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	initUi := new(cli.MockUi)
	initView, initDone := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               initUi,
			View:             initView,
			ProviderSource:   providerSource,
		},
	}

	initCode := initCmd.Run([]string{})
	initOutput := initDone(t)
	if initCode != 0 {
		t.Fatalf("init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", initCode, initOutput.Stderr(), initOutput.Stdout())
	}

	// Run plan with -out
	planPath := filepath.Join(td, "saved.plan")
	planUi := new(cli.MockUi)
	planView, planDone := testView(t)
	planCmd := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               planUi,
			View:             planView,
		},
	}

	planArgs := []string{"-out", planPath}
	code := planCmd.Run(planArgs)
	planOutput := planDone(t)
	if code != 0 {
		t.Fatalf("plan failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, planOutput.Stderr(), planOutput.Stdout())
	}

	// Verify the plan file was created
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Fatalf("plan file was not created at %s", planPath)
	}

	// Apply the saved plan
	applyUi := new(cli.MockUi)
	applyView, applyDone := testView(t)
	applyCmd := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               applyUi,
			View:             applyView,
		},
	}

	applyArgs := []string{planPath}
	code = applyCmd.Run(applyArgs)
	applyOutput := applyDone(t)
	if code != 0 {
		t.Fatalf("apply failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", code, applyOutput.Stderr(), applyOutput.Stdout())
	}

	output := applyOutput.Stdout()
	if !strings.Contains(output, "Apply complete!") {
		t.Fatalf("expected apply to succeed, got:\n%s", output)
	}
}

func TestPlan_dynamicModuleSourceWithCount(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "module-with-count")), td)
	t.Chdir(td)

	p := planFixtureProvider()
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	args := []string{"-var", "module_name=example"}

	initUi := new(cli.MockUi)
	initView, initDone := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               initUi,
			View:             initView,
			ProviderSource:   providerSource,
		},
	}

	initCode := initCmd.Run(args)
	initOutput := initDone(t)
	if initCode != 0 {
		t.Fatalf("init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", initCode, initOutput.Stderr(), initOutput.Stdout())
	}

	// Now run plan
	planUi := new(cli.MockUi)
	planView, planDone := testView(t)
	planCmd := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               planUi,
			View:             planView,
		},
	}

	planCode := planCmd.Run(args)
	planOutput := planDone(t)
	if planCode != 0 {
		t.Fatalf("plan failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", planCode, planOutput.Stderr(), planOutput.Stdout())
	}

	output := planOutput.Stdout()
	if !strings.Contains(output, "2 to add") {
		t.Fatalf("expected plan to show 2 resources to add, got:\n%s", output)
	}
}

func TestPlan_dynamicModuleSourceWithForEach(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "module-with-for-each")), td)
	t.Chdir(td)

	p := planFixtureProvider()
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
	})
	defer close()

	args := []string{"-var", "module_name=example"}

	initUi := new(cli.MockUi)
	initView, initDone := testView(t)
	initCmd := &InitCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               initUi,
			View:             initView,
			ProviderSource:   providerSource,
		},
	}

	initCode := initCmd.Run(args)
	initOutput := initDone(t)
	if initCode != 0 {
		t.Fatalf("init failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", initCode, initOutput.Stderr(), initOutput.Stdout())
	}

	// Now run plan
	planUi := new(cli.MockUi)
	planView, planDone := testView(t)
	planCmd := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               planUi,
			View:             planView,
		},
	}

	planCode := planCmd.Run(args)
	planOutput := planDone(t)
	if planCode != 0 {
		t.Fatalf("plan failed with exit status %d\nstderr:\n%s\n\nstdout:\n%s", planCode, planOutput.Stderr(), planOutput.Stdout())
	}

	output := planOutput.Stdout()
	if !strings.Contains(output, "2 to add") {
		t.Fatalf("expected plan to show 2 resources to add, got:\n%s", output)
	}
}

func TestPlan_dynamicModuleVersionMismatch(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(filepath.Join("dynamic-module-sources", "plan-with-version-mismatch")), td)
	t.Chdir(td)

	p := planFixtureProvider()

	// Plan should fail because the installed module version (0.0.1 in
	// modules.json) doesn't satisfy the constraint we provide.
	planUi := new(cli.MockUi)
	planView, planDone := testView(t)
	planCmd := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               planUi,
			View:             planView,
		},
	}

	planArgs := []string{"-var", "module_version=0.0.2"}
	code := planCmd.Run(planArgs)
	planOutput := planDone(t)
	if code == 0 {
		t.Fatalf("expected plan to fail, but got exit status 0\nstdout:\n%s", planOutput.Stdout())
	}
	got := planOutput.All()

	want := "Module version requirements have changed"
	if !strings.Contains(got, want) {
		t.Fatalf("wrong error\ngot:\n%s\n\nwant: containing %q", got, want)
	}
}
