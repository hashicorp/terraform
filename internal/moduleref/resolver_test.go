// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleref

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/modsdir"
)

func TestResolver_Resolve(t *testing.T) {
	cfg := configs.NewEmptyConfig()
	cfg.Module = &configs.Module{
		ModuleCalls: map[string]*configs.ModuleCall{
			"foo": {Name: "foo"},
		},
	}

	manifest := modsdir.Manifest{
		"foo": modsdir.Record{
			Key:        "foo",
			SourceAddr: "./foo",
		},
		"bar": modsdir.Record{
			Key:        "bar",
			SourceAddr: "./bar",
		},
	}

	resolver := NewResolver(manifest)
	result := resolver.Resolve(cfg)

	if len(result.Records) != 2 {
		t.Fatalf("expected the resolved number of entries to match the manifest, got: %d", len(result.Records))
	}

	// For the foo record
	if !result.Records[0].ReferencedInConfiguration {
		t.Fatal("expected to find reference for module \"foo\"")
	}

	// For the bar record
	if result.Records[1].ReferencedInConfiguration {
		t.Fatal("expected module \"bar\" to not be referenced in config")
	}
}

func TestResolver_ResolveNestedChildren(t *testing.T) {
	cfg := configs.NewEmptyConfig()
	cfg.Children = make(map[string]*configs.Config)
	cfg.Module = &configs.Module{
		ModuleCalls: map[string]*configs.ModuleCall{
			"foo": {Name: "foo"},
		},
	}

	childCfg := &configs.Config{
		Path:     addrs.Module{"fellowship"},
		Children: make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{
				"frodo": {Name: "frodo"},
			},
		},
	}

	childCfg2 := &configs.Config{
		Path:     addrs.Module{"fellowship", "weapons"},
		Children: make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{
				"sting": {Name: "sting"},
			},
		},
	}

	cfg.Children["fellowship"] = childCfg
	childCfg.Children["weapons"] = childCfg2

	manifest := modsdir.Manifest{
		"foo": modsdir.Record{
			Key:        "foo",
			SourceAddr: "./foo",
		},
		"bar": modsdir.Record{
			Key:        "bar",
			SourceAddr: "./bar",
		},
		"fellowship.frodo": modsdir.Record{
			Key:        "fellowship.frodo",
			SourceAddr: "fellowship/frodo",
		},
		"fellowship.weapons.sting": modsdir.Record{
			Key:        "fellowship.weapons.sting",
			SourceAddr: "fellowship/weapons/sting",
		},
		"fellowship.weapons.anduril": modsdir.Record{
			Key:        "fellowship.weapons.anduril",
			SourceAddr: "fellowship/weapons/anduril",
		},
	}

	resolver := NewResolver(manifest)
	result := resolver.Resolve(cfg)

	if len(result.Records) != 5 {
		t.Fatalf("expected the resolved number of entries to match the manifest, got: %d", len(result.Records))
	}

	assertions := map[string]bool{
		"foo":                        true,
		"bar":                        false,
		"fellowship.frodo":           true,
		"fellowship.weapons.sting":   true,
		"fellowship.weapons.anduril": false,
	}

	for _, record := range result.Records {
		referenced, ok := assertions[record.Key]
		if !ok {
			t.Fatalf("expected to find entry with key: %s", record.Key)
		}

		if referenced != record.ReferencedInConfiguration {
			t.Fatalf("expected key %s to be referenced: %t, got: %t", record.Key, referenced, record.ReferencedInConfiguration)
		}
	}
}

func TestResolver_normalizeModulePath(t *testing.T) {
	testCases := []struct {
		path string
		want string
	}{
		{
			path: "",
			want: "",
		},
		{
			path: "foo",
			want: "foo",
		},
		{
			path: "module.bar",
			want: "bar",
		},
		{
			path: "module.foo.module.bar",
			want: "foo.bar",
		},
		{
			path: "module.foomodule.module.barmodule.module.a",
			want: "foomodule.barmodule.a",
		},
		{
			path: "module.modulea.module.moduleb",
			want: "modulea.moduleb",
		},
	}

	for _, tc := range testCases {
		r := &Resolver{}
		got := r.normalizeModulePath(tc.path)
		if got != tc.want {
			t.Fatalf("expected %s, got: %s\n", tc.want, got)
		}
	}
}
