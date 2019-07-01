package initwd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestDirFromModule_registry(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("this test accesses registry.terraform.io and github.com; set TF_ACC=1 to run it")
	}

	fixtureDir := filepath.Clean("testdata/empty")
	dir, done := tempChdir(t, fixtureDir)
	modsDir := filepath.Join(dir, ".terraform/modules")
	defer done()

	hooks := &testInstallHooks{}

	reg := registry.NewClient(nil, nil)
	diags := DirFromModule(dir, modsDir, "hashicorp/module-installer-acctest/aws//examples/main", reg, hooks)
	assertNoDiagnostics(t, diags)

	v := version.Must(version.NewVersion("0.0.1"))

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
			LocalPath:  filepath.Join(dir, ".terraform/modules/root/hashicorp-terraform-aws-module-installer-acctest-5e87aff"),
		},
		{
			Name:       "Install",
			ModuleAddr: "root.child_a",
			LocalPath:  filepath.Join(dir, ".terraform/modules/root/hashicorp-terraform-aws-module-installer-acctest-5e87aff/modules/child_a"),
		},
		{
			Name:       "Install",
			ModuleAddr: "root.child_a.child_b",
			LocalPath:  filepath.Join(dir, ".terraform/modules/root/hashicorp-terraform-aws-module-installer-acctest-5e87aff/modules/child_b"),
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
