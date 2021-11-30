package initwd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-test/deep"
	"github.com/google/go-cmp/cmp"
	version "github.com/hashicorp/go-version"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/copy"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/tfdiags"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestModuleInstaller(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/local-modules")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)
	assertNoDiagnostics(t, diags)

	wantCalls := []testInstallHookCall{
		{
			Name:        "Install",
			ModuleAddr:  "child_a",
			PackageAddr: "",
			LocalPath:   "child_a",
		},
		{
			Name:        "Install",
			ModuleAddr:  "child_a.child_b",
			PackageAddr: "",
			LocalPath:   "child_a/child_b",
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags))

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

func TestModuleInstaller_error(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/local-module-error")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		assertDiagnosticSummary(t, diags, "Invalid module source address")
	}
}

func TestModuleInstaller_packageEscapeError(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/load-module-package-escape")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	// For this particular test we need an absolute path in the root module
	// that must actually resolve to our temporary directory in "dir", so
	// we need to do a little rewriting. We replace the arbitrary placeholder
	// %%BASE%% with the temporary directory path.
	{
		rootFilename := filepath.Join(dir, "package-escape.tf")
		template, err := ioutil.ReadFile(rootFilename)
		if err != nil {
			t.Fatal(err)
		}
		final := bytes.ReplaceAll(template, []byte("%%BASE%%"), []byte(filepath.ToSlash(dir)))
		err = ioutil.WriteFile(rootFilename, final, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		assertDiagnosticSummary(t, diags, "Local module path escapes module package")
	}
}

func TestModuleInstaller_explicitPackageBoundary(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/load-module-package-prefix")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	// For this particular test we need an absolute path in the root module
	// that must actually resolve to our temporary directory in "dir", so
	// we need to do a little rewriting. We replace the arbitrary placeholder
	// %%BASE%% with the temporary directory path.
	{
		rootFilename := filepath.Join(dir, "package-prefix.tf")
		template, err := ioutil.ReadFile(rootFilename)
		if err != nil {
			t.Fatal(err)
		}
		final := bytes.ReplaceAll(template, []byte("%%BASE%%"), []byte(filepath.ToSlash(dir)))
		err = ioutil.WriteFile(rootFilename, final, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)

	if diags.HasErrors() {
		t.Fatalf("unexpected errors\n%s", diags.Err().Error())
	}
}

func TestModuleInstaller_invalid_version_constraint_error(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/invalid-version-constraint")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		// We use the presence of the "version" argument as a heuristic for
		// user intent to use a registry module, and so we intentionally catch
		// this as an invalid registry module address rather than an invalid
		// version constraint, so we can surface the specific address parsing
		// error instead of a generic version constraint error.
		assertDiagnosticSummary(t, diags, "Invalid registry module source address")
	}
}

func TestModuleInstaller_invalidVersionConstraintGetter(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/invalid-version-constraint")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		// We use the presence of the "version" argument as a heuristic for
		// user intent to use a registry module, and so we intentionally catch
		// this as an invalid registry module address rather than an invalid
		// version constraint, so we can surface the specific address parsing
		// error instead of a generic version constraint error.
		assertDiagnosticSummary(t, diags, "Invalid registry module source address")
	}
}

func TestModuleInstaller_invalidVersionConstraintLocal(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/invalid-version-constraint-local")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		// We use the presence of the "version" argument as a heuristic for
		// user intent to use a registry module, and so we intentionally catch
		// this as an invalid registry module address rather than an invalid
		// version constraint, so we can surface the specific address parsing
		// error instead of a generic version constraint error.
		assertDiagnosticSummary(t, diags, "Invalid registry module source address")
	}
}

