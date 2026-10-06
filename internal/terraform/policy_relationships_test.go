// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
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

	t.Run("providers the walk didn't configure are classified statically", func(t *testing.T) {
		table := newPolicyProviderTable()
		table.idFor(otherAddr)
		table.idFor(aliasAddr)
		table.configured(defaultAddr, cfg("us-east-1"), cfgType)
		var unconfigured []string
		for _, addr := range table.unconfiguredAddrs() {
			unconfigured = append(unconfigured, addr.String())
		}
		if diff := cmp.Diff([]string{aliasAddr.String(), otherAddr.String()}, unconfigured); diff != "" {
			t.Fatalf("wrong unconfigured providers (-want +got):\n%s", diff)
		}

		// A static class doesn't replace the class of a provider the walk
		// configured.
		table.configuredStatically(defaultAddr, cfg("eu-west-1"), cfgType)
		table.configuredStatically(aliasAddr, cfg("us-east-1"), cfgType)
		got := byAddr(t, table)
		def, alias, other := got[defaultAddr.String()], got[aliasAddr.String()], got[otherAddr.String()]
		if !alias.Known || !bytes.Equal(def.ConfigClass, alias.ConfigClass) {
			t.Fatalf("expected the static class to equal the class of the same configuration, got %v and %v", def, alias)
		}
		if other.Known || len(other.ConfigClass) != 0 {
			t.Fatalf("expected a provider without a static class not to be known, got %v", other)
		}

		// A configuration during the walk replaces a static class.
		table.configured(aliasAddr, cfg("eu-west-1"), cfgType)
		if alias := byAddr(t, table)[aliasAddr.String()]; bytes.Equal(def.ConfigClass, alias.ConfigClass) {
			t.Fatal("expected the walk's configuration to replace the static class")
		}
	})
}

