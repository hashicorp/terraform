// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package planfile

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestRoundtrip(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "test-config")
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: filepath.Join(fixtureDir, ".terraform", "modules"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, snapIn, diags := loader.LoadConfigWithSnapshot(fixtureDir)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	// Just a minimal state file so we can test that it comes out again at all.
	// We don't need to test the entire thing because the state file
	// serialization is already tested in its own package.
	stateFileIn := &statefile.File{
		TerraformVersion: tfversion.SemVer,
		Serial:           2,
		Lineage:          "abc123",
		State:            states.NewState(),
	}
	prevStateFileIn := &statefile.File{
		TerraformVersion: tfversion.SemVer,
		Serial:           1,
		Lineage:          "abc123",
		State:            states.NewState(),
	}

	// Minimal plan too, since the serialization of the tfplan portion of the
	// file is tested more fully in tfplan_test.go .
	planIn := &plans.Plan{
		Changes: &plans.ChangesSrc{
			Resources: []*plans.ResourceInstanceChangeSrc{},
			Outputs:   []*plans.OutputChangeSrc{},
		},
		DriftedResources:  []*plans.ResourceInstanceChangeSrc{},
		DeferredResources: []*plans.DeferredResourceInstanceChangeSrc{},
		VariableValues: map[string]plans.DynamicValue{
			"foo": plans.DynamicValue([]byte("foo placeholder")),
		},
		Backend: plans.Backend{
			Type:      "local",
			Config:    plans.DynamicValue([]byte("config placeholder")),
			Workspace: "default",
		},
		Checks: &states.CheckResults{},

		// Due to some historical oddities in how we've changed modelling over
		// time, we also include the states (without the corresponding file
		// headers) in the plans.Plan object. This is currently ignored by
		// Create but will be returned by ReadPlan and so we need to include
		// it here so that we'll get a match when we compare input and output
		// below.
		PrevRunState: prevStateFileIn.State,
		PriorState:   stateFileIn.State,
	}

	locksIn := depsfile.NewLocks()
	locksIn.SetProvider(
		addrs.NewDefaultProvider("boop"),
		providerreqs.MustParseVersion("1.0.0"),
		providerreqs.MustParseVersionConstraints(">= 1.0.0"),
		[]providerreqs.Hash{
			providerreqs.MustParseHash("fake:hello"),
		},
	)

	planFn := filepath.Join(t.TempDir(), "tfplan")

	err = Create(planFn, CreateArgs{
		ConfigSnapshot:       snapIn,
		PreviousRunStateFile: prevStateFileIn,
		StateFile:            stateFileIn,
		Plan:                 planIn,
		DependencyLocks:      locksIn,
	})
	if err != nil {
		t.Fatalf("failed to create plan file: %s", err)
	}

	wpf, err := OpenWrapped(planFn)
	if err != nil {
		t.Fatalf("failed to open plan file for reading: %s", err)
	}
	pr, ok := wpf.Local()
	if !ok {
		t.Fatalf("failed to open plan file as a local plan file")
	}
	if wpf.IsCloud() {
		t.Fatalf("wrapped plan claims to be both kinds of plan at once")
	}

	t.Run("ReadPlan", func(t *testing.T) {
		planOut, err := pr.ReadPlan()
		if err != nil {
			t.Fatalf("failed to read plan: %s", err)
		}
		if diff := cmp.Diff(planIn, planOut, collections.CmpOptions); diff != "" {
			t.Errorf("plan did not survive round-trip\n%s", diff)
		}
	})

	t.Run("ReadStateFile", func(t *testing.T) {
		stateFileOut, err := pr.ReadStateFile()
		if err != nil {
			t.Fatalf("failed to read state: %s", err)
		}
		if diff := cmp.Diff(stateFileIn, stateFileOut); diff != "" {
			t.Errorf("state file did not survive round-trip\n%s", diff)
		}
	})

	t.Run("ReadPrevStateFile", func(t *testing.T) {
		prevStateFileOut, err := pr.ReadPrevStateFile()
		if err != nil {
			t.Fatalf("failed to read state: %s", err)
		}
		if diff := cmp.Diff(prevStateFileIn, prevStateFileOut); diff != "" {
			t.Errorf("state file did not survive round-trip\n%s", diff)
		}
	})

	t.Run("ReadConfigSnapshot", func(t *testing.T) {
		snapOut, err := pr.ReadConfigSnapshot()
		if err != nil {
			t.Fatalf("failed to read config snapshot: %s", err)
		}
		if diff := cmp.Diff(snapIn, snapOut); diff != "" {
			t.Errorf("config snapshot did not survive round-trip\n%s", diff)
		}
	})

	t.Run("ReadConfig", func(t *testing.T) {
		// Reading from snapshots is tested in the configload package, so
		// here we'll just test that we can successfully do it, to see if the
		// glue code in _this_ package is correct.
		_, diags := pr.ReadConfig()
		if diags.HasErrors() {
			t.Errorf("when reading config: %s", diags.Err())
		}
	})

	t.Run("ReadDependencyLocks", func(t *testing.T) {
		locksOut, diags := pr.ReadDependencyLocks()
		if diags.HasErrors() {
			t.Fatalf("when reading config: %s", diags.Err())
		}
		got := locksOut.AllProviders()
		want := locksIn.AllProviders()
		if diff := cmp.Diff(want, got, cmp.AllowUnexported(depsfile.ProviderLock{})); diff != "" {
			t.Errorf("provider locks did not survive round-trip\n%s", diff)
		}
	})
}

func TestWrappedError(t *testing.T) {
	// Open something that isn't a cloud or local planfile: should error
	wrongFile := "not a valid zip file"
	_, err := OpenWrapped(filepath.Join("testdata", "test-config", "root.tf"))
	if !strings.Contains(err.Error(), wrongFile) {
		t.Fatalf("expected  %q, got %q", wrongFile, err)
	}

	// Open something that doesn't exist: should error
	missingFile := "no such file or directory"
	_, err = OpenWrapped(filepath.Join("testdata", "absent.tfplan"))
	if !strings.Contains(err.Error(), missingFile) {
		t.Fatalf("expected  %q, got %q", missingFile, err)
	}
}

func TestWrappedCloud(t *testing.T) {
	// Loading valid cloud plan results in a wrapped cloud plan
	wpf, err := OpenWrapped(filepath.Join("testdata", "cloudplan.json"))
	if err != nil {
		t.Fatalf("failed to open valid cloud plan: %s", err)
	}
	if !wpf.IsCloud() {
		t.Fatalf("failed to open cloud file as a cloud plan")
	}
	if wpf.IsLocal() {
		t.Fatalf("wrapped plan claims to be both kinds of plan at once")
	}
}
