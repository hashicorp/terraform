// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"math/big"
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/marks"
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
		// planFn replaces the provider's planning when set.
		planFn func(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse
		// want are the origins of every record that has any, by address,
		// then by key path.
		want map[string]map[string][]string
		// noOrigin are the no-origin reasons of every record that has any,
		// by address, then by key path; not checked when nil.
		noOrigin map[string]map[string][]string
		// records are records that must exist, by relRecordKey.
		records []string
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
		"index into nested blocks": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "other" {
					nic {
						net_id = test_net.a.id
					}
					nic {
						net_id = test_net.b.id
					}
				}
				resource "test_vm" "v" {
					net_id  = test_vm.other.nic[1].net_id
					net_ids = [test_vm.other.nic[0].net_id, test_vm.other.nic[1].net_id]
					# A legacy index step.
					zone = test_vm.other.nic.0.net_id
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.other": {"nic.net_id": {"test_net.a.id", "test_net.b.id"}},
				"test_vm.v": {
					"net_id":  {"test_vm.other.nic[1].net_id"},
					"net_ids": {"test_vm.other.nic[0].net_id", "test_vm.other.nic[1].net_id"},
					"zone":    {"test_vm.other.nic[0].net_id"},
				},
			},
		},
		"index into a list attribute": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "other" {
					net_ids = ["x", test_net.a.id]
				}
				resource "test_vm" "v" {
					net_id = test_vm.other.net_ids[1]
				}
				resource "test_vm" "w" {
					count  = 2
					net_id = test_vm.other.net_ids[count.index]
				}
			`},
			want: map[string]map[string][]string{
				"test_vm.other": {"net_ids": {"test_net.a.id"}},
				"test_vm.v":     {"net_id": {"test_vm.other.net_ids[1]"}},
				"test_vm.w[0]":  {"net_id": {"test_vm.other.net_ids[0]"}},
				"test_vm.w[1]":  {"net_id": {"test_vm.other.net_ids[1]"}},
			},
		},
		"index into a null element": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "other" {
					net_ids = [test_net.a.id, null]
				}
				resource "test_vm" "v" {
					net_ids = [test_vm.other.net_ids[0], test_vm.other.net_ids[1]]
				}
			`},
			// The referenced null element is null in the planned value, so
			// it contributes nothing.
			want: map[string]map[string][]string{
				"test_vm.other": {"net_ids": {"test_net.a.id"}},
				"test_vm.v":     {"net_ids": {"test_vm.other.net_ids[0]"}},
			},
		},
		"index steps not followed": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
					tags = { "0" = "x" }
				}
				resource "test_vm" "other" {
					net_ids = ["x"]
					nic {
						net_id = "n"
					}
				}
				resource "test_vm" "string_key" {
					net_id = test_vm.other.net_ids["0"]
				}
				resource "test_vm" "block_string_key" {
					net_id = test_vm.other.nic["0"].net_id
				}
				resource "test_vm" "map" {
					net_id = test_net.a.tags[0]
				}
			`},
			want: map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_net.a":               {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
				"test_vm.other":            {"net_ids": {"LITERAL"}, "nic.net_id": {"LITERAL"}},
				"test_vm.string_key":       {"net_id": {"UNSUPPORTED"}},
				"test_vm.block_string_key": {"net_id": {"UNSUPPORTED"}},
				"test_vm.map":              {"net_id": {"UNSUPPORTED"}},
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
			noOrigin: map[string]map[string][]string{
				"test_net.a": {"id": {"NOT_CONFIGURED"}},
				"test_vm.v":  {"zone": {"UNSUPPORTED"}},
			},
		},
		"null leaves": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
				}
				resource "test_vm" "v" {
					net_id  = null
					net_ids = [test_net.a.id, null, test_net.b.name]
				}
				resource "test_vm" "w" {
					nic {
						net_id = test_net.a.id
					}
					nic {
					}
				}
			`},
			// The null leaves are null in the planned values, so they
			// contribute nothing, and the null key path has no entry.
			want: map[string]map[string][]string{
				"test_vm.v": {"net_ids": {"test_net.a.id"}},
				"test_vm.w": {"nic.net_id": {"test_net.a.id"}},
			},
			noOrigin: map[string]map[string][]string{
				"test_net.a": {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
				"test_net.b": {"id": {"NOT_CONFIGURED"}},
			},
		},
		"different number of leaves": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_ids = [test_net.a.id]
				}
			`},
			// A legacy provider plans a different list than the configured
			// one.
			planFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				planned, err := cty.Transform(req.ProposedNewState, func(path cty.Path, v cty.Value) (cty.Value, error) {
					if len(path) != 1 {
						return v, nil
					}
					switch path[0].(cty.GetAttrStep).Name {
					case "id":
						return cty.UnknownVal(cty.String), nil
					case "net_ids":
						if !v.IsNull() {
							return cty.ListVal([]cty.Value{cty.StringVal("x"), cty.StringVal("y")}), nil
						}
					}
					return v, nil
				})
				if err != nil {
					panic(err)
				}
				return providers.PlanResourceChangeResponse{PlannedState: planned, LegacyTypeSystem: true}
			},
			want: map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_net.a": {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
				"test_vm.v":  {"net_ids": {"PROVIDER_CHANGED"}},
			},
		},

		// No-origin reasons.
		"no_origin: literals": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "c" {
					count   = 1
					net_id  = "fixed"
					zone    = count.index
					net_ids = [test_net.a.id, "fixed"]
				}
				resource "test_vm" "e" {
					for_each = toset(["k"])
					net_id   = each.key
				}
			`},
			// A partial key path has origins and reasons.
			want: map[string]map[string][]string{
				"test_vm.c[0]": {"net_ids": {"test_net.a.id"}},
			},
			noOrigin: map[string]map[string][]string{
				"test_net.a":     {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
				"test_vm.c[0]":   {"net_id": {"LITERAL"}, "zone": {"LITERAL"}, "net_ids": {"LITERAL"}},
				`test_vm.e["k"]`: {"net_id": {"LITERAL"}},
			},
		},
		"no_origin: data sources and variables": {
			files: map[string]string{
				"main.tf": `
					variable "zone" {
						type    = string
						default = "z"
					}
					data "test_info" "i" {
						net_id = "x"
					}
					resource "test_vm" "v" {
						net_id = data.test_info.i.net_id
						zone   = var.zone
					}
					module "child" {
						source = "./child"
						net_id = var.zone
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type = string
					}
					variable "zone" {
						type    = string
						default = "d"
					}
					resource "test_vm" "v" {
						net_id = var.net_id
						zone   = var.zone
					}
				`,
			},
			want: map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_vm.v":              {"net_id": {"DATA_SOURCE"}, "zone": {"VARIABLE"}},
				"module.child.test_vm.v": {"net_id": {"VARIABLE"}, "zone": {"VARIABLE"}},
			},
		},
		"no_origin: expressions": {
			files: map[string]string{"main.tf": `
				variable "i" {
					type    = number
					default = 0
				}
				resource "test_net" "c" {
					count = 1
					name  = "c-${count.index}"
				}
				resource "test_net" "f" {
					for_each = toset(["x"])
					name     = each.key
				}
				resource "test_vm" "v" {
					net_id  = upper(test_net.c[0].id)
					zone    = test_net.c[0].name == "c-0" ? "x" : "y"
					net_ids = [for n in test_net.c : n.id]
				}
				resource "test_vm" "w" {
					net_id  = test_net.c[var.i].id
					zone    = "${test_net.f["x"].name}-${test_net.c[0].name}"
					net_ids = test_net.f[*]["x"].id
				}
			`},
			want: map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_net.c[0]":   {"id": {"NOT_CONFIGURED"}, "name": {"EXPRESSION"}},
				`test_net.f["x"]`: {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
				"test_vm.v":       {"net_id": {"EXPRESSION"}, "zone": {"EXPRESSION"}, "net_ids": {"EXPRESSION"}},
				"test_vm.w":       {"net_id": {"EXPRESSION"}, "zone": {"EXPRESSION"}, "net_ids": {"EXPRESSION"}},
			},
		},
		"no_origin: ignore_changes and computed attributes": {
			files: map[string]string{"main.tf": `
				resource "test_vm" "v" {
					net_id = "x"
					lifecycle {
						ignore_changes = [net_id]
					}
				}
				resource "test_vm" "all" {
					net_id = "x"
					lifecycle {
						ignore_changes = all
					}
				}
			`},
			computedZone: true,
			want:         map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_vm.v":   {"net_id": {"IGNORE_CHANGES"}, "zone": {"NOT_CONFIGURED"}},
				"test_vm.all": {"net_id": {"IGNORE_CHANGES"}, "zone": {"IGNORE_CHANGES"}},
			},
		},
		"no_origin: unsupported": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
					tags = { x = "y" }
				}
				resource "test_vm" "other" {
					net_ids = ["n"]
				}
				resource "test_vm" "v" {
					net_id  = test_net.a.tags["x"]
					zone    = path.module
					net_ids = test_vm.other.net_ids
				}
				resource "test_vm" "w" {
					net_id = path.root
					nic {
						net_id = test_net.a.id
					}
					dynamic "nic" {
						for_each = ["n"]
						content {
							net_id = "n"
						}
					}
				}
			`},
			want: map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_net.a":    {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
				"test_vm.other": {"net_ids": {"LITERAL"}},
				"test_vm.v":     {"net_id": {"UNSUPPORTED"}, "zone": {"UNSUPPORTED"}, "net_ids": {"UNSUPPORTED"}},
				"test_vm.w":     {"net_id": {"UNSUPPORTED"}, "nic.net_id": {"UNSUPPORTED"}},
			},
		},
		"no_origin: JSON and override files": {
			files: map[string]string{
				"main.tf": `
					resource "test_vm" "o" {
						net_id = "x"
					}
				`,
				"main_override.tf": `
					resource "test_vm" "o" {
						zone = "z"
					}
				`,
				"vm.tf.json": `{"resource": {"test_vm": {"j": {"net_id": "x"}}}}`,
			},
			want: map[string]map[string][]string{},
			noOrigin: map[string]map[string][]string{
				"test_vm.o": {"net_id": {"UNSUPPORTED"}, "zone": {"UNSUPPORTED"}},
				"test_vm.j": {"net_id": {"UNSUPPORTED"}},
			},
		},
		// Test overrides. The outputs of an overridden module come from the
		// override, so they have no origins, but overridden resources still
		// plan their configured attributes from their configuration.
		"overridden module": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						name = "a"
					}
					module "over" {
						source = "./child"
						net_id = test_net.a.id
					}
					module "kept" {
						source = "./child"
						net_id = test_net.a.id
					}
					module "fed" {
						source = "./child"
						net_id = module.over.net_id
					}
					locals {
						over = module.over
						kept = module.kept
					}
					resource "test_vm" "over" {
						net_id = module.over.net_id
					}
					resource "test_vm" "kept" {
						net_id = module.kept.net_id
					}
					resource "test_vm" "over_call" {
						net_id = local.over.net_id
					}
					resource "test_vm" "kept_call" {
						net_id = local.kept.net_id
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type = string
					}
					resource "test_vm" "inner" {
						net_id = var.net_id
					}
					output "net_id" {
						value = var.net_id
					}
				`,
			},
			opts: &PlanOpts{
				Mode: plans.NormalMode,
				Overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
					overrides.Put(addrs.RootModuleInstance.Child("over", addrs.NoKey), &configs.Override{
						Values: cty.ObjectVal(map[string]cty.Value{"net_id": cty.StringVal("overridden")}),
					})
				}),
			},
			want: map[string]map[string][]string{
				"test_vm.kept":              {"net_id": {"test_net.a.id"}},
				"test_vm.kept_call":         {"net_id": {"test_net.a.id"}},
				"module.kept.test_vm.inner": {"net_id": {"test_net.a.id"}},
			},
		},
		"overridden module instance": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						count = 2
						name  = "a${count.index}"
					}
					module "child" {
						count  = 2
						source = "./child"
						net_id = test_net.a[count.index].id
					}
					resource "test_vm" "v0" {
						net_id = module.child[0].net_id
					}
					resource "test_vm" "v1" {
						net_id = module.child[1].net_id
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type = string
					}
					resource "test_vm" "inner" {
						net_id = var.net_id
					}
					output "net_id" {
						value = var.net_id
					}
				`,
			},
			opts: &PlanOpts{
				Mode: plans.NormalMode,
				Overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
					overrides.Put(addrs.RootModuleInstance.Child("child", addrs.IntKey(0)), &configs.Override{
						Values: cty.ObjectVal(map[string]cty.Value{"net_id": cty.StringVal("overridden")}),
					})
				}),
			},
			want: map[string]map[string][]string{
				"test_vm.v1":                    {"net_id": {"test_net.a[1].id"}},
				"module.child[1].test_vm.inner": {"net_id": {"test_net.a[1].id"}},
			},
		},
		"module overridden through its containing call": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						count = 2
						name  = "a${count.index}"
					}
					module "child" {
						count  = 2
						source = "./child"
						net_id = test_net.a[count.index].id
					}
					module "kept" {
						source = "./child"
						net_id = test_net.a[0].id
					}
					resource "test_vm" "v0" {
						net_id = module.child[0].net_id
					}
					resource "test_vm" "all" {
						net_ids = module.child[*].net_id
					}
					resource "test_vm" "kept" {
						net_id = module.kept.net_id
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
			want: map[string]map[string][]string{
				"test_vm.kept": {"net_id": {"test_net.a[0].id"}},
			},
		},
		"overridden nested module": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						name = "a"
					}
					module "parent" {
						source = "./parent"
						net_id = test_net.a.id
					}
					resource "test_vm" "over" {
						net_id = module.parent.over_net_id
					}
					resource "test_vm" "kept" {
						net_id = module.parent.kept_net_id
					}
				`,
				"parent/main.tf": `
					variable "net_id" {
						type = string
					}
					module "over" {
						source = "../child"
						net_id = var.net_id
					}
					module "kept" {
						source = "../child"
						net_id = var.net_id
					}
					output "over_net_id" {
						value = module.over.net_id
					}
					output "kept_net_id" {
						value = module.kept.net_id
					}
				`,
				"child/main.tf": `
					variable "net_id" {
						type = string
					}
					resource "test_vm" "inner" {
						net_id = var.net_id
					}
					output "net_id" {
						value = var.net_id
					}
				`,
			},
			opts: &PlanOpts{
				Mode: plans.NormalMode,
				Overrides: mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
					overrides.Put(addrs.RootModuleInstance.Child("parent", addrs.NoKey).Child("over", addrs.NoKey), &configs.Override{
						Values: cty.ObjectVal(map[string]cty.Value{"net_id": cty.StringVal("overridden")}),
					})
				}),
			},
			want: map[string]map[string][]string{
				"test_vm.kept": {"net_id": {"test_net.a.id"}},
				"module.parent.module.kept.test_vm.inner": {"net_id": {"test_net.a.id"}},
			},
		},
		"overridden resources": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
					zone   = test_net.a.name
				}
				resource "test_vm" "w" {
					net_id = test_net.b.id
					zone   = test_net.b.name
				}
				resource "test_vm" "over" {
					net_id = test_net.a.id
				}
			`},
			opts: &PlanOpts{
				Mode: plans.NormalMode,
				// test_net.b is overridden like a mock_provider override.
				Overrides: mocking.OverridesForTesting(func(overrides map[addrs.RootProviderConfig]addrs.Map[addrs.Targetable, *configs.Override]) {
					m := addrs.MakeMap[addrs.Targetable, *configs.Override]()
					m.Put(mustResourceInstanceAddr("test_net.b").ContainingResource(), &configs.Override{
						Values:     cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("b-over")}),
						UseForPlan: true,
					})
					overrides[addrs.RootProviderConfig{Provider: addrs.NewDefaultProvider("test")}] = m
				}, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
					overrides.Put(mustResourceInstanceAddr("test_net.a").ContainingResource(), &configs.Override{
						Values:     cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("a-over")}),
						UseForPlan: true,
					})
					overrides.Put(mustResourceInstanceAddr("test_vm.over"), &configs.Override{
						Values: cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("vm-over")}),
					})
				}),
			},
			want: map[string]map[string][]string{
				"test_vm.v": {
					"net_id": {"test_net.a.id"},
					"zone":   {"test_net.a.name"},
				},
				"test_vm.w": {
					"net_id": {"test_net.b.id"},
					"zone":   {"test_net.b.name"},
				},
				"test_vm.over": {"net_id": {"test_net.a.id"}},
			},
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
				s.SetResourceInstanceDeposed(mustResourceInstanceAddr("test_vm.targeted"), "00000001", &states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"id":"d-id","net_id":"a-id"}`),
					Status:    states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			},
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_vm.targeted")},
			},
			// Neither the state record of test_vm.untargeted nor the record
			// of the deposed object of test_vm.targeted have origins or
			// reasons.
			want: map[string]map[string][]string{
				"test_vm.targeted": {"net_id": {"test_net.a.id"}},
			},
			noOrigin: map[string]map[string][]string{
				"test_net.a": {"id": {"NOT_CONFIGURED"}, "name": {"LITERAL"}},
			},
			records: []string{"test_vm.untargeted", "test_vm.targeted deposed 00000001"},
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
			provider.PlanResourceChangeFn = test.planFn
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
			gotNoOrigin := make(map[string]map[string][]string)
			records := run.records(t)
			for _, key := range test.records {
				if _, ok := records[key]; !ok {
					t.Errorf("missing record %s", key)
				}
			}
			for addr, rec := range records {
				if len(rec.Origins) > 0 && (rec.Source != proto.RecordSource_PLANNED_RECORD_SOURCE || rec.Attrs == nil || rec.DeposedKey != "") {
					t.Errorf("%s: unexpected origins on a %s record", addr, rec.Source)
				}
				if origins := relOrigins(t, rec); origins != nil {
					got[addr] = origins
				}
				if reasons := relNoOrigins(rec); reasons != nil {
					gotNoOrigin[addr] = reasons
				}
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("wrong origins (-want +got):\n%s", diff)
			}
			if test.noOrigin != nil {
				if diff := cmp.Diff(test.noOrigin, gotNoOrigin); diff != "" {
					t.Fatalf("wrong no-origin reasons (-want +got):\n%s", diff)
				}
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

// TestOriginsForOverriddenModules checks each place that excludes overridden
// modules on its own, because in a walk the outputs of an overridden module
// are the only way to reach into it.
func TestOriginsForOverriddenModules(t *testing.T) {
	cfg := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_net" "a" {
				name = "a"
			}
			module "parent" {
				source = "./parent"
				net_id = test_net.a.id
			}
		`,
		"parent/main.tf": `
			variable "net_id" {
				type = string
			}
			module "child" {
				source = "../child"
				net_id = var.net_id
			}
		`,
		"child/main.tf": `
			variable "net_id" {
				type = string
			}
			resource "test_net" "n" {
				name = "n"
			}
			resource "test_vm" "v" {
				net_id = var.net_id
			}
			output "net_id" {
				value = test_net.n.id
			}
		`,
	})
	parent := addrs.RootModuleInstance.Child("parent", addrs.NoKey)
	child := parent.Child("child", addrs.NoKey)
	childNet := mustResourceInstanceAddr("module.parent.module.child.test_net.n")
	childVM := mustResourceInstanceAddr("module.parent.module.child.test_vm.v")

	exp := instances.NewExpander(nil)
	exp.SetResourceSingle(addrs.RootModuleInstance, mustResourceInstanceAddr("test_net.a").Resource.Resource)
	exp.SetModuleSingle(addrs.RootModuleInstance, addrs.ModuleCall{Name: "parent"})
	exp.SetModuleSingle(parent, addrs.ModuleCall{Name: "child"})
	exp.SetResourceSingle(child, childNet.Resource.Resource)
	exp.SetResourceSingle(child, childVM.Resource.Resource)

	lookup := &relationshipOriginLookup{
		schemas: &schemarepo.Schemas{
			Providers: map[addrs.Provider]providers.ProviderSchema{
				addrs.NewDefaultProvider("test"): *relationshipsTestProvider().GetProviderSchemaResponse,
			},
		},
		planned: map[string]cty.Value{},
	}
	overrideModule := func(mod addrs.ModuleInstance) *mocking.Overrides {
		return mocking.OverridesForTesting(nil, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
			overrides.Put(mod, &configs.Override{Values: cty.EmptyObjectVal})
		})
	}

	tests := map[string]struct {
		overrides *mocking.Overrides
		// overridden is whether the child module is overridden.
		overridden bool
	}{
		"no overrides": {
			overrides: nil,
		},
		"empty overrides": {
			overrides: mocking.OverridesForTesting(nil, nil),
		},
		"overridden resources": {
			overrides: mocking.OverridesForTesting(func(overrides map[addrs.RootProviderConfig]addrs.Map[addrs.Targetable, *configs.Override]) {
				m := addrs.MakeMap[addrs.Targetable, *configs.Override]()
				m.Put(childNet.ContainingResource(), &configs.Override{Values: cty.EmptyObjectVal})
				overrides[addrs.RootProviderConfig{Provider: addrs.NewDefaultProvider("test")}] = m
			}, func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(childVM, &configs.Override{Values: cty.EmptyObjectVal})
			}),
		},
		"overridden sibling": {
			overrides: overrideModule(parent.Child("sibling", addrs.NoKey)),
		},
		"overridden module": {
			overrides:  overrideModule(child),
			overridden: true,
		},
		"overridden parent": {
			overrides:  overrideModule(parent),
			overridden: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			overridden := policyOverriddenModules(test.overrides)
			if test.overrides.Empty() != (overridden == nil) {
				t.Fatalf("expected a predicate only with overrides")
			}

			// A record in the module.
			got := originsFor(cfg, exp, overridden, childVM, cty.ObjectVal(map[string]cty.Value{"net_id": cty.UnknownVal(cty.String)}), [][]string{{"net_id"}}, lookup)
			if len(got) != 1 {
				t.Fatalf("expected one entry for the key path, got %v", got)
			}
			if test.overridden {
				if len(got[0].Origins) != 0 || len(got[0].NoOrigin) != 1 || got[0].NoOrigin[0] != proto.NoOriginReason_UNSUPPORTED_NO_ORIGIN_REASON {
					t.Errorf("expected no origins and the reason UNSUPPORTED for a record in an overridden module, got %v", got)
				}
			} else if len(got[0].Origins) != 1 || len(got[0].NoOrigin) != 0 {
				t.Errorf("expected the origin of the record, got %v", got)
			}

			e := &originEval{cfg: cfg, exp: exp, overridden: overridden}

			// The module's outputs.
			_, opaque := e.moduleInstance(child, cfg.DescendantForInstance(child)).(symOpaque)
			if opaque != test.overridden {
				t.Errorf("wrong outputs of the module: opaque = %t, want %t", opaque, test.overridden)
			}

			// A reference to a resource in the module.
			allowed := e.originAllowed(symRef{addr: childNet, path: testOriginPath(t, "id")}, cty.String, lookup)
			if allowed == test.overridden {
				t.Errorf("wrong origin into the module: allowed = %t, want %t", allowed, !test.overridden)
			}
		})
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
			"test_vm.w": cty.ObjectVal(map[string]cty.Value{
				"net_ids": cty.ListVal([]cty.Value{cty.StringVal("a"), cty.NullVal(cty.String), cty.UnknownVal(cty.String)}),
				"nic": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"net_id": cty.StringVal("n"),
				})}),
				"disks": cty.TupleVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"size": cty.NumberIntVal(1),
				})}),
				"zone": cty.SetVal([]cty.Value{cty.StringVal("z")}),
				"name": cty.MapVal(map[string]cty.Value{"0": cty.StringVal("m")}),
			}),
			"test_vm.null": cty.ObjectVal(map[string]cty.Value{
				"net_ids": cty.NullVal(cty.List(cty.String)),
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

			// Index steps.
			{"test_vm", "nic[0].net_id", cty.String},
			{"test_vm", "nic[0].key", cty.NilType}, // write-only
			{"test_vm", "nic[0]", cty.NilType},     // object
			{"test_vm", "disks[1].size", cty.Number},
			{"test_vm", "disks[0].password", cty.NilType}, // write-only
			{"test_vm", "net_ids[2]", cty.String},
			{"test_vm", "net_ids[0].x", cty.NilType},
			{"test_net", "tags[0]", cty.NilType}, // map
			{"test_net", "name[0]", cty.NilType}, // primitive
		}
		for _, test := range tests {
			ty, ok := lookup.attrType(provider, test.resType, testOriginPath(t, test.path))
			if ok != (test.want != cty.NilType) || (ok && !ty.Equals(test.want)) {
				t.Errorf("%s.%s: got %#v, %t; want %#v", test.resType, test.path, ty, ok, test.want)
			}
		}
		if _, ok := lookup.attrType(addrs.NewDefaultProvider("other"), "test_net", testOriginPath(t, "id")); ok {
			t.Error("expected no type for another provider")
		}
		if _, ok := lookup.attrType(provider, "test_vm", []originStep{{index: 0}, {name: "net_id"}}); ok {
			t.Error("expected no type for a path starting with an index step")
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

			// Index steps.
			{"test_vm.w", "net_ids[0]", cty.StringVal("a"), true},
			{"test_vm.w", "net_ids[1]", cty.NullVal(cty.String), true},
			{"test_vm.w", "net_ids[2]", cty.NilVal, false}, // unknown
			{"test_vm.w", "nic[0].net_id", cty.StringVal("n"), true},
			{"test_vm.w", "disks[0].size", cty.NumberIntVal(1), true}, // tuple
			{"test_vm.v", "nic[0].net_id", cty.NilVal, false},         // unknown list
			{"test_vm.null", "net_ids[0]", cty.NullVal(cty.List(cty.String)), true},
			{"test_vm.w", "zone[0]", cty.NilVal, false}, // set
			{"test_vm.w", "name[0]", cty.NilVal, false}, // map
			{"test_vm.w", "nic.net_id", cty.NilVal, false},

			// There's no element out of range of a known list or tuple.
			{"test_vm.w", "net_ids[3]", cty.NullVal(cty.DynamicPseudoType), true},
			{"test_vm.w", "nic[1].net_id", cty.NullVal(cty.DynamicPseudoType), true},
			{"test_vm.w", "disks[1].size", cty.NullVal(cty.DynamicPseudoType), true},
		}
		for _, test := range tests {
			got, ok := lookup.plannedValue(mustResourceInstanceAddr(test.addr), testOriginPath(t, test.path))
			if ok != test.wantOK || (ok && !got.RawEquals(test.want)) {
				t.Errorf("%s.%s: got %#v, %t; want %#v, %t", test.addr, test.path, got, ok, test.want, test.wantOK)
			}
		}
	})
}

func TestOriginEvalIndex(t *testing.T) {
	addr := mustResourceInstanceAddr("test_vm.v")
	ref := symRef{addr: addr, path: []originStep{{name: "net_ids"}}}
	tooLarge, _ := new(big.Float).SetString("1e30")
	tests := map[string]struct {
		base originSym
		key  cty.Value
		want originSym
	}{
		"whole number": {ref, cty.NumberIntVal(1), symRef{addr: addr, path: []originStep{{name: "net_ids"}, {index: 1}}}},
		"zero":         {ref, cty.Zero, symRef{addr: addr, path: []originStep{{name: "net_ids"}, {index: 0}}}},
		"nested": {
			symRef{addr: addr, path: []originStep{{name: "m"}, {index: 2}}},
			cty.NumberIntVal(3),
			symRef{addr: addr, path: []originStep{{name: "m"}, {index: 2}, {index: 3}}},
		},
		"negative":       {ref, cty.NumberIntVal(-1), opaqueUnsupported},
		"fractional":     {ref, cty.NumberFloatVal(1.5), opaqueUnsupported},
		"too large":      {ref, cty.NumberVal(tooLarge), opaqueUnsupported},
		"string":         {ref, cty.StringVal("1"), opaqueUnsupported},
		"unknown":        {ref, cty.UnknownVal(cty.Number), opaqueExpression},
		"null":           {ref, cty.NullVal(cty.Number), opaqueExpression},
		"marked":         {ref, cty.NumberIntVal(1).Mark(marks.Sensitive), opaqueExpression},
		"whole resource": {symRef{addr: addr}, cty.Zero, opaqueUnsupported},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			e := &originEval{}
			got := e.index(test.base, symLit{val: test.key})
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestOriginEvalOriginAllowed_indexSteps(t *testing.T) {
	cfg := testModuleInline(t, map[string]string{
		"main.tf": `
			resource "test_net" "a" {
			}
			resource "test_vm" "v" {
			}
		`,
	})
	lookup := &relationshipOriginLookup{
		schemas: &schemarepo.Schemas{
			Providers: map[addrs.Provider]providers.ProviderSchema{
				addrs.NewDefaultProvider("test"): *relationshipsTestProvider().GetProviderSchemaResponse,
			},
		},
		planned: map[string]cty.Value{
			"test_net.a": cty.ObjectVal(map[string]cty.Value{
				"tags": cty.MapVal(map[string]cty.Value{"0": cty.StringVal("x")}),
			}),
			"test_vm.v": cty.ObjectVal(map[string]cty.Value{
				"net_ids": cty.ListVal([]cty.Value{cty.StringVal("a"), cty.NullVal(cty.String)}),
				"nic":     cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{"net_id": cty.String}))),
			}),
		},
	}
	tests := []struct {
		addr string
		path string
		want bool
	}{
		{"test_vm.v", "net_ids[0]", true},
		{"test_vm.v", "net_ids[1]", false}, // null
		{"test_vm.v", "net_ids[2]", false}, // out of range
		{"test_vm.v", "nic[3].net_id", true},
		{"test_vm.v", "nic[0].key", false}, // write-only
		{"test_net.a", "tags[0]", false},   // map
	}
	e := &originEval{cfg: cfg, exp: instances.NewExpander(nil)}
	for _, test := range tests {
		ref := symRef{addr: mustResourceInstanceAddr(test.addr), path: testOriginPath(t, test.path)}
		if got := e.originAllowed(ref, cty.String, lookup); got != test.want {
			t.Errorf("%s.%s: allowed = %t, want %t", test.addr, test.path, got, test.want)
		}
	}
}

func TestOriginPath(t *testing.T) {
	paths := []string{"nic[10].net_id", "net_ids[1]", "nic[2].net_id", "name", "nic[2].id", "net_ids[0]", "m[1][0]"}
	wantOrder := []string{"m[1][0]", "name", "net_ids[0]", "net_ids[1]", "nic[2].id", "nic[2].net_id", "nic[10].net_id"}

	var parsed [][]originStep
	for _, p := range paths {
		path := testOriginPath(t, p)
		parsed = append(parsed, path)
		if got := originPathString(path); got != p {
			t.Errorf("wrong id for %s: %s", p, got)
		}
		if got := relPathString(policyOriginPath(path)); got != p {
			t.Errorf("wrong wire path for %s: %s", p, got)
		}
	}
	sort.Slice(parsed, func(i, j int) bool { return originPathLess(parsed[i], parsed[j]) })
	var gotOrder []string
	for _, path := range parsed {
		gotOrder = append(gotOrder, originPathString(path))
	}
	if diff := cmp.Diff(wantOrder, gotOrder); diff != "" {
		t.Errorf("wrong order (-want +got):\n%s", diff)
	}

	steps := policyOriginPath(testOriginPath(t, "nic[1].net_id")).GetSteps()
	if len(steps) != 3 || steps[0].GetAttributeName() != "nic" || steps[2].GetAttributeName() != "net_id" {
		t.Fatalf("wrong wire path: %v", steps)
	}
	if step, ok := steps[1].GetSelector().(*proto.AttributePath_Step_ElementKeyInt); !ok || step.ElementKeyInt != 1 {
		t.Errorf("expected an element_key_int step 1, got %v", steps[1])
	}
}

// testOriginPath parses an origin path such as "nic[0].net_id".
func testOriginPath(t *testing.T, s string) []originStep {
	t.Helper()
	trav, diags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("invalid origin path %q: %s", s, diags.Error())
	}
	var path []originStep
	for _, step := range trav {
		switch step := step.(type) {
		case hcl.TraverseRoot:
			path = append(path, originStep{name: step.Name})
		case hcl.TraverseAttr:
			path = append(path, originStep{name: step.Name})
		case hcl.TraverseIndex:
			i, acc := step.Key.AsBigFloat().Int64()
			if acc != big.Exact {
				t.Fatalf("invalid index in origin path %q", s)
			}
			path = append(path, originStep{index: i})
		default:
			t.Fatalf("invalid step in origin path %q", s)
		}
	}
	return path
}