func TestPolicyStaticProviderConfig(t *testing.T) {
	schemas := &schemarepo.Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
		addrs.NewDefaultProvider("test"): *relationshipsTestProvider().GetProviderSchemaResponse,
	}}
	region := func(v cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{"region": v})
	}
	tests := map[string]struct {
		files   map[string]string
		addr    string
		outside bool
		// want is the decoded configuration; cty.NilVal if it can't be
		// known statically.
		want cty.Value
	}{
		"literal": {
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "a"
					region = "r1"
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"].a`,
			want: region(cty.StringVal("r1")),
		},
		"no provider block": {
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "a"
					region = "r1"
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
			want: region(cty.NullVal(cty.String)),
		},
		"JSON literal": {
			files: map[string]string{"main.tf.json": `{"provider": {"test": [{"alias": "j", "region": "r1"}]}}`},
			addr:  `provider["registry.terraform.io/hashicorp/test"].j`,
			want:  region(cty.StringVal("r1")),
		},
		"JSON template": {
			files: map[string]string{"main.tf.json": `{
				"variable": {"r": {"default": "r1"}},
				"provider": {"test": [{"alias": "j", "region": "${var.r}"}]}
			}`},
			addr: `provider["registry.terraform.io/hashicorp/test"].j`,
		},
		"variable": {
			files: map[string]string{"main.tf": `
				variable "r" {
					default = "r1"
				}
				provider "test" {
					region = var.r
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
		},
		"function call": {
			files: map[string]string{"main.tf": `
				provider "test" {
					region = lower("R1")
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
		},
		"unsupported argument": {
			files: map[string]string{"main.tf": `
				provider "test" {
					zone = "z"
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
		},
		"module": {
			files: map[string]string{
				"main.tf": `
					module "child" {
						source = "./child"
					}
				`,
				"child/main.tf": `
					provider "test" {
						region = "r1"
					}
				`,
			},
			addr: `module.child.provider["registry.terraform.io/hashicorp/test"]`,
			want: region(cty.StringVal("r1")),
		},
		"module not in the configuration": {
			files: map[string]string{"main.tf": ``},
			addr:  `module.gone.provider["registry.terraform.io/hashicorp/test"]`,
		},
		"values from outside the configuration": {
			files:   map[string]string{"main.tf": ``},
			addr:    `provider["registry.terraform.io/hashicorp/test"]`,
			outside: true,
		},
		"no schema": {
			files: map[string]string{"main.tf": `
				terraform {
					required_providers {
						other = {
							source = "hashicorp/other"
						}
					}
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/other"]`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testModuleInline(t, test.files)
			got, gotType, ok := policyStaticProviderConfig(cfg, schemas, mustProviderConfig(test.addr), test.outside)
			if test.want == cty.NilVal {
				if ok {
					t.Fatalf("expected no static configuration, got %#v", got)
				}
				return
			}
			if !ok {
				t.Fatal("expected a static configuration")
			}
			if !got.RawEquals(test.want) {
				t.Fatalf("wrong configuration\ngot:  %#v\nwant: %#v", got, test.want)
			}
			if !gotType.Equals(cty.Object(map[string]cty.Type{"region": cty.String})) {
				t.Fatalf("wrong type %#v", gotType)
			}
		})
	}
}

func TestPolicyProviderSchemas(t *testing.T) {
	block := func(attrs map[string]*configschema.Attribute, blocks map[string]*configschema.NestedBlock) *configschema.Block {
		return &configschema.Block{Attributes: attrs, BlockTypes: blocks}
	}
	vmSchema := block(map[string]*configschema.Attribute{
		"id":       {Type: cty.String, Computed: true},
		"password": {Type: cty.String, Optional: true, WriteOnly: true},
		"disks": {
			NestedType: &configschema.Object{
				Nesting: configschema.NestingList,
				Attributes: map[string]*configschema.Attribute{
					"size": {Type: cty.Number, Optional: true},
					"key":  {Type: cty.String, Optional: true, WriteOnly: true},
				},
			},
			Optional: true,
		},
	}, map[string]*configschema.NestedBlock{
		"nic": {
			Nesting: configschema.NestingSet,
			Block: *block(map[string]*configschema.Attribute{
				"net_id": {Type: cty.String, Optional: true},
			}, map[string]*configschema.NestedBlock{
				"auth": {
					Nesting: configschema.NestingSingle,
					Block: *block(map[string]*configschema.Attribute{
						"secret": {Type: cty.String, Optional: true, WriteOnly: true},
					}, nil),
				},
			}),
		},
	})
	netSchema := block(map[string]*configschema.Attribute{
		"id": {Type: cty.String, Computed: true},
	}, nil)
	infoSchema := block(map[string]*configschema.Attribute{
		"id":    {Type: cty.String, Computed: true},
		"token": {Type: cty.String, Optional: true, WriteOnly: true},
	}, nil)
	tokenSchema := block(map[string]*configschema.Attribute{
		"value": {Type: cty.String, Computed: true},
	}, nil)

	testProvider := addrs.NewDefaultProvider("test")
	otherProvider := addrs.NewProvider("example.com", "acme", "other")
	schemas := &schemarepo.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			testProvider: {
				Provider: providers.Schema{Body: block(map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				}, nil)},
				ResourceTypes: map[string]providers.Schema{
					"test_vm":  {Body: vmSchema},
					"test_net": {Body: netSchema},
				},
				DataSources: map[string]providers.Schema{
					"test_info": {Body: infoSchema},
				},
				EphemeralResourceTypes: map[string]providers.Schema{
					"test_token": {Body: tokenSchema},
				},
			},
			otherProvider: {
				ResourceTypes: map[string]providers.Schema{
					"other_thing": {Body: netSchema},
				},
			},
		},
	}

	got := policyProviderSchemas(schemas)
	if len(got) != 2 {
		t.Fatalf("expected 2 provider schemas, got %d", len(got))
	}

	// sorted by source
	if got[0].Source != otherProvider.String() || got[1].Source != testProvider.String() {
		t.Fatalf("wrong order: %s, %s", got[0].Source, got[1].Source)
	}
	if got[0].Type != "other" || got[1].Type != "test" {
		t.Fatalf("wrong types: %s, %s", got[0].Type, got[1].Type)
	}

	test := got[1]
	assertType := func(t *testing.T, encoded map[string][]byte, name string, want cty.Type) {
		t.Helper()
		raw, ok := encoded[name]
		if !ok {
			t.Fatalf("no type for %s", name)
		}
		ty, err := ctyjson.UnmarshalType(raw)
		if err != nil {
			t.Fatalf("invalid type for %s: %s", name, err)
		}
		if !ty.Equals(want) {
			t.Fatalf("wrong type for %s\ngot:  %#v\nwant: %#v", name, ty, want)
		}
	}
	if len(test.Resources) != 2 || len(test.DataSources) != 1 || len(test.EphemeralResources) != 1 {
		t.Fatalf("wrong number of types: %d resources, %d data sources, %d ephemeral resources", len(test.Resources), len(test.DataSources), len(test.EphemeralResources))
	}
	assertType(t, test.Resources, "test_vm", vmSchema.ImpliedType())
	assertType(t, test.Resources, "test_net", netSchema.ImpliedType())
	assertType(t, test.DataSources, "test_info", infoSchema.ImpliedType())
	assertType(t, test.EphemeralResources, "test_token", tokenSchema.ImpliedType())

	if len(test.WriteOnlyPaths) != 1 {
		t.Fatalf("expected write-only paths for test_vm only, got %v", test.WriteOnlyPaths)
	}
	var gotPaths []string
	for _, path := range test.WriteOnlyPaths["test_vm"].GetPaths() {
		for _, step := range path.Steps {
			if _, ok := step.Selector.(*proto.AttributePath_Step_AttributeName); !ok {
				t.Fatalf("expected only attribute name steps, got %v", path)
			}
		}
		gotPaths = append(gotPaths, relPathString(path))
	}
	wantPaths := []string{"disks.key", "nic.auth.secret", "password"}
	if diff := cmp.Diff(wantPaths, gotPaths); diff != "" {
		t.Fatalf("wrong write-only paths (-want +got):\n%s", diff)
	}

	if len(got[0].WriteOnlyPaths) != 0 {
		t.Fatalf("expected no write-only paths for %s, got %v", got[0].Source, got[0].WriteOnlyPaths)
	}
	assertType(t, got[0].Resources, "other_thing", netSchema.ImpliedType())

	if got := policyProviderSchemas(nil); len(got) != 0 {
		t.Fatalf("expected no schemas for nil, got %v", got)
	}
}

