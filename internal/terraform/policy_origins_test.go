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
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

// The origins of contract §3.5 relate instances through unknown values at
// plan. Every test_net is new, so its id is unknown, and each test_vm relates
// to the test_net whose id its net_id, or zone, refers to only through an
// origin. A test_vm that refers to a test_net in a form that isn't followed
// is undecided, so its relation is PLAN.
func TestContext2Plan_PolicyRelationships_origins(t *testing.T) {
	tests := map[string]struct {
		files map[string]string
		state func(*states.SyncState)
		vars  InputValues
		// planFn replaces the provider's planning when set.
		planFn func(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse
		// want is the relation of each test_vm: its completeness, then the
		// test_nets related to it.
		want map[string]string
	}{
		// Followed forms.
		"resource attribute": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
				}
			`},
			want: map[string]string{"test_vm.v": "COMPLETE test_net.a"},
		},
		"literal instance keys": {
			files: map[string]string{"main.tf": `
				resource "test_net" "c" {
					count = 2
					name  = "c"
				}
				resource "test_net" "e" {
					for_each = toset(["x"])
					name     = each.key
				}
				resource "test_vm" "c" {
					net_id = test_net.c[1].id
				}
				resource "test_vm" "e" {
					net_id = test_net.e["x"].id
				}
			`},
			want: map[string]string{
				"test_vm.c": `PLAN test_net.c[1]`,
				"test_vm.e": `PLAN test_net.e["x"]`,
			},
		},
		"local values": {
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
			want: map[string]string{"test_vm.v": "COMPLETE test_net.a"},
		},
		// A data source attribute is an origin of its own, which a test_vm's
		// zone and a test_net's name can share.
		"data source attribute": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				data "test_info" "i" {
					net_id = test_net.a.id
				}
				resource "test_net" "b" {
					name = data.test_info.i.id
				}
				resource "test_vm" "v" {
					zone = data.test_info.i.id
				}
			`},
			want: map[string]string{"test_vm.v": "PLAN test_net.b"},
		},
		"module input variable": {
			files: map[string]string{
				"main.tf": `
					resource "test_net" "a" {
						name = "a"
					}
					module "m" {
						source = "./m"
						net_id = test_net.a.id
					}
				`,
				"m/main.tf": `
					variable "net_id" {
						type = string
					}
					resource "test_vm" "v" {
						net_id = var.net_id
					}
				`,
			},
			want: map[string]string{"module.m.test_vm.v": "COMPLETE test_net.a"},
		},
		"module outputs": {
			files: map[string]string{
				"main.tf": `
					module "n" {
						source = "./n"
					}
					module "k" {
						source   = "./n"
						for_each = toset(["x"])
					}
					resource "test_vm" "n" {
						net_id = module.n.id
					}
					resource "test_vm" "k" {
						net_id = module.k["x"].id
					}
				`,
				"n/main.tf": `
					resource "test_net" "a" {
						name = "a"
					}
					output "id" {
						value = test_net.a.id
					}
				`,
			},
			want: map[string]string{
				"test_vm.n": `PLAN module.n.test_net.a`,
				"test_vm.k": `PLAN module.k["x"].test_net.a`,
			},
		},

		// Forms that aren't followed.
		"not followed": {
			files: map[string]string{"main.tf": `
				variable "net_id" {
					type = string
				}
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "c" {
					count = 1
					name  = "c"
				}
				resource "test_vm" "template" {
					net_id = "${test_net.a.id}"
				}
				resource "test_vm" "function" {
					net_id = lower(test_net.a.id)
				}
				resource "test_vm" "conditional" {
					net_id = true ? test_net.a.id : "x"
				}
				resource "test_vm" "tuple" {
					net_ids = [test_net.a.id]
				}
				resource "test_vm" "splat" {
					net_ids = test_net.c[*].id
				}
				resource "test_vm" "for" {
					net_ids = [for n in test_net.c : n.id]
				}
				resource "test_vm" "count" {
					count  = 1
					net_id = test_net.c[count.index].id
				}
				resource "test_vm" "each" {
					for_each = { x = test_net.a }
					net_id   = each.value.id
				}
				resource "test_vm" "nested_block" {
					nic {
						net_id = test_net.a.id
					}
				}
				resource "test_vm" "root_variable" {
					zone = var.net_id
				}
				resource "test_net" "root_variable" {
					name = var.net_id
				}
			`},
			vars: InputValues{"net_id": &InputValue{Value: cty.UnknownVal(cty.String), SourceType: ValueFromCaller}},
			want: map[string]string{
				"test_vm.template":      "PLAN",
				"test_vm.function":      "PLAN",
				"test_vm.conditional":   "PLAN",
				"test_vm.tuple":         "PLAN",
				"test_vm.splat":         "PLAN",
				"test_vm.for":           "PLAN",
				"test_vm.count[0]":      "PLAN",
				`test_vm.each["x"]`:     "PLAN",
				"test_vm.nested_block":  "PLAN",
				"test_vm.root_variable": "PLAN",
			},
		},

		// Dropped origins.
		"ignore_changes": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "create" {
					net_id = test_net.a.id
					lifecycle {
						ignore_changes = [net_id]
					}
				}
				resource "test_vm" "update" {
					name   = "new"
					net_id = test_net.a.id
					lifecycle {
						ignore_changes = [net_id]
					}
				}
			`},
			state: func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_vm.update"), &states.ResourceInstanceObjectSrc{
					AttrsJSON: []byte(`{"id":"u-id","name":"old","net_id":"old-id"}`),
					Status:    states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			},
			// A create keeps its origins.
			want: map[string]string{
				"test_vm.create": "COMPLETE test_net.a",
				"test_vm.update": "PLAN",
			},
		},
		"planned value differs": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
				}
				resource "test_vm" "null" {
					name   = "null"
					net_id = test_net.a.id
				}
			`},
			planFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				resp := testDiffFn(req)
				if req.TypeName != "test_vm" {
					return resp
				}
				attrs := resp.PlannedState.AsValueMap()
				attrs["zone"] = cty.StringVal("z")
				if name := attrs["name"]; !name.IsNull() {
					attrs["net_id"] = cty.NullVal(cty.String)
				} else {
					attrs["net_id"] = cty.StringVal("other-id")
				}
				resp.PlannedState = cty.ObjectVal(attrs)
				resp.LegacyTypeSystem = true
				return resp
			},
			want: map[string]string{
				"test_vm.v":    "PLAN",
				"test_vm.null": "COMPLETE",
			},
		},
	}

	defs := []*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_id=id", "net_ids=id", "nic.net_id=id", "zone=name")}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, test.files)
			var state *states.State
			if test.state != nil {
				state = states.BuildState(test.state)
			}
			client := relClient(t, relRelationships(defs, relUsage("test_vm", relHop("vm_net"))))
			provider := relationshipsTestProvider()
			if test.planFn != nil {
				provider.PlanResourceChangeFn = test.planFn
			}
			vars := testInputValuesUnset(mod.Module.Variables)
			for name, v := range test.vars {
				vars[name] = v
			}
			relPlan(t, mod, state, &PlanOpts{Mode: plans.NormalMode, PolicyClient: client, SetVariables: vars}, provider)

			got := make(map[string]string)
			for _, req := range client.EvaluateRequests {
				if len(req.Relationships) != 1 {
					continue
				}
				rel := req.Relationships[0]
				line := []string{strings.TrimSuffix(rel.Completeness.String(), "_COMPLETENESS")}
				var addrs []string
				for id := range rel.Resources {
					addrs = append(addrs, relResolve(t, client, id).Metadata.Address)
				}
				sort.Strings(addrs)
				got[req.Meta.Address] = strings.Join(append(line, addrs...), " ")
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("wrong relations (-want +got):\n%s", diff)
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
