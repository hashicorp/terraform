// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/spf13/afero"
)

// childModuleFile matches the descendant module configuration files in the
// plan-modules-race fixture (e.g. "mod3/main.tf"), so the delaying filesystem
// below only slows down the parts of config loading we care about.
var childModuleFile = regexp.MustCompile(`(^|/)mod\d+/`)

// delayingFS is an afero.Fs that artificially slows down reads of descendant
// module configuration files. This widens the window during which the config
// loader's shared parser is being written by the config-loading graph walk,
// so that an interrupt can land while modules are still being loaded.
//
// After deadline has passed it stops delaying, allowing the abandoned config
// walk to drain promptly once the operation has been cancelled.
type delayingFS struct {
	afero.Fs
	delay    time.Duration
	deadline time.Time
}

func (d *delayingFS) Open(name string) (afero.File, error) {
	if childModuleFile.MatchString(filepath.ToSlash(name)) && time.Now().Before(d.deadline) {
		time.Sleep(d.delay)
	}
	return d.Fs.Open(name)
}

// TestPlan_configLoaderRace is a regression test for a data race in the
// configuration loader's shared parser: https://github.com/hashicorp/terraform/issues/38725
//
// A plan loads descendant modules by walking the init graph, which parses each
// module into the loader's shared *configs.Parser (the write side). When the
// run is interrupted (Ctrl-C), Meta.RunOperation stops waiting for the
// operation and the command renders the resulting diagnostics, which reads the
// same parser's source cache via Loader.Sources (the read side).
//
// To make the interrupt land while modules are still loading, the loader is
// given a filesystem that delays reads of module files, and two interrupts are
// delivered on the ShutdownCh to force the cancel path (which returns after a
// timeout without waiting for the still-running config walk).
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

	// The loader the plan command will actually use reads through a filesystem
	// that delays module file reads, so the config walk is still in progress
	// when we interrupt below.
	fs := &delayingFS{
		Fs:       afero.NewOsFs(),
		delay:    1 * time.Second,
		deadline: time.Now().Add(7 * time.Second),
	}
	testLoader, err := configload.NewLoader(&configload.Config{
		ModulesDir: modulesDir,
		OverrideFS: fs,
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

	// Simulate the user pressing Ctrl-C twice while modules are still loading.
	// The first interrupt asks the operation to stop gracefully; the second
	// forces a cancel, after which RunOperation stops waiting for the operation
	// (which is blocked in the delayed config walk) and the command proceeds to
	// render diagnostics.
	go func() {
		time.Sleep(1 * time.Second)
		close(shutdownCh)
	}()

	c.Run([]string{})

	// Allow the abandoned config walk to drain before the test returns, so it
	// doesn't keep writing the parser after the test completes.
	time.Sleep(time.Until(fs.deadline) + 1*time.Second)
}