func TestPolicyResourceAction(t *testing.T) {
	tests := []struct {
		action    plans.Action
		want      proto.ResourceAction
		wantValid bool
	}{
		{plans.NoOp, proto.ResourceAction_NO_OP_RESOURCE_ACTION, true},
		{plans.Create, proto.ResourceAction_CREATE_RESOURCE_ACTION, true},
		{plans.Update, proto.ResourceAction_UPDATE_RESOURCE_ACTION, true},
		{plans.Delete, proto.ResourceAction_DELETE_RESOURCE_ACTION, true},
		{plans.DeleteThenCreate, proto.ResourceAction_DELETE_THEN_CREATE_RESOURCE_ACTION, true},
		{plans.CreateThenDelete, proto.ResourceAction_CREATE_THEN_DELETE_RESOURCE_ACTION, true},
		{plans.Forget, proto.ResourceAction_FORGET_RESOURCE_ACTION, true},
		{plans.CreateThenForget, proto.ResourceAction_CREATE_THEN_FORGET_RESOURCE_ACTION, true},
		{plans.ForgetThenCreate, proto.ResourceAction_FORGET_THEN_CREATE_RESOURCE_ACTION, true},
		{plans.Read, proto.ResourceAction_INVALID_RESOURCE_ACTION, false},
		{plans.Open, proto.ResourceAction_INVALID_RESOURCE_ACTION, false},
		{plans.Renew, proto.ResourceAction_INVALID_RESOURCE_ACTION, false},
		{plans.Close, proto.ResourceAction_INVALID_RESOURCE_ACTION, false},
	}
	for _, test := range tests {
		t.Run(test.action.String(), func(t *testing.T) {
			got, ok := policyResourceAction(test.action)
			if ok != test.wantValid || got != test.want {
				t.Fatalf("got %s, %t; want %s, %t", got, ok, test.want, test.wantValid)
			}
		})
	}
}

func TestPolicyPriorAttrsAction(t *testing.T) {
	want := map[plans.Action]bool{
		plans.Delete:           true,
		plans.Forget:           true,
		plans.DeleteThenCreate: true,
		plans.CreateThenDelete: true,
		plans.CreateThenForget: true,
		plans.ForgetThenCreate: true,
	}
	for _, action := range []plans.Action{plans.NoOp, plans.Create, plans.Update, plans.Delete, plans.DeleteThenCreate, plans.CreateThenDelete, plans.Forget, plans.CreateThenForget, plans.ForgetThenCreate} {
		if got := policyRecordHasPriorAttrs(action, false); got != want[action] {
			t.Errorf("%s: got %t, want %t", action, got, want[action])
		}
		// include_prior adds the prior attrs of updates only; the engine
		// uses the attrs of a no-op as its prior attrs.
		wantIncluded := want[action] || action == plans.Update
		if got := policyRecordHasPriorAttrs(action, true); got != wantIncluded {
			t.Errorf("%s with include_prior: got %t, want %t", action, got, wantIncluded)
		}
	}
	for _, action := range []plans.Action{plans.Delete, plans.Forget} {
		if policyRecordHasAttrs(action) {
			t.Errorf("%s: expected no attrs", action)
		}
	}
	for _, action := range []plans.Action{plans.NoOp, plans.Create, plans.Update, plans.DeleteThenCreate, plans.CreateThenDelete, plans.CreateThenForget, plans.ForgetThenCreate} {
		if !policyRecordHasAttrs(action) {
			t.Errorf("%s: expected attrs", action)
		}
	}
}

