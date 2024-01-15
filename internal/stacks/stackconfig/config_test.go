// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/zclconf/go-cty/cty"
)

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
		if got, want := len(config.Root.Stack.InputVariables), 1; got != want {
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
		})
	})
	t.Run("root output values", func(t *testing.T) {
		if got, want := len(config.Root.Stack.OutputValues), 2; got != want {
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
		})
	})
	// TODO: More thorough testing!
}
