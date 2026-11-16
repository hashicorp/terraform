// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestModule_backend_overrides_a_backend(t *testing.T) {
	t.Run("it can override a backend block with a different backend block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-backend")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		gotType := mod.Backend.Type
		wantType := "bar"

		if gotType != wantType {
			t.Errorf("wrong result for backend type: got %#v, want %#v\n", gotType, wantType)
		}

		attrs, _ := mod.Backend.Config.JustAttributes()

		gotAttr, diags := attrs["path"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		wantAttr := cty.StringVal("CHANGED/relative/path/to/terraform.tfstate")

		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for backend 'path': got %#v, want %#v\n", gotAttr, wantAttr)
		}
	})
}

// Unlike most other overrides, backend blocks do not require a base configuration in a primary
// configuration file, as an omitted backend there implies the local backend.
func TestModule_backend_overrides_no_base(t *testing.T) {
	t.Run("it can introduce a backend block via overrides when the base config has has no cloud or backend blocks", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-backend-no-base")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.Backend == nil {
			t.Errorf("expected module Backend not to be nil")
		}
	})
}

func TestModule_cloud_overrides_a_backend(t *testing.T) {
	t.Run("it can override a backend block with a cloud block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-backend-with-cloud")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.Backend != nil {
			t.Errorf("expected module Backend to be nil")
		}

		if mod.CloudConfig == nil {
			t.Errorf("expected module CloudConfig not to be nil")
		}
	})
}

func TestModule_cloud_overrides_cloud(t *testing.T) {
	t.Run("it can override a cloud block with a different cloud block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-cloud")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		attrs, _ := mod.CloudConfig.Config.JustAttributes()

		gotAttr, diags := attrs["organization"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		wantAttr := cty.StringVal("CHANGED")

		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for Cloud 'organization': got %#v, want %#v\n", gotAttr, wantAttr)
		}

		// The override should have completely replaced the cloud block in the primary file, no merging
		if attrs["should_not_be_present_with_override"] != nil {
			t.Errorf("expected 'should_not_be_present_with_override' attribute to be nil")
		}
	})
}

// Unlike most other overrides, cloud blocks do not require a base configuration in a primary
// configuration file, as an omitted backend there implies the local backend and cloud blocks
// override backends.
func TestModule_cloud_overrides_no_base(t *testing.T) {
	t.Run("it can introduce a cloud block via overrides when the base config has no cloud or backend blocks", func(t *testing.T) {

		mod, diags := testModuleFromDir("testdata/valid-modules/override-cloud-no-base")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.CloudConfig == nil {
			t.Errorf("expected module CloudConfig not to be nil")
		}
	})
}

func TestModule_backend_overrides_cloud(t *testing.T) {
	t.Run("it can override a cloud block with a backend block", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-cloud-with-backend")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		gotType := mod.Backend.Type
		wantType := "override"

		if gotType != wantType {
			t.Errorf("wrong result for backend type: got %#v, want %#v\n", gotType, wantType)
		}

		attrs, _ := mod.Backend.Config.JustAttributes()

		gotAttr, diags := attrs["path"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		wantAttr := cty.StringVal("value from override")

		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for backend 'path': got %#v, want %#v\n", gotAttr, wantAttr)
		}
	})
}

