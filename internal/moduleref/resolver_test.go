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

	cfg.Children = map[string]*configs.Config{
		"foo": &configs.Config{
			Path:       addrs.Module{"foo"},
			Parent:     cfg,
			Children:   make(map[string]*configs.Config),
			SourceAddr: addrs.ModuleSourceLocal("./foo"),
			Module: &configs.Module{
				ModuleCalls: map[string]*configs.ModuleCall{},
			},
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

	if len(result.Records) != 1 {
		t.Fatalf("expected the resolved number of entries to equal 1, got: %d", len(result.Records))
	}

	// For the foo record
	if result.Records[0].Key != "foo" {
		t.Fatal("expected to find reference for module \"foo\"")
	}
}

func TestResolver_ResolveNestedChildren(t *testing.T) {
	cfg := configs.NewEmptyConfig()
	cfg.Children = make(map[string]*configs.Config)
	cfg.Module = &configs.Module{
		ModuleCalls: map[string]*configs.ModuleCall{
			"foo":        {Name: "foo"},
			"fellowship": {Name: "fellowship"},
		},
	}

	cfg.Children["foo"] = &configs.Config{
		Path:       addrs.Module{"foo"},
		Parent:     cfg,
		SourceAddr: addrs.ModuleSourceLocal("./foo"),
		Children:   make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{},
		},
	}

	childCfgFellowship := &configs.Config{
		Path:   addrs.Module{"fellowship"},
		Parent: cfg,
		SourceAddr: addrs.ModuleSourceRemote{
			Package: addrs.ModulePackage("fellowship"),
		},
		Children: make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{
				"frodo": {Name: "frodo"},
			},
		},
	}
	cfg.Children["fellowship"] = childCfgFellowship

	childCfgFellowship.Children["frodo"] = &configs.Config{
		Path:   addrs.Module{"fellowship", "frodo"},
		Parent: childCfgFellowship,
		SourceAddr: addrs.ModuleSourceRemote{
			Package: addrs.ModulePackage("fellowship/frodo"),
		},
		Children: make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{},
		},
	}

	childCfgWeapons := &configs.Config{
		Path:   addrs.Module{"fellowship", "weapons"},
		Parent: childCfgFellowship,
		SourceAddr: addrs.ModuleSourceRemote{
			Package: addrs.ModulePackage("fellowship/weapons"),
		},
		Children: make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{
				"sting": {Name: "sting"},
			},
		},
	}
	childCfgFellowship.Children["weapons"] = childCfgWeapons

	childCfgWeapons.Children["sting"] = &configs.Config{
		Path:   addrs.Module{"fellowship", "weapons", "sting"},
		Parent: childCfgWeapons,
		SourceAddr: addrs.ModuleSourceRemote{
			Package: addrs.ModulePackage("fellowship/weapons/sting"),
		},
		Children: make(map[string]*configs.Config),
		Module: &configs.Module{
			ModuleCalls: map[string]*configs.ModuleCall{},
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
		"fellowship": modsdir.Record{
			Key:        "fellowship",
			SourceAddr: "fellowship",
		},
		"fellowship.frodo": modsdir.Record{
			Key:        "fellowship.frodo",
			SourceAddr: "fellowship/frodo",
		},
		"fellowship.weapons": modsdir.Record{
			Key:        "fellowship.weapons",
			SourceAddr: "fellowship/weapons",
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
	recordsCount, sources := countAndListSources(result.Records)
	if recordsCount != 5 {
		t.Fatalf("expected the resolved number of entries to equal 5, got: %d", recordsCount)
	}

	assertions := map[string]bool{
		"./foo":                      true,
		"./bar":                      false,
		"fellowship":                 true,
		"fellowship/frodo":           true,
		"fellowship/weapons":         true,
		"fellowship/weapons/sting":   true,
		"fellowship/weapons/anduril": false,
	}

	for _, source := range sources {
		referenced, ok := assertions[source]
		if !ok || !referenced {
			t.Fatalf("expected to find referenced entry with key: %s", source)
		}
	}
}

func countAndListSources(records Records) (count int, sources []string) {
	for _, record := range records {
		sources = append(sources, record.Source.String())
		count++
		childCount, childSources := countAndListSources(record.Children)
		count += childCount
		sources = append(sources, childSources...)
	}
	return
}
