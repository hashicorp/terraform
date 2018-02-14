package configload

import (
	"os"
	"path/filepath"
	"testing"

	version "github.com/hashicorp/go-version"
)

func TestLoaderInstallModules_local(t *testing.T) {
	fixtureDir := filepath.Clean("test-fixtures/local-modules")
	loader, done := tempChdirLoader(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	diags := loader.InstallModules(".", false, hooks)
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

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	_, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, loadDiags)
}

func TestLoaderInstallModules_registry(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("this test accesses registry.terraform.io and github.com; set TF_ACC=1 to run it")
	}

	fixtureDir := filepath.Clean("test-fixtures/registry-modules")
	loader, done := tempChdirLoader(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	diags := loader.InstallModules(".", false, hooks)
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
			LocalPath:  ".terraform/modules/acctest_child_a/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_a",
		},

		// acctest_child_a.child_b
		// (no download because it's a relative path inside acctest_child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a.child_b",
			LocalPath:  ".terraform/modules/acctest_child_a/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_b",
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
			LocalPath:  ".terraform/modules/acctest_child_b/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_b",
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
			LocalPath:  ".terraform/modules/acctest_root/hashicorp-terraform-aws-module-installer-acctest-853d038",
		},

		// acctest_root.child_a
		// (no download because it's a relative path inside acctest_root)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a",
			LocalPath:  ".terraform/modules/acctest_root/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_a",
		},

		// acctest_root.child_a.child_b
		// (no download because it's a relative path inside acctest_root, via acctest_root.child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a.child_b",
			LocalPath:  ".terraform/modules/acctest_root/hashicorp-terraform-aws-module-installer-acctest-853d038/modules/child_b",
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	_, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, loadDiags)
}

func TestLoaderInstallModules_goGetter(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("this test accesses github.com; set TF_ACC=1 to run it")
	}

	fixtureDir := filepath.Clean("test-fixtures/go-getter-modules")
	loader, done := tempChdirLoader(t, fixtureDir)
	defer done()

	hooks := &testInstallHooks{}

	diags := loader.InstallModules(".", false, hooks)
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
			LocalPath:  ".terraform/modules/acctest_child_a/modules/child_a",
		},

		// acctest_child_a.child_b
		// (no download because it's a relative path inside acctest_child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_child_a.child_b",
			LocalPath:  ".terraform/modules/acctest_child_a/modules/child_b",
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
			LocalPath:  ".terraform/modules/acctest_child_b/modules/child_b",
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
			LocalPath:  ".terraform/modules/acctest_root",
		},

		// acctest_root.child_a
		// (no download because it's a relative path inside acctest_root)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a",
			LocalPath:  ".terraform/modules/acctest_root/modules/child_a",
		},

		// acctest_root.child_a.child_b
		// (no download because it's a relative path inside acctest_root, via acctest_root.child_a)
		{
			Name:       "Install",
			ModuleAddr: "acctest_root.child_a.child_b",
			LocalPath:  ".terraform/modules/acctest_root/modules/child_b",
		},
	}

	if assertResultDeepEqual(t, hooks.Calls, wantCalls) {
		return
	}

	// Make sure the configuration is loadable now.
	// (This ensures that correct information is recorded in the manifest.)
	_, loadDiags := loader.LoadConfig(".")
	assertNoDiagnostics(t, loadDiags)
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
