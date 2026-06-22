// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/spf13/afero"
)

// moduleDir matches a descendant module's directory in the "plan-modules-race" fixture (e.g. ".../mod0")
var moduleDir = regexp.MustCompile(`(^|/)mod\d+$`)

type mockFS struct {
	afero.Fs
	loadingStarted chan struct{}
	proceed        chan struct{}
	once           sync.Once
}

func (g *mockFS) Open(name string) (afero.File, error) {
	if moduleDir.MatchString(filepath.ToSlash(name)) {
		// Indicate the loading of modules has started, so we can trigger the shutdown channel
		g.once.Do(func() { close(g.loadingStarted) })
		<-g.proceed
	}
	return g.Fs.Open(name)
}

// TestPlan_configLoaderRace is a regression test for a data race in the
// configuration loader's shared parser: https://github.com/hashicorp/terraform/issues/38725
//
// A plan loads descendant modules by walking the init graph, which parses each
// module into the loader's shared *configs.Parser (the write side). When the
// run is interrupted via the shutdown channel, (*Meta).RunOperation stops waiting for the
// operation and the command renders the resulting diagnostics, which reads the
// same parser's source cache via Loader.Sources (the read side).
func TestPlan_configLoaderRace(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-modules-race"), td)
	t.Chdir(td)

	modulesDir := t.TempDir()

	// Install the local modules with a fresh loader so the module manifest
	// exists on disk before the plan runs.
	installLoader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
	})
	if err != nil {
		t.Fatalf("failed to create install loader: %s", err)
	}
	inst := initwd.NewModuleInstaller(modulesDir, installLoader, registry.NewClient(nil, nil), nil)
	if _, instDiags := inst.InstallModules(context.Background(), ".", "tests", true, false); instDiags.HasErrors() {
		t.Fatalf("failed to install modules: %s", instDiags.Err())
	}

	mockFS := &mockFS{
		Fs:             afero.NewOsFs(),
		loadingStarted: make(chan struct{}),
		proceed:        make(chan struct{}),
	}
	testLoader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
		OverrideFS: mockFS,
	})
	if err != nil {
		t.Fatalf("failed to create test loader: %s", err)
	}
	if err := testLoader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules: %s", err)
	}

	view, done := testView(t)
	defer done(t)
	// Wire the diagnostics renderer to read this loader's sources,
	// mirroring what Meta.initConfigLoader does for a non-injected loader.
	view.SetConfigSources(testLoader.Sources)

	shutdownCh := make(chan struct{})
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(planFixtureProvider()),
			View:             view,
			configLoader:     testLoader,
			ShutdownCh:       shutdownCh,
		},
	}

	go func() {
		// Wait until the modules begin loading
		<-mockFS.loadingStarted

		// Cancel the operation, which after 5 seconds will trigger the read side of
		// the race condition (via the diagnostic renderer).
		close(shutdownCh)

		// Allow one module load to proceed, which will trigger the write side of
		// the race condition (via the module being stored in the shared parser).
		//
		// The next module load will block until the shutdown has completed/timed
		// out (which is where the race condition would occur).
		mockFS.proceed <- struct{}{}
	}()

	c.Run([]string{})

	// Now that the run command has timed out, allow the remaining modules to proceed
	close(mockFS.proceed)
}