func TestModuleInstaller_symlink(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/local-module-symlink")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(context.Background(), ".", false, hooks)
	assertNoDiagnostics(t, diags)

	wantCalls := []testInstallHookCall{
		{
			Name:        "Install",
			ModuleAddr:  "child_a",
			PackageAddr: "",
			LocalPath:   "child_a",
		},
		{
			Name:        "Install",
			ModuleAddr:  "child_a.child_b",
			PackageAddr: "",
			LocalPath:   "child_a/child_b",
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags))

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

func TestLoaderInstallModules_registry(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("this test accesses registry.terraform.io and github.com; set TF_ACC=1 to run it")
	}

	fixtureDir := filepath.Clean("testdata/registry-modules")
	tmpDir, done := tempChdir(t, fixtureDir)
	// the module installer runs filepath.EvalSymlinks() on the destination
	// directory before copying files, and the resultant directory is what is
	// returned by the install hooks. Without this, tests could fail on machines
	// where the default temp dir was a symlink.
	dir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Error(err)
	}

	defer done()

	hooks := &testInstallHooks{}
	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, registry.NewClient(nil, nil))
	_, diags := inst.InstallModules(context.Background(), dir, false, hooks)
	assertNoDiagnostics(t, diags)

	v := version.Must(version.NewVersion("0.0.1"))

	wantCalls := []testInstallHookCall{
		// the configuration builder visits each level of calls in lexicographical
		// order by name, so the following list is kept in the same order.

		// acctest_child_a accesses //modules/child_a directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_a",
			PackageAddr: "registry.terraform.io/hashicorp/module-installer-acctest/aws", // intentionally excludes the subdir because we're downloading the whole package here
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a",
			Version:    v,
			// NOTE: This local path and the other paths derived from it below
			// can vary depending on how the registry is implemented. At the
			// time of writing this test, registry.terraform.io returns
			// git repository source addresses and so this path refers to the
			// root of the git clone, but historically the registry referred
			// to GitHub-provided tar archives which meant that there was an
			// extra level of subdirectory here for the typical directory
			// nesting in tar archives, which would've been reflected as
			// an extra segment on this path. If this test fails due to an
			// additional path segment in future, then a change to the upstream
			// registry might be the root cause.
			LocalPath: filepath.Join(dir, ".terraform/modules/acctest_child_a/modules/child_a"),
		},

		// acctest_child_a.child_b
		// (no download because it's a relative path inside acctest_child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_a/modules/child_b"),
		},

		// acctest_child_b accesses //modules/child_b directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_b",
			PackageAddr: "registry.terraform.io/hashicorp/module-installer-acctest/aws", // intentionally excludes the subdir because we're downloading the whole package here
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_b",
			Version:    v,
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_b/modules/child_b"),
		},

		// acctest_root
		{
			Name:        "Download",
			ModuleAddr:  "acctest_root",
			PackageAddr: "registry.terraform.io/hashicorp/module-installer-acctest/aws",
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_root",
			Version:    v,
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root"),
		},

		// acctest_root.child_a
		// (no download because it's a relative path inside acctest_root)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/modules/child_a"),
		},

		// acctest_root.child_a.child_b
		// (no download because it's a relative path inside acctest_root, via acctest_root.child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/modules/child_b"),
		},
	}

	if diff := cmp.Diff(wantCalls, hooks.Calls); diff != "" {
		t.Fatalf("wrong installer calls\n%s", diff)
	}

	//check that the registry reponses were cached
	packageAddr := addrs.ModuleRegistryPackage{
		Host:         svchost.Hostname("registry.terraform.io"),
		Namespace:    "hashicorp",
		Name:         "module-installer-acctest",
		TargetSystem: "aws",
	}
	if _, ok := inst.registryPackageVersions[packageAddr]; !ok {
		t.Errorf("module versions cache was not populated\ngot: %s\nwant: key hashicorp/module-installer-acctest/aws", spew.Sdump(inst.registryPackageVersions))
	}
	if _, ok := inst.registryPackageSources[moduleVersion{module: packageAddr, version: "0.0.1"}]; !ok {
		t.Errorf("module download url cache was not populated\ngot: %s", spew.Sdump(inst.registryPackageSources))
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags))

	wantTraces := map[string]string{
		"":                             "in local caller for registry-modules",
		"acctest_root":                 "in root module",
		"acctest_root.child_a":         "in child_a module",
		"acctest_root.child_a.child_b": "in child_b module",
		"acctest_child_a":              "in child_a module",
		"acctest_child_a.child_b":      "in child_b module",
		"acctest_child_b":              "in child_b module",
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

func TestLoaderInstallModules_goGetter(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("this test accesses github.com; set TF_ACC=1 to run it")
	}

	fixtureDir := filepath.Clean("testdata/go-getter-modules")
	tmpDir, done := tempChdir(t, fixtureDir)
	// the module installer runs filepath.EvalSymlinks() on the destination
	// directory before copying files, and the resultant directory is what is
	// returned by the install hooks. Without this, tests could fail on machines
	// where the default temp dir was a symlink.
	dir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer done()

	hooks := &testInstallHooks{}
	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, registry.NewClient(nil, nil))
	_, diags := inst.InstallModules(context.Background(), dir, false, hooks)
	assertNoDiagnostics(t, diags)

	wantCalls := []testInstallHookCall{
		// the configuration builder visits each level of calls in lexicographical
		// order by name, so the following list is kept in the same order.

		// acctest_child_a accesses //modules/child_a directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_a",
			PackageAddr: "git::https://github.com/hashicorp/terraform-aws-module-installer-acctest.git?ref=v0.0.1", // intentionally excludes the subdir because we're downloading the whole repo here
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_a/modules/child_a"),
		},

		// acctest_child_a.child_b
		// (no download because it's a relative path inside acctest_child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_a/modules/child_b"),
		},

		// acctest_child_b accesses //modules/child_b directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_b",
			PackageAddr: "git::https://github.com/hashicorp/terraform-aws-module-installer-acctest.git?ref=v0.0.1", // intentionally excludes the subdir because we're downloading the whole package here
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_b/modules/child_b"),
		},

		// acctest_root
		{
			Name:        "Download",
			ModuleAddr:  "acctest_root",
			PackageAddr: "git::https://github.com/hashicorp/terraform-aws-module-installer-acctest.git?ref=v0.0.1",
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_root",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root"),
		},

		// acctest_root.child_a
		// (no download because it's a relative path inside acctest_root)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/modules/child_a"),
		},

		// acctest_root.child_a.child_b
		// (no download because it's a relative path inside acctest_root, via acctest_root.child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/modules/child_b"),
		},
	}

	if diff := cmp.Diff(wantCalls, hooks.Calls); diff != "" {
		t.Fatalf("wrong installer calls\n%s", diff)
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	config, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, tfdiags.Diagnostics{}.Append(loadDiags))

	wantTraces := map[string]string{
		"":                             "in local caller for go-getter-modules",
		"acctest_root":                 "in root module",
		"acctest_root.child_a":         "in child_a module",
		"acctest_root.child_a.child_b": "in child_b module",
		"acctest_child_a":              "in child_a module",
		"acctest_child_a.child_b":      "in child_b module",
		"acctest_child_b":              "in child_b module",
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

type testInstallHooks struct {
	Calls []testInstallHookCall
}

type testInstallHookCall struct {
	Name        string
	ModuleAddr  string
	PackageAddr string
	Version     *version.Version
	LocalPath   string
}

func (h *testInstallHooks) Download(moduleAddr, packageAddr string, version *version.Version) {
	h.Calls = append(h.Calls, testInstallHookCall{
		Name:        "Download",
		ModuleAddr:  moduleAddr,
		PackageAddr: packageAddr,
		Version:     version,
	})
}

func (h *testInstallHooks) Install(moduleAddr string, version *version.Version, localPath string) {
	h.Calls = append(h.Calls, testInstallHookCall{
		Name:       "Install",
		ModuleAddr: moduleAddr,
		Version:    version,
		LocalPath:  localPath,
	})
}

// tempChdir copies the contents of the given directory to a temporary
// directory and changes the test process's current working directory to
// point to that directory. Also returned is a function that should be
// called at the end of the test (e.g. via "defer") to restore the previous
// working directory.
//
// Tests using this helper cannot safely be run in parallel with other tests.
func tempChdir(t *testing.T, sourceDir string) (string, func()) {
	t.Helper()

	tmpDir, err := ioutil.TempDir("", "terraform-configload")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
		return "", nil
	}

	if err := copy.CopyDir(tmpDir, sourceDir); err != nil {
		t.Fatalf("failed to copy fixture to temporary directory: %s", err)
		return "", nil
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to determine current working directory: %s", err)
		return "", nil
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("failed to switch to temp dir %s: %s", tmpDir, err)
		return "", nil
	}

	// Most of the tests need this, so we'll make it just in case.
	os.MkdirAll(filepath.Join(tmpDir, ".terraform/modules"), os.ModePerm)

	t.Logf("tempChdir switched to %s after copying from %s", tmpDir, sourceDir)

	return tmpDir, func() {
		err := os.Chdir(oldDir)
		if err != nil {
			panic(fmt.Errorf("failed to restore previous working directory %s: %s", oldDir, err))
		}

		if os.Getenv("TF_CONFIGLOAD_TEST_KEEP_TMP") == "" {
			os.RemoveAll(tmpDir)
		}
	}
}

func assertNoDiagnostics(t *testing.T, diags tfdiags.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags tfdiags.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != 0 {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %#v", diag)
		}
		return true
	}
	return false
}

func assertDiagnosticSummary(t *testing.T, diags tfdiags.Diagnostics, want string) bool {
	t.Helper()

	for _, diag := range diags {
		if diag.Description().Summary == want {
			return false
		}
	}

	t.Errorf("missing diagnostic summary %q", want)
	for _, diag := range diags {
		t.Logf("- %#v", diag)
	}
	return true
}

func assertResultDeepEqual(t *testing.T, got, want interface{}) bool {
	t.Helper()
	if diff := deep.Equal(got, want); diff != nil {
		for _, problem := range diff {
			t.Errorf("%s", problem)
		}
		return true
	}
	return false
}
