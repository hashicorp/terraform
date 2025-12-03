// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"sort"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestLoadConfigDirErrors(t *testing.T) {
	bundle, err := sourcebundle.OpenDir("testdata/basics-bundle")
	if err != nil {
		t.Fatal(err)
	}

	rootAddr := sourceaddrs.MustParseSource("git::https://example.com/errored.git").(sourceaddrs.RemoteSource)
	_, gotDiags := LoadConfigDir(rootAddr, bundle)

	sort.SliceStable(gotDiags, func(i, j int) bool {
		if gotDiags[i].Severity() != gotDiags[j].Severity() {
			return gotDiags[i].Severity() < gotDiags[j].Severity()
		}

		if gotDiags[i].Description().Summary != gotDiags[j].Description().Summary {
			return gotDiags[i].Description().Summary < gotDiags[j].Description().Summary
		}

		return gotDiags[i].Description().Detail < gotDiags[j].Description().Detail
	})

	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(tfdiags.Error, "Component exists for removed block", "A removed block for component \"a\" was declared without an index, but a component block with the same name was declared at git::https://example.com/errored.git//main.tfcomponent.hcl:10,1-14.\n\nA removed block without an index indicates that the component and all instances were removed from the configuration, and this is not the case."),
		tfdiags.Sourceless(tfdiags.Error, "Invalid for_each expression", "A removed block with a for_each expression must reference that expression within the `from` attribute."),
		tfdiags.Sourceless(tfdiags.Error, "Invalid for_each expression", "A removed block with a for_each expression must reference that expression within the `from` attribute."),
	}

	count := len(wantDiags)
	if len(gotDiags) > count {
		count = len(gotDiags)
	}

	for i := 0; i < count; i++ {
		if i >= len(wantDiags) {
			t.Errorf("unexpected diagnostic:\n%s", gotDiags[i])
			continue
		}

		if i >= len(gotDiags) {
			t.Errorf("missing diagnostic:\n%s", wantDiags[i])
			continue
		}

		got, want := gotDiags[i], wantDiags[i]

		if got, want := got.Severity(), want.Severity(); got != want {
			t.Errorf("diagnostics[%d] severity\ngot:  %s\nwant: %s", i, got, want)
		}

		if got, want := got.Description().Summary, want.Description().Summary; got != want {
			t.Errorf("diagnostics[%d] summary\ngot:  %s\nwant: %s", i, got, want)
		}

		if got, want := got.Description().Detail, want.Description().Detail; got != want {
			t.Errorf("diagnostics[%d] detail\ngot:  %s\nwant: %s", i, got, want)
		}
	}
}

func TestLoadConfigDirSourceErrors(t *testing.T) {
	bundle, err := sourcebundle.OpenDir("testdata/basics-bundle")
	if err != nil {
		t.Fatal(err)
	}

	rootAddr := sourceaddrs.MustParseSource("git::https://example.com/errored-sources.git").(sourceaddrs.RemoteSource)
	_, gotDiags := LoadConfigDir(rootAddr, bundle)

	sort.SliceStable(gotDiags, func(i, j int) bool {
		if gotDiags[i].Severity() != gotDiags[j].Severity() {
			return gotDiags[i].Severity() < gotDiags[j].Severity()
		}

		if gotDiags[i].Description().Summary != gotDiags[j].Description().Summary {
			return gotDiags[i].Description().Summary < gotDiags[j].Description().Summary
		}

		return gotDiags[i].Description().Detail < gotDiags[j].Description().Detail
	})

	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(tfdiags.Error, "Invalid removed block", "The linked removed block was not executed because the `from` attribute of the removed block targets a component or embedded stack within an orphaned embedded stack.\n\nIn order to remove an entire stack, update your removed block to target the entire removed stack itself instead of the specific elements within it."),
		tfdiags.Sourceless(tfdiags.Error, "Invalid source address", "Cannot use \"git::https://example.com/errored-sources.git\" as a source address here: the target stack is already initialised with another source \"git::https://example.com/errored-sources.git//subdir\"."),
		tfdiags.Sourceless(tfdiags.Error, "Invalid source address", "Cannot use \"git::https://example.com/errored-sources.git//subdir\" as a source address here: the target stack is already initialised with another source \"git::https://example.com/errored-sources.git\"."),
	}

	count := len(wantDiags)
	if len(gotDiags) > count {
		count = len(gotDiags)
	}

	for i := 0; i < count; i++ {
		if i >= len(wantDiags) {
			t.Errorf("unexpected diagnostic:\n%s", gotDiags[i])
			continue
		}

		if i >= len(gotDiags) {
			t.Errorf("missing diagnostic:\n%s", wantDiags[i])
			continue
		}

		got, want := gotDiags[i], wantDiags[i]

		if got, want := got.Severity(), want.Severity(); got != want {
			t.Errorf("diagnostics[%d] severity\ngot:  %s\nwant: %s", i, got, want)
		}

		if got, want := got.Description().Summary, want.Description().Summary; got != want {
			t.Errorf("diagnostics[%d] summary\ngot:  %s\nwant: %s", i, got, want)
		}

		if got, want := got.Description().Detail, want.Description().Detail; got != want {
			t.Errorf("diagnostics[%d] detail\ngot:  %s\nwant: %s", i, got, want)
		}
	}
}

