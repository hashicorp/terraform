package initwd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-test/deep"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}

func TestModuleInstaller(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/local-modules")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(".", false, hooks)
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
	_, diags := inst.InstallModules(".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		assertDiagnosticSummary(t, diags, "Module not found")
	}
}

func TestModuleInstaller_invalid_version_constraint_error(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/invalid-version-constraint")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		assertDiagnosticSummary(t, diags, "Invalid version constraint")
	}
}

func TestModuleInstaller_invalidVersionConstraintGetter(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/invalid-version-constraint")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		assertDiagnosticSummary(t, diags, "Invalid version constraint")
	}
}

func TestModuleInstaller_invalidVersionConstraintLocal(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/invalid-version-constraint-local")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(".", false, hooks)

	if !diags.HasErrors() {
		t.Fatal("expected error")
	} else {
		assertDiagnosticSummary(t, diags, "Invalid version constraint")
	}
}

func TestModuleInstaller_symlink(t *testing.T) {
	fixtureDir := filepath.Clean("testdata/local-module-symlink")
	dir, done := tempChdir(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, nil)
	_, diags := inst.InstallModules(".", false, hooks)
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
	dir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Error(err)
	}

	defer done()

	hooks := &testInstallHooks{}
	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, registry.NewClient(nil, nil))
	_, diags := inst.InstallModules(dir, false, hooks)
	assertNoDiagnostics(t, diags)

	v := version.Must(version.NewVersion("0.0.1"))

	wantCalls := []testInstallHookCall{
		// the configuration builder visits each level of calls in lexicographical
		// order by name, so the following list is kept in the same order.

		// acctest_child_a accesses //modules/child_a directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_a",
			PackageAddr: "hashicorp/module-installer-acctest/aws", // intentionally excludes the subdir because we're downloading the whole package here
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a",
			Version:    v,
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_a/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_a"),
		},

		// acctest_child_a.child_b
		// (no download because it's a relative path inside acctest_child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_a/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_b"),
		},

		// acctest_child_b accesses //modules/child_b directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_b",
			PackageAddr: "hashicorp/module-installer-acctest/aws", // intentionally excludes the subdir because we're downloading the whole package here
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_b",
			Version:    v,
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_child_b/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_b"),
		},

		// acctest_root
		{
			Name:        "Download",
			ModuleAddr:  "acctest_root",
			PackageAddr: "hashicorp/module-installer-acctest/aws",
			Version:     v,
		},
		{
			Name:       "Install",
			ModuleAddr: "acctest_root",
			Version:    v,
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/hashicorp-terraform-aws-module-installer-acctest-853d038"),
		},

		// acctest_root.child_a
		// (no download because it's a relative path inside acctest_root)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_a"),
		},

		// acctest_root.child_a.child_b
		// (no download because it's a relative path inside acctest_root, via acctest_root.child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/acctest_root/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_b"),
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
	dir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Error(err)
	}
	defer done()

	hooks := &testInstallHooks{}
	modulesDir := filepath.Join(dir, ".terraform/modules")
	inst := NewModuleInstaller(modulesDir, registry.NewClient(nil, nil))
	_, diags := inst.InstallModules(dir, false, hooks)
	assertNoDiagnostics(t, diags)

	wantCalls := []testInstallHookCall{
		// the configuration builder visits each level of calls in lexicographical
		// order by name, so the following list is kept in the same order.

		// acctest_child_a accesses //modules/child_a directly
		{
			Name:        "Download",
			ModuleAddr:  "acctest_child_a",
			PackageAddr: "github.com/hashicorp/terraform-aws-module-installer-acctest?ref=v0.0.1", // intentionally excludes the subdir because we're downloading the whole repo here
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
			PackageAddr: "github.com/hashicorp/terraform-aws-module-installer-acctest?ref=v0.0.1", // intentionally excludes the subdir because we're downloading the whole package here
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
			PackageAddr: "github.com/hashicorp/terraform-aws-module-installer-acctest?ref=v0.0.1",
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

	if err := copyDir(tmpDir, sourceDir); err != nil {
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
			t.Logf("- %s", diag)
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
		t.Logf("- %s", diag)
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
