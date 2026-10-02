// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
)

func TestContext2Plan_PolicyRelationships_origins(t *testing.T) {
	tests := map[string]struct {
		files map[string]string
		state func(*states.SyncState)
		opts  *PlanOpts
		// computedZone makes the provider plan an unknown zone when the
		// configuration has none, like providers do for optional and
		// computed attributes.
		computedZone bool
		// want are the origins of every record that has any, by address,
		// then by key path.
		want map[string]map[string][]string
	}{
		// Followed forms.
		"direct reference": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
					zone   = test_net.a.name
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v": {
					"net_id": {"test_net.a.id"},
					"zone":   {"test_net.a.name"},
				},
			},
		},
		"local value": {
			files: map[string]string{"main.tf": `
				locals {
					net_id = test_net.a.id
					alias  = local.net_id
				}
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_id = local.alias
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v": {"net_id": {"test_net.a.id"}},
			},
		},
		"object constructor": {
			files: map[string]string{"main.tf": `
				locals {
					nets = {
						a     = test_net.a.id
						"b"   = test_net.b.id
						other = "literal"
					}
				}
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "v1" {
					net_id = local.nets.a
				}
				resource "test_vm" "v2" {
					net_id = local.nets["b"]
				}
				resource "test_vm" "v3" {
					net_id = local.nets.other
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v1": {"net_id": {"test_net.a.id"}},
				"test_vm.v2": {"net_id": {"test_net.b.id"}},
			},
		},
		"template with a single interpolation and parentheses": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v1" {
					net_id = "${test_net.a.id}"
				}
				resource "test_vm" "v2" {
					net_id = (test_net.a.id)
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v1": {"net_id": {"test_net.a.id"}},
				"test_vm.v2": {"net_id": {"test_net.a.id"}},
			},
		},
		"tuple constructor": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "v" {
					net_ids = [test_net.a.id, test_net.b.id, "literal", test_net.a.id]
				}
				resource "test_vm" "w" {
					net_id = [test_net.a.id, test_net.b.id][1]
				}
			`},
			want: map[string]map[string][]string{
				// Origins are a set.
				"test_vm.v": {"net_ids": {"test_net.a.id", "test_net.b.id"}},
				"test_vm.w": {"net_id": {"test_net.b.id"}},
			},
		},
		"splat": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					count = 2
					name  = "a${count.index}"
				}
				resource "test_net" "single" {
					name = "single"
				}
				resource "test_vm" "v1" {
					net_ids = test_net.a[*].id
				}
				resource "test_vm" "v2" {
					net_ids = test_net.a.*.id
				}
				resource "test_vm" "v3" {
					net_ids = test_net.single[*].id
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v1": {"net_ids": {"test_net.a[0].id", "test_net.a[1].id"}},
				"test_vm.v2": {"net_ids": {"test_net.a[0].id", "test_net.a[1].id"}},
				"test_vm.v3": {"net_ids": {"test_net.single.id"}},
			},
		},
		"nested blocks": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "v" {
					nic {
						net_id = test_net.a.id
					}
					nic {
						net_id = test_net.b.id
					}
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v": {"nic.net_id": {"test_net.a.id", "test_net.b.id"}},
			},
		},
		"count.index": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					count = 2
					name  = "a${count.index}"
				}
				resource "test_vm" "v" {
					count  = 2
					net_id = test_net.a[count.index].id
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.v[0]": {"net_id": {"test_net.a[0].id"}},
				"test_vm.v[1]": {"net_id": {"test_net.a[1].id"}},
			},
		},
		"each.key": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					for_each = { x = "x", y = "y" }
					name     = each.value
				}
				resource "test_vm" "v" {
					for_each = { x = "1" }
					net_id   = test_net.a[each.key].id
				}
			`},
			want: map[string]map[string][]string{
				`test_vm.v["x"]`: {"net_id": {`test_net.a["x"].id`}},
			},
		},
		"each.value over a resource": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					for_each = { x = "x", y = "y" }
					name     = each.value
				}
				resource "test_vm" "v" {
					for_each = test_net.a
					net_id   = each.value.id
				}
			`},
			want: map[string]map[string][]string{
				`test_vm.v["x"]`: {"net_id": {`test_net.a["x"].id`}},
				`test_vm.v["y"]`: {"net_id": {`test_net.a["y"].id`}},
			},
		},
		"each.value over an object": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					for_each = { x = test_net.a.id }
					net_id   = each.value
				}
			`},
			want: map[string]map[string][]string{
				`test_vm.v["x"]`: {"net_id": {"test_net.a.id"}},
			},
		},
		"module input variable": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						count = 2
						name  = "a${count.index}"
					}
					module "single" {
						source = "./child"
						net_id = test_net.a[0].id
					}
					module "counted" {
						count  = 2
						source = "./child"
						net_id = test_net.a[count.index].id
					}
					module "each" {
						for_each = { x = test_net.a[1].id }
						source   = "./child"
						net_id   = each.value
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type = string
					}
					resource "test_vm" "v" {
						net_id = var.net_id
					}
				`,
			},
			want: map[string]map[string][]string{
				"module.single.test_vm.v":     {"net_id": {"test_net.a[0].id"}},
				"module.counted[0].test_vm.v": {"net_id": {"test_net.a[0].id"}},
				"module.counted[1].test_vm.v": {"net_id": {"test_net.a[1].id"}},
				`module.each["x"].test_vm.v`:  {"net_id": {"test_net.a[1].id"}},
			},
		},
		"module output": {
			files: map[string]string{
				"main.tf": `
					module "single" {
						source = "./child"
					}
					module "counted" {
						count  = 1
						source = "./child"
					}
					resource "test_vm" "v1" {
						net_id = module.single.net_id
					}
					resource "test_vm" "v2" {
						net_id = module.counted[0].net_id
					}
				`,
				"child/main.tf": `
					resource "test_net" "n" {
						name = "n"
					}
					output "net_id" {
						value = test_net.n.id
					}
				`,
			},
			want: map[string]map[string][]string{
				"test_vm.v1": {"net_id": {"module.single.test_net.n.id"}},
				"test_vm.v2": {"net_id": {"module.counted[0].test_net.n.id"}},
			},
		},

		// Forms that aren't followed.
		"not followed": {
			files: map[string]string{"main.tf": `
				variable "net_id" {
					type    = string
					default = "root"
				}
				resource "test_net" "a" {
					name  = "a"
					defer = false
				}
				data "test_info" "i" {
					net_id = "x"
				}
				resource "test_vm" "template" {
					net_id = "${test_net.a.id}-x"
				}
				resource "test_vm" "function" {
					net_id = lower(test_net.a.id)
				}
				resource "test_vm" "conditional" {
					net_id = true ? test_net.a.id : "x"
				}
				resource "test_vm" "for" {
					net_ids = [for n in [test_net.a] : n.id]
				}
				resource "test_vm" "data_source" {
					net_id = data.test_info.i.id
				}
				resource "test_vm" "root_variable" {
					net_id = var.net_id
				}
				resource "test_vm" "other_type" {
					net_id = test_net.a.defer
				}
			`},
			want: map[string]map[string][]string{},
		},
		"dynamic block": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					nic {
						net_id = test_net.a.id
					}
					# Even without any dynamic blocks, a dynamic block type has
					# no origins.
					dynamic "nic" {
						for_each = []
						content {
							net_id = test_net.a.id
						}
					}
				}
			`},
			want: map[string]map[string][]string{},
		},
		"list-valued reference": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "other" {
					net_ids = [test_net.a.id]
				}
				resource "test_vm" "v" {
					net_ids = test_vm.other.net_ids
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.other": {"net_ids": {"test_net.a.id"}},
			},
		},
		"module variable default": {
			files: map[string]string{
				"main.tf": `
					module "child" {
						source = "./child"
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type    = string
						default = "default"
					}
					resource "test_vm" "v" {
						net_id = var.net_id
					}
				`,
			},
			want: map[string]map[string][]string{},
		},
		"JSON configuration": {
			files: map[string]string{"main.tf.json": `{
				"resource": {
					"test_net": {"a": {"name": "a"}},
					"test_vm": {"v": {"net_id": "${test_net.a.id}"}}
				}
			}`},
			want: map[string]map[string][]string{},
		},

		// Drop rules.
		"ignore_changes": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
					nic {
						net_id = test_net.a.id
					}
					lifecycle {
						ignore_changes = [net_id]
					}
				}
				resource "test_vm" "all" {
					net_id = test_net.a.id
					lifecycle {
						ignore_changes = all
					}
				}
			`},
			state: func(s *states.SyncState) {
				for _, addr := range []string{"test_vm.v", "test_vm.all"} {
					s.SetResourceInstanceCurrent(mustResourceInstanceAddr(addr), &states.ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{"id":"old","net_id":"old","nic":[]}`),
						Status:    states.ObjectReady,
					}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
				}
			},
			want: map[string]map[string][]string{
				"test_vm.v": {"nic.net_id": {"test_net.a.id"}},
			},
		},
		"planned value is null": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
				}
				resource "test_vm" "v" {
					net_id = test_net.a.name
				}
			`},
			want: map[string]map[string][]string{},
		},
		"referenced value is null": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
				}
				resource "test_vm" "v" {
					# zone is computed, so its planned value is unknown, but the
					# provider might choose a value other than null.
					zone = test_net.a.name
				}
			`},
			computedZone: true,
			want:         map[string]map[string][]string{},
		},
		"different number of leaves": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
				}
				resource "test_vm" "v" {
					net_ids = [test_net.a.id, test_net.b.name]
				}
				resource "test_vm" "w" {
					nic {
						net_id = test_net.a.id
					}
					nic {
					}
				}
			`},
			want: map[string]map[string][]string{},
		},
		"test overrides": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						name = "a"
					}
					module "child" {
						source = "./child"
						net_id = test_net.a.id
					}
					resource "test_vm" "v" {
						net_id = module.child.net_id
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type = string
					}
					output "net_id" {
						value = var.net_id
					}
				`,
			},
			opts: &PlanOpts{
				Mode: plans.NormalMode,
				Overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
					overrides.Put(addrs.RootModuleInstance.Child("child", addrs.NoKey), &configs.Override{
						Values: cty.ObjectVal(map[string]cty.Value{"net_id": cty.StringVal("overridden")}),
					})
				}),
			},
			want: map[string]map[string][]string{},
		},
		"state records": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "targeted" {
					net_id = test_net.a.id
				}
				resource "test_vm" "untargeted" {
					net_id = test_net.a.id
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_vm.untargeted"), &states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"id":"u-id","net_id":"a-id"}`),
					Status:    states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			},
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_vm.targeted")},
			},
			want: map[string]map[string][]string{
				"test_vm.targeted": {"net_id": {"test_net.a.id"}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, test.files)
			var state *states.State
			if test.state != nil {
				state = states.BuildState(test.state)
			}
			opts := test.opts
			if opts == nil {
				opts = &PlanOpts{Mode: plans.NormalMode}
			}
			provider := relationshipsTestProvider()
			if test.computedZone {
				provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					planned, err := cty.Transform(req.ProposedNewState, func(path cty.Path, v cty.Value) (cty.Value, error) {
						if len(path) != 1 || !v.IsNull() {
							return v, nil
						}
						if step, ok := path[0].(cty.GetAttrStep); ok && (step.Name == "id" || step.Name == "zone") {
							return cty.UnknownVal(cty.String), nil
						}
						return v, nil
					})
					if err != nil {
						t.Fatal(err)
					}
					return providers.PlanResourceChangeResponse{PlannedState: planned}
				}
			}
			_, run := planRelationships(t, mod, state, opts, provider,
				relTypeSpec("test_net", "id", "name"),
				relTypeSpec("test_vm", "net_id", "net_ids", "nic.net_id", "zone"))
			run.assertRunSequence(t)

			got := make(map[string]map[string][]string)
			for addr, rec := range run.records(t) {
				if origins := relOrigins(t, rec); origins != nil {
					if rec.Source != proto.RecordSource_PLANNED_RECORD_SOURCE || rec.Attrs == nil {
						t.Errorf("%s: unexpected origins on a %s record", addr, rec.Source)
					}
					got[addr] = origins
				}
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("wrong origins (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPolicyPlannedLeaves(t *testing.T) {
	obj := func(netID cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{"net_id": netID})
	}
	nicTy := cty.Object(map[string]cty.Type{"net_id": cty.String})
	tests := map[string]struct {
		val    cty.Value
		path   []string
		want   int
		wantOK bool
	}{
		"primitive": {
			val:    obj(cty.StringVal("a")),
			path:   []string{"net_id"},
			want:   1,
			wantOK: true,
		},
		"unknown primitive": {
			val:    obj(cty.UnknownVal(cty.String)),
			path:   []string{"net_id"},
			want:   1,
			wantOK: true,
		},
		"null": {
			val:    obj(cty.NullVal(cty.String)),
			path:   []string{"net_id"},
			want:   0,
			wantOK: true,
		},
		"list of primitives": {
			val:    cty.ObjectVal(map[string]cty.Value{"ids": cty.ListVal([]cty.Value{cty.StringVal("a"), cty.UnknownVal(cty.String), cty.NullVal(cty.String)})}),
			path:   []string{"ids"},
			want:   2,
			wantOK: true,
		},
		"set of primitives": {
			val:    cty.ObjectVal(map[string]cty.Value{"ids": cty.SetVal([]cty.Value{cty.StringVal("a"), cty.StringVal("b")})}),
			path:   []string{"ids"},
			want:   2,
			wantOK: true,
		},
		"wholly unknown list": {
			val:    cty.ObjectVal(map[string]cty.Value{"ids": cty.UnknownVal(cty.List(cty.String))}),
			path:   []string{"ids"},
			wantOK: false,
		},
		"nested blocks": {
			val: cty.ObjectVal(map[string]cty.Value{"nic": cty.ListVal([]cty.Value{
				obj(cty.StringVal("a")),
				obj(cty.UnknownVal(cty.String)),
				obj(cty.NullVal(cty.String)),
			})}),
			path:   []string{"nic", "net_id"},
			want:   2,
			wantOK: true,
		},
		"unknown nested block": {
			val: cty.ObjectVal(map[string]cty.Value{"nic": cty.ListVal([]cty.Value{
				obj(cty.StringVal("a")),
				cty.UnknownVal(nicTy),
			})}),
			path:   []string{"nic", "net_id"},
			wantOK: false,
		},
		"empty nested blocks": {
			val:    cty.ObjectVal(map[string]cty.Value{"nic": cty.ListValEmpty(nicTy)}),
			path:   []string{"nic", "net_id"},
			want:   0,
			wantOK: true,
		},
		"map": {
			val:    cty.ObjectVal(map[string]cty.Value{"tags": cty.MapVal(map[string]cty.Value{"a": cty.StringVal("a")})}),
			path:   []string{"tags", "a"},
			wantOK: false,
		},
		"object at the leaf": {
			val:    cty.ObjectVal(map[string]cty.Value{"nic": obj(cty.StringVal("a"))}),
			path:   []string{"nic"},
			wantOK: false,
		},
		"missing attribute": {
			val:    obj(cty.StringVal("a")),
			path:   []string{"other"},
			wantOK: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := policyPlannedLeaves(test.val, test.path)
			if ok != test.wantOK {
				t.Fatalf("wrong ok %t, want %t", ok, test.wantOK)
			}
			if ok && got != test.want {
				t.Fatalf("wrong number of leaves %d, want %d", got, test.want)
			}
		})
	}
}