func TestChunkRelationshipBatch(t *testing.T) {
	record := func(addr string, size int) *proto.InstanceRecord {
		rec := &proto.InstanceRecord{Address: addr, Type: "test_thing"}
		if size > 0 {
			rec.Attrs = &proto.ResourceAttributes{Raw: make([]byte, size)}
		}
		return rec
	}
	records := func(n int) []*proto.InstanceRecord {
		ret := make([]*proto.InstanceRecord, n)
		for i := range ret {
			ret[i] = record(fmt.Sprintf("test_thing.r%d", i), 0)
		}
		return ret
	}
	providers := []*proto.ProviderInstance{{Id: 1, ConfigAddress: "p1"}, {Id: 2, ConfigAddress: "p2"}}
	statuses := []*proto.TypeStatus{{Type: "test_thing", Completeness: proto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS}}

	setLimits := func(t *testing.T, maxRecords, maxBytes int) {
		oldRecords, oldBytes := relationshipChunkMaxRecords, relationshipChunkMaxBytes
		relationshipChunkMaxRecords, relationshipChunkMaxBytes = maxRecords, maxBytes
		t.Cleanup(func() {
			relationshipChunkMaxRecords, relationshipChunkMaxBytes = oldRecords, oldBytes
		})
	}
	addrsOf := func(req *proto.ReportInstancesRequest) []string {
		var ret []string
		for _, rec := range req.Records {
			ret = append(ret, rec.Address)
		}
		return ret
	}
	assertPlacement := func(t *testing.T, reqs []*proto.ReportInstancesRequest) {
		t.Helper()
		for i, req := range reqs {
			if req.RunId != "run" {
				t.Fatalf("request %d has run id %q", i, req.RunId)
			}
			if wantProviders := i == 0; (len(req.Providers) > 0) != wantProviders {
				t.Fatalf("request %d: expected providers only in the first request, got %d", i, len(req.Providers))
			}
			if wantStatuses := i == len(reqs)-1; (len(req.Statuses) > 0) != wantStatuses {
				t.Fatalf("request %d: expected statuses only in the last request, got %d", i, len(req.Statuses))
			}
		}
		if len(reqs[0].Providers) != len(providers) || len(reqs[len(reqs)-1].Statuses) != len(statuses) {
			t.Fatal("expected all providers and statuses to be sent")
		}
	}

	t.Run("defaults", func(t *testing.T) {
		if relationshipChunkMaxRecords != 1000 || relationshipChunkMaxBytes != 3<<20 {
			t.Fatalf("wrong default limits: %d records, %d bytes", relationshipChunkMaxRecords, relationshipChunkMaxBytes)
		}
	})

	t.Run("zero records", func(t *testing.T) {
		reqs := chunkRelationshipBatch(nil, statuses, providers, "run")
		if len(reqs) != 1 {
			t.Fatalf("expected 1 request, got %d", len(reqs))
		}
		assertPlacement(t, reqs)
	})

	t.Run("zero records, no statuses, no providers", func(t *testing.T) {
		reqs := chunkRelationshipBatch(nil, nil, nil, "run")
		if len(reqs) != 1 || reqs[0].RunId != "run" {
			t.Fatalf("expected 1 request, got %v", reqs)
		}
	})

	t.Run("record count limit", func(t *testing.T) {
		setLimits(t, 2, 1<<20)
		reqs := chunkRelationshipBatch(records(5), statuses, providers, "run")
		if len(reqs) != 3 {
			t.Fatalf("expected 3 requests, got %d", len(reqs))
		}
		want := [][]string{
			{"test_thing.r0", "test_thing.r1"},
			{"test_thing.r2", "test_thing.r3"},
			{"test_thing.r4"},
		}
		for i, req := range reqs {
			if diff := cmp.Diff(want[i], addrsOf(req)); diff != "" {
				t.Fatalf("wrong records in request %d (-want +got):\n%s", i, diff)
			}
		}
		assertPlacement(t, reqs)
	})

	t.Run("exactly at the record count limit", func(t *testing.T) {
		setLimits(t, 2, 1<<20)
		reqs := chunkRelationshipBatch(records(4), statuses, providers, "run")
		if len(reqs) != 2 {
			t.Fatalf("expected 2 requests, got %d", len(reqs))
		}
		assertPlacement(t, reqs)
	})

	t.Run("byte size limit", func(t *testing.T) {
		recs := []*proto.InstanceRecord{
			record("test_thing.a", 100),
			record("test_thing.b", 100),
			record("test_thing.c", 100),
		}
		size := protobuf.Size(recs[0])
		setLimits(t, 1000, 2*size)
		reqs := chunkRelationshipBatch(recs, statuses, providers, "run")
		if len(reqs) != 2 {
			t.Fatalf("expected 2 requests, got %d", len(reqs))
		}
		if diff := cmp.Diff([]string{"test_thing.a", "test_thing.b"}, addrsOf(reqs[0])); diff != "" {
			t.Fatalf("wrong records in request 1 (-want +got):\n%s", diff)
		}
		assertPlacement(t, reqs)
	})

	t.Run("a record over the byte size limit goes alone", func(t *testing.T) {
		recs := []*proto.InstanceRecord{
			record("test_thing.a", 10),
			record("test_thing.huge", 1000),
			record("test_thing.b", 10),
		}
		setLimits(t, 1000, 200)
		reqs := chunkRelationshipBatch(recs, statuses, providers, "run")
		want := [][]string{{"test_thing.a"}, {"test_thing.huge"}, {"test_thing.b"}}
		if len(reqs) != len(want) {
			t.Fatalf("expected %d requests, got %d", len(want), len(reqs))
		}
		for i, req := range reqs {
			if diff := cmp.Diff(want[i], addrsOf(req)); diff != "" {
				t.Fatalf("wrong records in request %d (-want +got):\n%s", i, diff)
			}
		}
		assertPlacement(t, reqs)
	})
}