func TestModule_cloud_duplicate_overrides(t *testing.T) {
	t.Run("it raises an error when a override file contains multiple cloud blocks", func(t *testing.T) {
		_, diags := testModuleFromDir("testdata/invalid-modules/override-cloud-duplicates")
		want := `Duplicate HCP Terraform configurations`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected module error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

func TestModule_backend_multiple(t *testing.T) {
	t.Run("it detects when two backend blocks are present within the same module in separate files", func(t *testing.T) {
		_, diags := testModuleFromDir("testdata/invalid-modules/multiple-backends")
		if !diags.HasErrors() {
			t.Fatal("module should have error diags, but does not")
		}

		want := `Duplicate 'backend' configuration block`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

func TestModule_cloud_multiple(t *testing.T) {
	t.Run("it detects when two cloud blocks are present within the same module in separate files", func(t *testing.T) {

		_, diags := testModuleFromDir("testdata/invalid-modules/multiple-cloud")
		if !diags.HasErrors() {
			t.Fatal("module should have error diags, but does not")
		}

		want := `Duplicate HCP Terraform configurations`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
		}
	})
}

// Cannot combine use of backend, cloud, state_store blocks.
func TestModule_conflicting_backend_cloud_stateStore(t *testing.T) {
	testCases := map[string]struct {
		dir              string
		wantMsg          string
		allowExperiments bool
	}{
		"cloud backends conflict": {
			// detects when both cloud and backend blocks are in the same terraform block
			dir:     "testdata/invalid-modules/conflict-cloud-backend",
			wantMsg: `Conflicting 'cloud' and 'backend' configuration blocks are present`,
		},
		"cloud backends conflict separate": {
			// it detects when both cloud and backend blocks are present in the same module in separate files
			dir:     "testdata/invalid-modules/conflict-cloud-backend-separate-files",
			wantMsg: `Conflicting 'cloud' and 'backend' configuration blocks are present`,
		},
		"cloud state store conflict": {
			// detects when both cloud and state_store blocks are in the same terraform block
			dir:              "testdata/invalid-modules/conflict-cloud-statestore",
			wantMsg:          `Conflicting 'cloud' and 'state_store' configuration blocks are present`,
			allowExperiments: true,
		},
		"cloud state store conflict separate": {
			// it detects when both cloud and state_store blocks are present in the same module in separate files
			dir:              "testdata/invalid-modules/conflict-cloud-statestore-separate-files",
			wantMsg:          `Conflicting 'cloud' and 'state_store' configuration blocks are present`,
			allowExperiments: true,
		},
		"state store backend conflict": {
			// it detects when both state_store and backend blocks are in the same terraform block
			dir:              "testdata/invalid-modules/conflict-statestore-backend",
			wantMsg:          `Conflicting 'state_store' and 'backend' configuration blocks are present`,
			allowExperiments: true,
		},
		"state store backend conflict separate": {
			// it detects when both state_store and backend blocks are present in the same module in separate files
			dir:              "testdata/invalid-modules/conflict-statestore-backend-separate-files",
			wantMsg:          `Conflicting 'state_store' and 'backend' configuration blocks are present`,
			allowExperiments: true,
		},
		"cloud backend state store conflict": {
			// it detects all 3 of cloud, state_storage and backend blocks are in the same terraform block
			dir:              "testdata/invalid-modules/conflict-cloud-backend-statestore",
			wantMsg:          `Only one of 'cloud', 'state_store', or 'backend' configuration blocks are allowed`,
			allowExperiments: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.dir, func(t *testing.T) {
			var diags hcl.Diagnostics
			if tc.allowExperiments {
				// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
				_, diags = testModuleFromDirWithExperiments(tc.dir)
			} else {
				_, diags = testModuleFromDir(tc.dir)
			}
			if !diags.HasErrors() {
				t.Fatal("module should have error diags, but does not")
			}

			if got := diags.Error(); !strings.Contains(got, tc.wantMsg) {
				t.Fatalf("expected error to contain %q\nerror was:\n%s", tc.wantMsg, got)
			}
		})
	}
}

func TestModule_stateStore_overrides_stateStore(t *testing.T) {
	t.Run("it can override a state_store block with a different state_store block", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-state-store")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.StateStore == nil {
			t.Fatal("expected parsed module to include a state store, found none")
		}

		// Check type override
		gotType := mod.StateStore.Type
		wantType := "foo_override"
		if gotType != wantType {
			t.Errorf("wrong result for state_store type: got %#v, want %#v\n", gotType, wantType)
		}

		// Check custom attribute override
		attrs, _ := mod.StateStore.Config.JustAttributes()
		gotAttr, diags := attrs["custom_attr"].Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}
		wantAttr := cty.StringVal("override")
		if !gotAttr.RawEquals(wantAttr) {
			t.Errorf("wrong result for state_store 'custom_attr': got %#v, want %#v\n", gotAttr, wantAttr)
		}

		// Check provider reference override
		wantLocalName := "bar"
		if mod.StateStore.Provider.Name != wantLocalName {
			t.Errorf("wrong result for state_store 'provider' value's local name: got %#v, want %#v\n", mod.StateStore.Provider.Name, wantLocalName)
		}
	})
}

// Unlike most other overrides, state_store blocks do not require a base configuration in a primary
// configuration file, as an omitted backend there implies the local backend.
func TestModule_stateStore_override_no_base(t *testing.T) {
	t.Run("it can introduce a state_store block via overrides when the base config has has no cloud, backend, or state_store blocks", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-state-store-no-base")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		if mod.StateStore == nil {
			t.Errorf("expected module StateStore not to be nil")
		}
	})
}

func TestModule_stateStore_overrides_backend(t *testing.T) {
	t.Run("it can override a backend block with a state_store block", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-backend-with-state-store")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		// Backend not set
		if mod.Backend != nil {
			t.Errorf("backend should not be set: got %#v\n", mod.Backend)
		}

		// Check state_store
		if mod.StateStore == nil {
			t.Fatal("expected parsed module to include a state store, found none")
		}

		gotType := mod.StateStore.Type
		wantType := "foo_override"
		if gotType != wantType {
			t.Errorf("wrong result for state_store type: got %#v, want %#v\n", gotType, wantType)
		}

		// Not necessary to assert all values in state_store
	})
}

func TestModule_stateStore_overrides_cloud(t *testing.T) {
	t.Run("it can override a cloud block with a state_store block", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-cloud-with-state-store")
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}

		// CloudConfig not set
		if mod.CloudConfig != nil {
			t.Errorf("backend should not be set: got %#v\n", mod.Backend)
		}

		// Check state_store
		if mod.StateStore == nil {
			t.Fatal("expected parsed module to include a state store, found none")
		}
		gotType := mod.StateStore.Type
		wantType := "foo_override"
		if gotType != wantType {
			t.Errorf("wrong result for state_store type: got %#v, want %#v\n", gotType, wantType)
		}

		// Not necessary to assert all values in state_store
	})
}

func TestModule_state_store_multiple(t *testing.T) {
	t.Run("it detects when two state_store blocks are present within the same module in separate files", func(t *testing.T) {
		// TODO(SarahFrench/radeksimko) - disable experiments in this test once the feature is GA.
		_, diags := testModuleFromDirWithExperiments("testdata/invalid-modules/multiple-state-store")
		if !diags.HasErrors() {
			t.Fatal("module should have error diags, but does not")
		}

		want := `Duplicate 'state_store' configuration block`
		if got := diags.Error(); !strings.Contains(got, want) {
			t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
		}
	})
}
