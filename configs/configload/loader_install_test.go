package configload

import (
	"path/filepath"
	"testing"

	version "github.com/hashicorp/go-version"
)

func TestLoaderInstallModules_local(t *testing.T) {
	fixtureDir := filepath.Clean("test-fixtures/local-modules")
	loader := newTestLoader(filepath.Join(fixtureDir, ".terraform/modules"))

	hooks := &testInstallHooks{}

	diags := loader.InstallModules(fixtureDir, false, hooks)
	assertNoDiagnostics(t, diags)

	wantCalls := []testInstallHookCall{
		{
			Name:        "Install",
			ModuleAddr:  "child_a",
			PackageAddr: "",
			LocalPath:   "test-fixtures/local-modules/child_a",
		},
		{
			Name:        "Install",
			ModuleAddr:  "child_a.child_b",
			PackageAddr: "",
			LocalPath:   "test-fixtures/local-modules/child_a/child_b",
		},
	}

	assertResultDeepEqual(t, hooks.Calls, wantCalls)
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