func TestPolicyStateValue(t *testing.T) {
	schema := providers.Schema{Body: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":     {Type: cty.String, Computed: true},
			"secret": {Type: cty.String, Optional: true, Sensitive: true},
			"other":  {Type: cty.String, Optional: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"key": {Type: cty.String, Optional: true, Sensitive: true},
					},
				},
			},
		},
	}}
	obj := &states.ResourceInstanceObjectSrc{
		AttrsJSON: []byte(`{"id":"i","secret":"s","other":"o","nested":[{"key":"k"}]}`),
		// A path that was marked sensitive in configuration.
		AttrSensitivePaths: []cty.Path{cty.GetAttrPath("other")},
		Status:             states.ObjectReady,
	}

	val, err := policyStateValue(obj, schema)
	if err != nil {
		t.Fatal(err)
	}
	attrs, err := policy.EncodeResourceAttributes(val)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, path := range attrs.RedactedPaths {
		var parts []string
		for _, step := range path.Steps {
			switch s := step.Selector.(type) {
			case *proto.AttributePath_Step_AttributeName:
				parts = append(parts, s.AttributeName)
			case *proto.AttributePath_Step_ElementKeyInt:
				parts = append(parts, fmt.Sprint(s.ElementKeyInt))
			}
		}
		got = append(got, strings.Join(parts, "."))
	}
	sort.Strings(got)
	if diff := cmp.Diff([]string{"nested.0.key", "other", "secret"}, got); diff != "" {
		t.Fatalf("wrong redacted paths (-want +got):\n%s", diff)
	}

	t.Run("sensitive nested block in state", func(t *testing.T) {
		marked := *obj
		marked.AttrSensitivePaths = []cty.Path{cty.GetAttrPath("nested")}
		val, err := policyStateValue(&marked, schema)
		if err != nil {
			t.Fatal(err)
		}
		_, pvms := val.UnmarkDeepWithPaths()
		paths, _ := marks.PathsWithMark(pvms, marks.Sensitive)
		var got []string
		for _, path := range paths {
			got = append(got, format.CtyPath(path))
		}
		sort.Strings(got)
		if diff := cmp.Diff([]string{".nested", ".nested[0].key", ".secret"}, got); diff != "" {
			t.Fatalf("wrong sensitive paths (-want +got):\n%s", diff)
		}
	})

	t.Run("schema version mismatch", func(t *testing.T) {
		old := *obj
		old.SchemaVersion = 1
		if _, err := policyStateValue(&old, schema); err == nil {
			t.Fatal("expected an error for a state object with another schema version")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		bad := *obj
		bad.AttrsJSON = []byte(`{"unknown_attr":true}`)
		if _, err := policyStateValue(&bad, schema); err == nil {
			t.Fatal("expected an error for a state object that doesn't conform to the schema")
		}
	})
}

