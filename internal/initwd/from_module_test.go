package initwd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/internal/copydir"
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestDirFromModule_registry(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("this test accesses registry.terraform.io and github.com; set TF_ACC=1 to run it")
	}

	fixtureDir := filepath.Clean("testdata/empty")
	tmpDir, done := tempChdir(t, fixtureDir)
	defer done()

	// the module installer runs filepath.EvalSymlinks() on the destination
	// directory before copying files, and the resultant directory is what is
	// returned by the install hooks. Without this, tests could fail on machines
	// where the default temp dir was a symlink.
	dir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Error(err)
	}
	modsDir := filepath.Join(dir, ".terraform/modules")

	hooks := &testInstallHooks{}

	reg := registry.NewClient(nil, nil)
	diags := DirFromModule(dir, modsDir, "hashicorp/module-installer-acctest/aws//examples/main", reg, hooks)
	assertNoDiagnostics(t, diags)

	v := version.Must(version.NewVersion("0.0.2"))

	wantCalls := []testInstallHookCall{
		// The module specified to populate the root directory is not mentioned
		// here, because the hook mechanism is defined to talk about descendent
		// modules only and so a caller to InitDirFromModule is expected to
		// produce its own user-facing announcement about the root module being
		// installed.

		// Note that "root" in the following examples is, confusingly, the
		// label on the module block in the example we've installed here:
		//     module "root" {

		{
			Name:        "Download",
			ModuleAddr:  "root",
			PackageAddr: "hashicorp/module-installer-acctest/aws",
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "root",
			Version:    v,
			LocalPath:  filepath.Join(dir, fmt.Sprintf(".terraform/modules/root/terraform-aws-module-installer-acctest-%s", v)),
		},
		{
			Name:       "Install",
			ModuleAddr: "root.child_a",
			LocalPath:  filepath.Join(dir, fmt.Sprintf(".terraform/modules/root/terraform-aws-module-installer-acctest-%s/modules/child_a", v)),
		},
		{
			Name:       "Install",
			ModuleAddr: "root.child_a.child_b",
			LocalPath:  filepath.Join(dir, fmt.Sprintf(".terraform/modules/root/terraform-aws-module-installer-acctest-%s/modules/child_b", v)),
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modsDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	if assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags)) {
		return
	}

	wantTraces := map[string]string{
		"":                     "in example",
		"root":                 "in root module",
		"root.child_a":         "in child_a module",
		"root.child_a.child_b": "in child_b module",
	}
	gotTraces := map[string]string{}
	config.DeepEach(func(c *configs.Config) {
		path := strings.Join(c.Path, ".")
		if c.Module.Variables["v"] == nil {
			gotTraces[path] = "<missing>"
			return
		}
		varDesc := c.Module.Variables["v"].Description
		gotTraces[path] = varDesc
	})
	assertResultDeepEqual(t, gotTraces, wantTraces)
}

func TestDirFromModule_submodules(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/empty")
	fromModuleDir, err := filepath.Abs("./testdata/local-modules")
	if err != nil {
		t.Fatal(err)
	}

	// DirFromModule will expand ("canonicalize") the pathnames, so we must do
	// the same for our "wantCalls" comparison values. Otherwise this test
	// will fail when building in a source tree with symlinks in $PWD.
	//
	// See also: https://github.com/hashicorp/terraform/issues/26014
	//
	fromModuleDirRealpath, err := filepath.EvalSymlinks(fromModuleDir)
	if err != nil {
		t.Error(err)
	}

	tmpDir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}
	dir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Error(err)
	}
	modInstallDir := filepath.Join(dir, ".terraform/modules")

	diags := DirFromModule(dir, modInstallDir, fromModuleDir, nil, hooks)
	assertNoDiagnostics(t, diags)
	wantCalls := []testInstallHookCall{
		{
			Name:       "Install",
			ModuleAddr: "child_a",
			LocalPath:  filepath.Join(fromModuleDirRealpath, "child_a"),
		},
		{
			Name:       "Install",
			ModuleAddr: "child_a.child_b",
			LocalPath:  filepath.Join(fromModuleDirRealpath, "child_a/child_b"),
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modInstallDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	if assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags)) {
		return
	}
	wantTraces := map[string]string{
		"":                "in root module",
		"child_a":         "in child_a module",
		"child_a.child_b": "in child_b module",
	}
	gotTraces := map[string]string{}

	config.DeepEach(func(c *configs.Config) {
		path := strings.Join(c.Path, ".")
		if c.Module.Variables["v"] == nil {
			gotTraces[path] = "<missing>"
			return
		}
		varDesc := c.Module.Variables["v"].Description
		gotTraces[path] = varDesc
	})
	assertResultDeepEqual(t, gotTraces, wantTraces)
}

// TestDirFromModule_rel_submodules is similar to the test above, but the
// from-module is relative to the install dir ("../"):
// https://github.com/hashicorp/terraform/issues/23010
func TestDirFromModule_rel_submodules(t *testing.T) {
	// This test creates a tmpdir with the following directory structure:
	// - tmpdir/local-modules (with contents of testdata/local-modules)
	// - tmpdir/empty: the workDir we CD into for the test
	// - tmpdir/empty/target (target, the destination for init -from-module)
	tmpDir, err := ioutil.TempDir("", "terraform-configload")
	if err != nil {
		t.Fatal(err)
	}
	fromModuleDir := filepath.Join(tmpDir, "local-modules")
	workDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(fromModuleDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := copydir.CopyDir(fromModuleDir, "testdata/local-modules"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(workDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(targetDir)
	if err != nil {
		t.Fatalf("failed to switch to temp dir %s: %s", tmpDir, err)
	}
	defer os.Chdir(oldDir)
	defer os.RemoveAll(tmpDir)

	hooks := &testInstallHooks{}

	modInstallDir := ".terraform/modules"
	sourceDir := "../local-modules"
	diags := DirFromModule(".", modInstallDir, sourceDir, nil, hooks)
	assertNoDiagnostics(t, diags)
	wantCalls := []testInstallHookCall{
		{
			Name:       "Install",
			ModuleAddr: "child_a",
			LocalPath:  filepath.Join(sourceDir, "child_a"),
		},
		{
			Name:       "Install",
			ModuleAddr: "child_a.child_b",
			LocalPath:  filepath.Join(sourceDir, "child_a/child_b"),
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modInstallDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	if assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags)) {
		return
	}
	wantTraces := map[string]string{
		"":                "in root module",
		"child_a":         "in child_a module",
		"child_a.child_b": "in child_b module",
	}
	gotTraces := map[string]string{}

	config.DeepEach(func(c *configs.Config) {
		path := strings.Join(c.Path, ".")
		if c.Module.Variables["v"] == nil {
			gotTraces[path] = "<missing>"
			return
		}
		varDesc := c.Module.Variables["v"].Description
		gotTraces[path] = varDesc
	})
	assertResultDeepEqual(t, gotTraces, wantTraces)
}
