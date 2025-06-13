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

func TestLoadConfigDirDeprecated(t *testing.T) {
	bundle, err := sourcebundle.OpenDir("testdata/basics-bundle")
	if err != nil {
		t.Fatal(err)
	}

	rootAddr := sourceaddrs.MustParseSource("git::https://example.com/deprecated.git").(sourceaddrs.RemoteSource)
	_, gotDiags := LoadConfigDir(rootAddr, bundle)

	wantDiags := tfdiags.Diagnostics{
		tfdiags.Sourceless(tfdiags.Warning, "Deprecated filename usage", "This configuration is using the deprecated .tfstack.hcl or .tfstack.json file extensions. This will not be supported in a future version of Terraform, please update your files to use the latest .tfcomponent.hcl or .tfcomponent.json file extensions."),
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