func TestPolicyPruneValue(t *testing.T) {
	nestedType := cty.List(cty.Object(map[string]cty.Type{"key": cty.String}))
	objType := cty.Object(map[string]cty.Type{
		"id":     cty.String,
		"name":   cty.String,
		"secret": cty.String,
		"nested": nestedType,
	})
	val := cty.ObjectVal(map[string]cty.Value{
		"id":     cty.StringVal("i"),
		"name":   cty.StringVal("n"),
		"secret": cty.StringVal("s").Mark(marks.Sensitive),
		"nested": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"key": cty.StringVal("k").Mark(marks.Sensitive),
		})}),
	})
	sensitivePaths := func(t *testing.T, v cty.Value) []string {
		t.Helper()
		_, pvms := v.UnmarkDeepWithPaths()
		paths, _ := marks.PathsWithMark(pvms, marks.Sensitive)
		var ret []string
		for _, path := range paths {
			ret = append(ret, format.CtyPath(path))
		}
		sort.Strings(ret)
		return ret
	}

	tests := map[string]struct {
		val           cty.Value
		keep          map[string]bool
		want          cty.Value
		wantSensitive []string
	}{
		"nothing to keep": {
			val:  val,
			keep: map[string]bool{},
			want: cty.ObjectVal(map[string]cty.Value{
				"id":     cty.NullVal(cty.String),
				"name":   cty.NullVal(cty.String),
				"secret": cty.NullVal(cty.String),
				"nested": cty.NullVal(nestedType),
			}),
		},
		"kept attributes keep their marks, pruned ones lose them": {
			val:  val,
			keep: map[string]bool{"id": true, "nested": true},
			want: cty.ObjectVal(map[string]cty.Value{
				"id":     cty.StringVal("i"),
				"name":   cty.NullVal(cty.String),
				"secret": cty.NullVal(cty.String),
				"nested": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"key": cty.StringVal("k"),
				})}),
			}),
			wantSensitive: []string{".nested[0].key"},
		},
		"a pruned nested block is a typed null": {
			val:  val,
			keep: map[string]bool{"secret": true},
			want: cty.ObjectVal(map[string]cty.Value{
				"id":     cty.NullVal(cty.String),
				"name":   cty.NullVal(cty.String),
				"secret": cty.StringVal("s"),
				"nested": cty.NullVal(nestedType),
			}),
			wantSensitive: []string{".secret"},
		},
		"no pruning": {
			val:  val,
			keep: nil,
			want: val,
		},
		"unknown attributes": {
			val: cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.UnknownVal(cty.String),
			}),
			keep: map[string]bool{"id": true},
			want: cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.NullVal(cty.String),
			}),
		},
		"a wholly sensitive value keeps its mark": {
			val: cty.ObjectVal(map[string]cty.Value{
				"id":   cty.StringVal("i"),
				"name": cty.StringVal("n"),
			}).Mark(marks.Sensitive),
			keep: map[string]bool{"id": true},
			want: cty.ObjectVal(map[string]cty.Value{
				"id":   cty.StringVal("i"),
				"name": cty.NullVal(cty.String),
			}),
			wantSensitive: []string{""},
		},
		"unknown object": {
			val:  cty.UnknownVal(objType),
			keep: map[string]bool{"id": true},
			want: cty.UnknownVal(objType),
		},
		"wholly unknown placeholder": {
			val:  cty.DynamicVal,
			keep: map[string]bool{"id": true},
			want: cty.DynamicVal,
		},
		"null object": {
			val:  cty.NullVal(objType),
			keep: map[string]bool{"id": true},
			want: cty.NullVal(objType),
		},
		"not an object": {
			val:  cty.StringVal("x"),
			keep: map[string]bool{"id": true},
			want: cty.StringVal("x"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := policyPruneValue(test.val, test.keep)
			if test.keep == nil {
				if !got.RawEquals(test.val) {
					t.Fatalf("expected the value unchanged, got %#v", got)
				}
				return
			}
			if diff := cmp.Diff(test.wantSensitive, sensitivePaths(t, got)); diff != "" {
				t.Errorf("wrong sensitive paths (-want +got):\n%s", diff)
			}
			unmarked, _ := got.UnmarkDeep()
			if !unmarked.RawEquals(test.want) {
				t.Fatalf("wrong value\ngot:  %#v\nwant: %#v", unmarked, test.want)
			}
		})
	}
}

func TestPolicyKeepAttributes(t *testing.T) {
	tests := map[string]struct {
		spec *proto.TypeSpec
		want map[string]bool
	}{
		"reads and the first steps of key paths": {
			spec: &proto.TypeSpec{
				KeyPaths:      []*proto.AttributePath{relAttrPath("nic.net_id"), relAttrPath("id")},
				Reads:         []string{"name", "zone"},
				ReadsComplete: true,
			},
			want: map[string]bool{"name": true, "zone": true, "nic": true, "id": true},
		},
		"only a subject": {
			spec: &proto.TypeSpec{ReadsComplete: true},
			want: map[string]bool{},
		},
		"incomplete reads": {
			spec: &proto.TypeSpec{
				KeyPaths: []*proto.AttributePath{relAttrPath("id")},
				Reads:    []string{"name"},
			},
			want: nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := policyKeepAttributes(test.spec)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("wrong attributes (-want +got):\n%s", diff)
			}
			if test.want != nil && got == nil {
				t.Fatal("expected a non-nil set")
			}
		})
	}
}

