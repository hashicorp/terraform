// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

func TestPolicyProviderTable(t *testing.T) {
	cfgType := cty.Object(map[string]cty.Type{
		"region": cty.String,
		"zones":  cty.Set(cty.String),
	})
	cfg := func(region string, zones ...string) cty.Value {
		zoneVals := cty.SetValEmpty(cty.String)
		if len(zones) > 0 {
			vals := make([]cty.Value, len(zones))
			for i, z := range zones {
				vals[i] = cty.StringVal(z)
			}
			zoneVals = cty.SetVal(vals)
		}
		return cty.ObjectVal(map[string]cty.Value{
			"region": cty.StringVal(region),
			"zones":  zoneVals,
		})
	}
	defaultAddr := mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`)
	aliasAddr := mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].alias`)
	otherAddr := mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].other`)
	moduleAddr := mustProviderConfig(`module.child.provider["registry.terraform.io/hashicorp/test"]`)
	otherSourceAddr := mustProviderConfig(`provider["registry.terraform.io/hashicorp/other"]`)

	byAddr := func(t *testing.T, table *policyProviderTable) map[string]*proto.ProviderInstance {
		t.Helper()
		ret := make(map[string]*proto.ProviderInstance)
		for _, p := range table.all() {
			ret[p.ConfigAddress] = p
		}
		return ret
	}

	t.Run("same config with different aliases has the same class", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cfg("us-east-1", "a", "b"), cfgType)
		table.configured(aliasAddr, cfg("us-east-1", "a", "b"), cfgType)
		table.configured(moduleAddr, cfg("us-east-1", "a", "b"), cfgType)

		got := byAddr(t, table)
		def, alias, mod := got[defaultAddr.String()], got[aliasAddr.String()], got[moduleAddr.String()]
		for _, p := range []*proto.ProviderInstance{def, alias, mod} {
			if !p.Known {
				t.Fatalf("expected %s to be known", p.ConfigAddress)
			}
			if len(p.ConfigClass) != 32 {
				t.Fatalf("expected a 32 byte class for %s, got %d bytes", p.ConfigAddress, len(p.ConfigClass))
			}
			if p.Source != "registry.terraform.io/hashicorp/test" {
				t.Fatalf("wrong source for %s: %s", p.ConfigAddress, p.Source)
			}
		}
		if !bytes.Equal(def.ConfigClass, alias.ConfigClass) || !bytes.Equal(def.ConfigClass, mod.ConfigClass) {
			t.Fatal("expected identical configurations to have the same class")
		}
	})

	t.Run("different arguments have different classes", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cfg("us-east-1"), cfgType)
		table.configured(aliasAddr, cfg("eu-west-1"), cfgType)
		table.configured(otherAddr, cfg("us-east-1", "a"), cfgType)

		got := byAddr(t, table)
		def, alias, other := got[defaultAddr.String()], got[aliasAddr.String()], got[otherAddr.String()]
		if bytes.Equal(def.ConfigClass, alias.ConfigClass) {
			t.Fatal("expected different regions to have different classes")
		}
		if bytes.Equal(def.ConfigClass, other.ConfigClass) {
			t.Fatal("expected different zones to have different classes")
		}
	})

	t.Run("the source is part of the class", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cfg("us-east-1"), cfgType)
		table.configured(otherSourceAddr, cfg("us-east-1"), cfgType)

		got := byAddr(t, table)
		if bytes.Equal(got[defaultAddr.String()].ConfigClass, got[otherSourceAddr.String()].ConfigClass) {
			t.Fatal("expected different provider sources to have different classes")
		}
	})

	t.Run("unknown arguments are not known", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cty.ObjectVal(map[string]cty.Value{
			"region": cty.UnknownVal(cty.String),
			"zones":  cty.SetValEmpty(cty.String),
		}), cfgType)

		p := table.all()[0]
		if p.Known {
			t.Fatal("expected a configuration with unknown values not to be known")
		}
		if len(p.ConfigClass) != 0 {
			t.Fatalf("expected no class, got %x", p.ConfigClass)
		}
	})

	t.Run("a configuration that doesn't conform to the type is not known", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cty.ObjectVal(map[string]cty.Value{
			"unexpected": cty.StringVal("x"),
		}), cfgType)
		table.configured(aliasAddr, cfg("us-east-1"), cty.NilType)

		for _, p := range table.all() {
			if p.Known || len(p.ConfigClass) != 0 {
				t.Fatalf("expected %s not to be known, got known=%t class=%x", p.ConfigAddress, p.Known, p.ConfigClass)
			}
		}
	})

	t.Run("marks don't matter", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cfg("us-east-1", "a"), cfgType)
		table.configured(aliasAddr, cty.ObjectVal(map[string]cty.Value{
			"region": cty.StringVal("us-east-1").Mark(marks.Sensitive),
			"zones":  cty.SetVal([]cty.Value{cty.StringVal("a").Mark(marks.Sensitive)}),
		}), cfgType)

		got := byAddr(t, table)
		if !got[aliasAddr.String()].Known {
			t.Fatal("expected a marked configuration to be known")
		}
		if !bytes.Equal(got[defaultAddr.String()].ConfigClass, got[aliasAddr.String()].ConfigClass) {
			t.Fatal("expected marks not to change the class")
		}
	})

	t.Run("sets built in different orders have the same class", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cfg("us-east-1", "c", "a", "b"), cfgType)
		table.configured(aliasAddr, cfg("us-east-1", "b", "c", "a"), cfgType)

		got := byAddr(t, table)
		if !bytes.Equal(got[defaultAddr.String()].ConfigClass, got[aliasAddr.String()].ConfigClass) {
			t.Fatal("expected the class not to depend on set element order")
		}
	})

	t.Run("classes differ between tables", func(t *testing.T) {
		a, b := newPolicyProviderTable(), newPolicyProviderTable()
		a.configured(defaultAddr, cfg("us-east-1"), cfgType)
		b.configured(defaultAddr, cfg("us-east-1"), cfgType)
		if bytes.Equal(a.all()[0].ConfigClass, b.all()[0].ConfigClass) {
			t.Fatal("expected each table to use its own key")
		}
	})

	t.Run("ids start at 1 in order of first use and are stable", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(aliasAddr, cfg("us-east-1"), cfgType)
		if got := table.idFor(otherAddr); got != 2 {
			t.Fatalf("expected the unconfigured provider to get id 2, got %d", got)
		}
		table.configured(defaultAddr, cfg("us-east-1"), cfgType)
		if got := table.idFor(aliasAddr); got != 1 {
			t.Fatalf("expected the first provider to keep id 1, got %d", got)
		}
		if got := table.idFor(otherAddr); got != 2 {
			t.Fatalf("expected the unconfigured provider to keep id 2, got %d", got)
		}
		if got := table.idFor(defaultAddr); got != 3 {
			t.Fatalf("expected the default provider to get id 3, got %d", got)
		}

		all := table.all()
		if len(all) != 3 {
			t.Fatalf("expected 3 providers, got %d", len(all))
		}
		wantOrder := []addrs.AbsProviderConfig{aliasAddr, otherAddr, defaultAddr}
		for i, p := range all {
			if p.Id != uint32(i+1) || p.ConfigAddress != wantOrder[i].String() {
				t.Fatalf("provider %d: got id %d for %s, want id %d for %s", i, p.Id, p.ConfigAddress, i+1, wantOrder[i])
			}
		}
		if unconfigured := all[1]; unconfigured.Known || len(unconfigured.ConfigClass) != 0 {
			t.Fatalf("expected an unconfigured provider not to be known, got %v", unconfigured)
		}
		if unconfigured := all[1]; unconfigured.Source != "registry.terraform.io/hashicorp/test" {
			t.Fatalf("expected the source of an unconfigured provider, got %q", unconfigured.Source)
		}
	})

	t.Run("all returns copies", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.configured(defaultAddr, cfg("us-east-1"), cfgType)
		table.all()[0].Known = false
		if !table.all()[0].Known {
			t.Fatal("expected all to return copies")
		}
	})
}