func TestLoadConfigDirBasics(t *testing.T) {
	bundle, err := sourcebundle.OpenDir("testdata/basics-bundle")
	if err != nil {
		t.Fatal(err)
	}

	rootAddr := sourceaddrs.MustParseSource("git::https://example.com/root.git").(sourceaddrs.RemoteSource)
	config, diags := LoadConfigDir(rootAddr, bundle)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics:\n%s", diags.NonFatalErr().Error())
	}

	t.Run("root input variables", func(t *testing.T) {
		if got, want := len(config.Root.Stack.InputVariables), 2; got != want {
			t.Errorf("wrong number of input variables %d; want %d", got, want)
		}
		t.Run("name", func(t *testing.T) {
			cfg, ok := config.Root.Stack.InputVariables["name"]
			if !ok {
				t.Fatal("Root stack config has no variable named \"name\".")
			}
			if got, want := cfg.Name, "name"; got != want {
				t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := cfg.Type.Constraint, cty.String; got != want {
				t.Errorf("wrong name\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Sensitive, false; got != want {
				t.Errorf("wrong sensitive\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Ephemeral, false; got != want {
				t.Errorf("wrong ephemeral\ngot:  %#v\nwant: %#v", got, want)
			}
		})
		t.Run("auth_jwt", func(t *testing.T) {
			cfg, ok := config.Root.Stack.InputVariables["auth_jwt"]
			if !ok {
				t.Fatal("Root stack config has no variable named \"auth_jwt\".")
			}
			if got, want := cfg.Name, "auth_jwt"; got != want {
				t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := cfg.Type.Constraint, cty.String; got != want {
				t.Errorf("wrong name\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Sensitive, true; got != want {
				t.Errorf("wrong sensitive\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Ephemeral, true; got != want {
				t.Errorf("wrong ephemeral\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	})
	t.Run("root output values", func(t *testing.T) {
		if got, want := len(config.Root.Stack.OutputValues), 3; got != want {
			t.Errorf("wrong number of output values %d; want %d", got, want)
		}
		t.Run("greeting", func(t *testing.T) {
			cfg, ok := config.Root.Stack.OutputValues["greeting"]
			if !ok {
				t.Fatal("Root stack config has no output value named \"greeting\".")
			}
			if got, want := cfg.Name, "greeting"; got != want {
				t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := cfg.Type.Constraint, cty.String; got != want {
				t.Errorf("wrong name\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Sensitive, false; got != want {
				t.Errorf("wrong sensitive\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Ephemeral, false; got != want {
				t.Errorf("wrong ephemeral\ngot:  %#v\nwant: %#v", got, want)
			}
		})
		t.Run("sound", func(t *testing.T) {
			cfg, ok := config.Root.Stack.OutputValues["sound"]
			if !ok {
				t.Fatal("Root stack config has no output value named \"sound\".")
			}
			if got, want := cfg.Name, "sound"; got != want {
				t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := cfg.Type.Constraint, cty.String; got != want {
				t.Errorf("wrong name\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Sensitive, false; got != want {
				t.Errorf("wrong sensitive\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Ephemeral, false; got != want {
				t.Errorf("wrong ephemeral\ngot:  %#v\nwant: %#v", got, want)
			}
		})
		t.Run("password", func(t *testing.T) {
			cfg, ok := config.Root.Stack.OutputValues["password"]
			if !ok {
				t.Fatal("Root stack config has no output value named \"password\".")
			}
			if got, want := cfg.Name, "password"; got != want {
				t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := cfg.Type.Constraint, cty.String; got != want {
				t.Errorf("wrong name\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Sensitive, true; got != want {
				t.Errorf("wrong sensitive\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Ephemeral, true; got != want {
				t.Errorf("wrong ephemeral\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	})
	// TODO: More thorough testing!
}

func TestOmittingBuiltInProviders(t *testing.T) {
	bundle, err := sourcebundle.OpenDir("testdata/basics-bundle")
	if err != nil {
		t.Fatal(err)
	}

	rootAddr := sourceaddrs.MustParseSource("git::https://example.com/builtin.git").(sourceaddrs.RemoteSource)
	config, diags := LoadConfigDir(rootAddr, bundle)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics:\n%s", diags.NonFatalErr().Error())
	}

	t.Run("built-in providers do NOT have to be listed in required providers", func(t *testing.T) {
		if got, want := len(config.Root.Stack.OutputValues), 1; got != want {
			t.Errorf("wrong number of output values %d; want %d", got, want)
		}

		t.Run("greeting", func(t *testing.T) {
			cfg, ok := config.Root.Stack.OutputValues["greeting"]
			if !ok {
				t.Fatal("Root stack config has no output value named \"greeting\".")
			}
			if got, want := cfg.Name, "greeting"; got != want {
				t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := cfg.Type.Constraint, cty.String; got != want {
				t.Errorf("wrong name\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Sensitive, false; got != want {
				t.Errorf("wrong sensitive\ngot:  %#v\nwant: %#v", got, want)
			}
			if got, want := cfg.Ephemeral, false; got != want {
				t.Errorf("wrong ephemeral\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	})
}

func TestComponentSourceResolution(t *testing.T) {
	stackResourceName := "pet-nulls"
	bundle, err := sourcebundle.OpenDir("testdata/embedded-stack-bundle")
	if err != nil {
		t.Fatal(err)
	}

	rootAddr := sourceaddrs.MustParseSource("git::https://example.com/root.git").(sourceaddrs.RemoteSource)
	config, diags := LoadConfigDir(rootAddr, bundle)
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics:\n%s", diags.NonFatalErr().Error())
	}

	t.Run("component source resolution", func(t *testing.T) {
		// Verify that the component was loaded
		if got, want := len(config.Root.Stack.EmbeddedStacks), 1; got != want {
			t.Errorf("wrong number of components %d; want %d", got, want)
		}

		t.Run("pet-nulls component", func(t *testing.T) {
			cmpn, ok := config.Root.Stack.EmbeddedStacks[stackResourceName]
			if !ok {
				t.Fatalf("Root stack config has no component named %q.", stackResourceName)
			}

			// Verify component name
			if got, want := cmpn.Name, stackResourceName; got != want {
				t.Errorf("wrong component name\ngot:  %s\nwant: %s", got, want)
			}

			// Verify that the source address was parsed correctly
			componentSource, ok := cmpn.SourceAddr.(sourceaddrs.ComponentSource)
			if !ok {
				t.Fatalf("expected ComponentSource, got %T", cmpn.SourceAddr)
			}

			expectedSourceStr := "example.com/awesomecorp/tfstack-pet-nulls"
			if got := componentSource.String(); got != expectedSourceStr {
				t.Errorf("wrong source address\ngot:  %s\nwant: %s", got, expectedSourceStr)
			}

			// Verify that version constraints were parsed
			if cmpn.VersionConstraints == nil {
				t.Fatal("component has no version constraints")
			}

			// Verify that the final source address was resolved
			if cmpn.FinalSourceAddr == nil {
				t.Fatal("component FinalSourceAddr was not resolved")
			}

			// The final source should be a ComponentSourceFinal
			componentSourceFinal, ok := cmpn.FinalSourceAddr.(sourceaddrs.ComponentSourceFinal)
			if !ok {
				t.Fatalf("expected ComponentSourceFinal for FinalSourceAddr, got %T", cmpn.FinalSourceAddr)
			}

			// Verify it resolved to the correct version (0.0.2)
			expectedVersion := "0.0.2"
			if got := componentSourceFinal.SelectedVersion().String(); got != expectedVersion {
				t.Errorf("wrong selected version\ngot:  %s\nwant: %s", got, expectedVersion)
			}

			// Verify the unversioned component source matches
			if got := componentSourceFinal.Unversioned().String(); got != expectedSourceStr {
				t.Errorf("wrong unversioned source in final address\ngot:  %s\nwant: %s", got, expectedSourceStr)
			}

			// Verify we can get the local path from the bundle
			localPath, err := bundle.LocalPathForSource(cmpn.FinalSourceAddr)
			if err != nil {
				t.Fatalf("failed to get local path for component source: %s", err)
			}

			// The local path should point to the pet-nulls directory
			if localPath == "" {
				t.Error("local path is empty")
			}
			t.Logf("Component resolved to local path: %s", localPath)
		})
	})
}