func TestPolicyBeginRunDiagnostics(t *testing.T) {
	warning, err := relDefinitionWarning(), relDefinitionError()
	invalid := &proto.Diagnostic{Severity: proto.Severity_INVALID, Summary: "No severity"}

	t.Run("warnings and errors are kept", func(t *testing.T) {
		assertRelDefinitionDiagnostics(t, policyBeginRunDiagnostics([]*proto.Diagnostic{err, invalid, warning}).AsTerraformDiags(), warning, err)
	})
	t.Run("diagnostics without severity are dropped", func(t *testing.T) {
		for _, diags := range [][]*proto.Diagnostic{nil, {invalid}} {
			if got := policyBeginRunDiagnostics(diags); len(got) != 0 {
				t.Errorf("expected no diagnostics, got %v", got.AsTerraformDiags().ErrWithWarnings())
			}
		}
	})
}

func TestCollectRelationshipBatch_deferrals(t *testing.T) {
	schemas := &schemarepo.Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
		addrs.NewDefaultProvider("test"): *relationshipsTestProvider().GetProviderSchemaResponse,
	}}
	deferredChange := func(addr addrs.AbsResourceInstance, provider addrs.AbsProviderConfig) *plans.ResourceInstanceChange {
		return &plans.ResourceInstanceChange{
			Addr:         addr,
			ProviderAddr: provider,
			Change: plans.Change{
				Action: plans.Create,
				Before: cty.NullVal(cty.DynamicPseudoType),
				After:  cty.DynamicVal,
			},
		}
	}
	deferInstance := func(d *deferring.Deferred, addr string, provider addrs.AbsProviderConfig) {
		inst := mustResourceInstanceAddr(addr)
		d.ReportResourceInstanceDeferred(inst, providers.DeferredReasonProviderConfigUnknown, deferredChange(inst, provider))
	}
	deferPartial := func(d *deferring.Deferred, addr addrs.PartialExpandedResource, provider addrs.AbsProviderConfig) {
		d.ReportResourceExpansionDeferred(addr, deferredChange(addr.UnknownResourceInstance(), provider))
	}
	defaultProvider := mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`)

	tests := map[string]struct {
		stage proto.EvaluationStage
		files map[string]string
		// setup registers the expansions and deferrals of the walk.
		setup func(exp *instances.Expander, d *deferring.Deferred)
		want  *proto.TypeStatus
		// wantProviders are the reported provider instances by config
		// address, with whether they're known.
		wantProviders map[string]bool
	}{
		"provider configuration only deferred instances use": {
			// The walk didn't configure the provider configuration of the
			// deferred instance, so its class comes from its configuration.
			stage: proto.EvaluationStage_APPLY_EVALUATION_STAGE,
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "lit"
					region = "lit"
				}
				resource "test_net" "x" {
					provider = test.lit
					name     = "x"
				}
			`},
			setup: func(exp *instances.Expander, d *deferring.Deferred) {
				deferInstance(d, "test_net.x", mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].lit`))
			},
			want: &proto.TypeStatus{
				ProviderSource: "registry.terraform.io/hashicorp/test",
				Type:           "test_net",
				Completeness:   proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS,
				Deferred:       []*proto.DeferredInstance{{Address: "test_net.x", ProviderInstanceId: 1}},
			},
			wantProviders: map[string]bool{`provider["registry.terraform.io/hashicorp/test"].lit`: true},
		},
		"provider ids in address order": {
			// Provider configurations the walk didn't configure get ids in
			// the order of the deferred instances' addresses.
			stage: proto.EvaluationStage_APPLY_EVALUATION_STAGE,
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "z"
					region = "z"
				}
				provider "test" {
					alias  = "y"
					region = "y"
				}
				resource "test_net" "a" {
					provider = test.z
				}
				resource "test_net" "b" {
					provider = test.y
				}
				resource "test_net" "c" {
					provider = test.z
				}
			`},
			setup: func(exp *instances.Expander, d *deferring.Deferred) {
				deferInstance(d, "test_net.c", mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].z`))
				deferInstance(d, "test_net.b", mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].y`))
				deferInstance(d, "test_net.a", mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"].z`))
			},
			want: &proto.TypeStatus{
				ProviderSource: "registry.terraform.io/hashicorp/test",
				Type:           "test_net",
				Completeness:   proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS,
				Deferred: []*proto.DeferredInstance{
					{Address: "test_net.a", ProviderInstanceId: 1},
					{Address: "test_net.b", ProviderInstanceId: 2},
					{Address: "test_net.c", ProviderInstanceId: 1},
				},
			},
			wantProviders: map[string]bool{
				`provider["registry.terraform.io/hashicorp/test"].z`: true,
				`provider["registry.terraform.io/hashicorp/test"].y`: true,
			},
		},
		"unknown keys without a deferral of their own": {
			// test_net.a's instance keys aren't known, but only test_net.b
			// was deferred, so the type isn't complete once the deferred
			// instances are planned.
			stage: proto.EvaluationStage_PLAN_EVALUATION_STAGE,
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					count = 1
				}
				resource "test_net" "b" {
				}
			`},
			setup: func(exp *instances.Expander, d *deferring.Deferred) {
				exp.SetResourceCountUnknown(addrs.RootModuleInstance, mustAbsResourceAddr("test_net.a").Resource)
				exp.SetResourceSingle(addrs.RootModuleInstance, mustAbsResourceAddr("test_net.b").Resource)
				deferInstance(d, "test_net.b", defaultProvider)
			},
			want: &proto.TypeStatus{
				ProviderSource: "registry.terraform.io/hashicorp/test",
				Type:           "test_net",
				Completeness:   proto.TypeCompleteness_INCOMPLETE_ERROR_TYPE_COMPLETENESS,
			},
		},
		"unknown keys with a deferral of their own": {
			stage: proto.EvaluationStage_PLAN_EVALUATION_STAGE,
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					count = 1
				}
				resource "test_net" "b" {
				}
			`},
			setup: func(exp *instances.Expander, d *deferring.Deferred) {
				exp.SetResourceCountUnknown(addrs.RootModuleInstance, mustAbsResourceAddr("test_net.a").Resource)
				exp.SetResourceSingle(addrs.RootModuleInstance, mustAbsResourceAddr("test_net.b").Resource)
				deferPartial(d, mustAbsResourceAddr("test_net.a").UnexpandedResource(), defaultProvider)
				deferInstance(d, "test_net.b", defaultProvider)
			},
			want: &proto.TypeStatus{
				ProviderSource: "registry.terraform.io/hashicorp/test",
				Type:           "test_net",
				Completeness:   proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS,
				Deferred: []*proto.DeferredInstance{
					{Address: "test_net.a[*]", ProviderInstanceId: 1},
					{Address: "test_net.b", ProviderInstanceId: 1},
				},
			},
			wantProviders: map[string]bool{defaultProvider.String(): true},
		},
		"unknown keys in a module instance with a deferral of their own": {
			stage: proto.EvaluationStage_PLAN_EVALUATION_STAGE,
			files: map[string]string{
				"main.tf": `
					module "m" {
						source = "./m"
						count  = 1
					}
				`,
				"m/main.tf": `
					resource "test_net" "a" {
						count = 1
					}
				`,
			},
			setup: func(exp *instances.Expander, d *deferring.Deferred) {
				exp.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "m"}, 1)
				res := mustAbsResourceAddr("module.m[0].test_net.a")
				exp.SetResourceCountUnknown(res.Module, res.Resource)
				deferPartial(d, res.UnexpandedResource(), mustProviderConfig(`module.m.provider["registry.terraform.io/hashicorp/test"]`))
			},
			want: &proto.TypeStatus{
				ProviderSource: "registry.terraform.io/hashicorp/test",
				Type:           "test_net",
				Completeness:   proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS,
				Deferred:       []*proto.DeferredInstance{{Address: "module.m[0].test_net.a[*]", ProviderInstanceId: 1}},
			},
			wantProviders: map[string]bool{`module.m.provider["registry.terraform.io/hashicorp/test"]`: true},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testModuleInline(t, test.files)
			exp := instances.NewExpander(nil)
			d := deferring.NewDeferred(true)
			test.setup(exp, d)
			ctx := &MockEvalContext{
				ConfigValue:              cfg,
				ChangesChanges:           plans.NewChanges().SyncWrapper(),
				StateState:               states.NewState().SyncWrapper(),
				DeferralsState:           d,
				InstanceExpanderExpander: exp,
			}
			ps := newPolicySubgraphForRun(&policyRunOpts{
				Stage:    test.stage,
				PlanMode: proto.PlanMode_NORMAL_PLAN_MODE,
				Schemas:  schemas,
			})
			spec := &proto.CollectionSpec{Types: []*proto.TypeSpec{relTypeSpec("test_net", "id")}}

			_, statuses, providerInstances := collectRelationshipBatch(ctx, ps, spec)

			if diff := cmp.Diff([]*proto.TypeStatus{test.want}, statuses, protocmp.Transform()); diff != "" {
				t.Errorf("wrong statuses (-want +got):\n%s", diff)
			}
			gotProviders := make(map[string]bool)
			for _, p := range providerInstances {
				gotProviders[p.ConfigAddress] = p.Known
				if p.Known && len(p.ConfigClass) != 32 {
					t.Errorf("%s is known without a class", p.ConfigAddress)
				}
			}
			if diff := cmp.Diff(test.wantProviders, gotProviders, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("wrong provider instances (-want +got):\n%s", diff)
			}
		})
	}
}