func TestPolicyIgnoresChanges(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_vm" "none" {
		}
		resource "test_vm" "some" {
			lifecycle {
				ignore_changes = [net_id, nic[0].net_id]
			}
		}
		resource "test_vm" "all" {
			lifecycle {
				ignore_changes = all
			}
		}
	`})
	got := make(map[string][]string)
	for _, name := range []string{"none", "some", "all"} {
		rc := mod.Module.ResourceByAddr(addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_vm", Name: name})
		for _, attr := range []string{"net_id", "nic", "zone"} {
			if policyIgnoresChanges(rc, attr) {
				got[name] = append(got[name], attr)
			}
		}
		sort.Strings(got[name])
	}
	want := map[string][]string{
		"some": {"net_id", "nic"},
		"all":  {"net_id", "nic", "zone"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong ignored attributes (-want +got):\n%s", diff)
	}
}

func TestRelationshipOriginLookup(t *testing.T) {
	provider := addrs.NewDefaultProvider("test")
	lookup := &relationshipOriginLookup{
		schemas: &schemarepo.Schemas{
			Providers: map[addrs.Provider]providers.ProviderSchema{
				provider: *relationshipsTestProvider().GetProviderSchemaResponse,
			},
		},
		planned: map[string]cty.Value{
			"test_net.a": cty.ObjectVal(map[string]cty.Value{
				"id":   cty.UnknownVal(cty.String),
				"name": cty.StringVal("a"),
				"tags": cty.NullVal(cty.Map(cty.String)),
			}),
			"test_vm.v": cty.ObjectVal(map[string]cty.Value{
				"zone": cty.NullVal(cty.String),
				"nic":  cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{"net_id": cty.String}))),
			}),
		},
	}

	t.Run("attrType", func(t *testing.T) {
		tests := []struct {
			resType string
			path    string
			want    cty.Type
		}{
			{"test_net", "id", cty.String},
			{"test_net", "defer", cty.Bool},
			{"test_vm", "disks", cty.NilType},          // list of objects
			{"test_vm", "disks.size", cty.NilType},     // through a list
			{"test_vm", "disks.password", cty.NilType}, // write-only, through a list
			{"test_vm", "nic", cty.NilType},            // nested blocks
			{"test_vm", "nic.net_id", cty.NilType},     // through a list
			{"test_vm", "net_ids", cty.NilType},        // list
			{"test_net", "tags", cty.NilType},          // map
			{"test_net", "token", cty.NilType},         // write-only
			{"test_net", "missing", cty.NilType},
			{"test_missing", "id", cty.NilType},
		}
		for _, test := range tests {
			ty, ok := lookup.attrType(provider, test.resType, strings.Split(test.path, "."))
			if ok != (test.want != cty.NilType) || (ok && !ty.Equals(test.want)) {
				t.Errorf("%s.%s: got %#v, %t; want %#v", test.resType, test.path, ty, ok, test.want)
			}
		}
		if _, ok := lookup.attrType(addrs.NewDefaultProvider("other"), "test_net", []string{"id"}); ok {
			t.Error("expected no type for another provider")
		}
	})

	t.Run("plannedValue", func(t *testing.T) {
		tests := []struct {
			addr   string
			path   string
			want   cty.Value
			wantOK bool
		}{
			{"test_net.a", "name", cty.StringVal("a"), true},
			{"test_net.a", "id", cty.NilVal, false}, // unknown
			{"test_net.a", "tags", cty.NullVal(cty.Map(cty.String)), true},
			{"test_net.a", "missing", cty.NilVal, false},
			{"test_vm.v", "zone", cty.NullVal(cty.String), true},
			{"test_vm.v", "nic.net_id", cty.NilVal, false},
			{"test_net.b", "id", cty.NilVal, false}, // no change
		}
		for _, test := range tests {
			got, ok := lookup.plannedValue(mustResourceInstanceAddr(test.addr), strings.Split(test.path, "."))
			if ok != test.wantOK || (ok && !got.RawEquals(test.want)) {
				t.Errorf("%s.%s: got %#v, %t; want %#v, %t", test.addr, test.path, got, ok, test.want, test.wantOK)
			}
		}
	})
}
