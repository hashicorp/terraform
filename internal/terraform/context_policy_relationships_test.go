// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const relTestSource = "registry.terraform.io/hashicorp/test"

// relDefinition returns a definition of the test provider. Each alternative
// is a list of keys joined by "&", each key a key path and a value path
// joined by "=", like "net_id=id&zone=name".
func relDefinition(name, keyType, valueType string, alts ...string) *proto.RelationshipDefinition {
	def := &proto.RelationshipDefinition{
		ProviderSource: relTestSource,
		Name:           name,
		KeyResource:    keyType,
		ValueResource:  valueType,
		DeclRange: &proto.Range{
			Filename: "relationships.tfpolicy.hcl",
			Start:    &proto.Position{Line: 1, Column: 1},
			End:      &proto.Position{Line: 1, Column: 20, Byte: 19},
		},
	}
	for _, alt := range alts {
		var keys []*proto.MatchKey
		for _, key := range strings.Split(alt, "&") {
			kp, vp, _ := strings.Cut(key, "=")
			keys = append(keys, &proto.MatchKey{KeyPath: relAttrPath(kp), ValuePath: relAttrPath(vp)})
		}
		def.Match = append(def.Match, &proto.MatchAlternative{Keys: keys})
	}
	return def
}

func relAttrPath(dotted string) *proto.AttributePath {
	path := &proto.AttributePath{}
	for _, name := range strings.Split(dotted, ".") {
		path.Steps = append(path.Steps, &proto.AttributePath_Step{
			Selector: &proto.AttributePath_Step_AttributeName{AttributeName: name},
		})
	}
	return path
}

func relUsage(typ string, hops ...*proto.RelationshipHop) *proto.RelationshipUsage {
	return &proto.RelationshipUsage{ProviderSource: relTestSource, Type: typ, Hops: hops}
}

func relHop(relationship string, hops ...*proto.RelationshipHop) *proto.RelationshipHop {
	return &proto.RelationshipHop{Relationship: relationship, Hops: hops}
}

// relVMNet relates a test_vm to the test_net whose id is its net_id, and both
// types use it.
func relVMNet() policy.Relationships {
	return policy.Relationships{
		Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_id=id")},
		Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net")), relUsage("test_net", relHop("vm_net"))},
	}
}

func relClient(t *testing.T, rels policy.Relationships) *policy.MockClient {
	client := policy.NewTestMockClient(t)
	client.RelationshipsResponse = rels
	return client
}

func relContext(t *testing.T, provider *testing_provider.MockProvider) *Context {
	return testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
		},
	})
}

// relPlan plans, and fails the test if planning fails.
func relPlan(t *testing.T, mod *configs.Config, state *states.State, opts *PlanOpts, provider *testing_provider.MockProvider) *plans.Plan {
	t.Helper()
	if opts.SetVariables == nil {
		opts.SetVariables = testInputValuesUnset(mod.Module.Variables)
	}
	plan, diags := relContext(t, provider).Plan(mod, relState(state), opts)
	tfdiags.AssertNoErrors(t, diags)
	return plan
}

func relState(state *states.State) *states.State {
	if state == nil {
		return states.NewState()
	}
	return state
}

func relNetState(s *states.SyncState, addr, attrsJSON string) {
	s.SetResourceInstanceCurrent(mustResourceInstanceAddr(addr), &states.ResourceInstanceObjectSrc{
		AttrsJSON: []byte(attrsJSON),
		Status:    states.ObjectReady,
	}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
}

func relNetDeposedState(s *states.SyncState, addr string, key states.DeposedKey, attrsJSON string) {
	s.SetResourceInstanceDeposed(mustResourceInstanceAddr(addr), key, &states.ResourceInstanceObjectSrc{
		AttrsJSON: []byte(attrsJSON),
		Status:    states.ObjectReady,
	}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
}

// relTrees returns the related members of every evaluated subject that has
// any, rendered by relTree, by subject: its address, " deposed KEY" for a
// deposed object, and its operation.
func relTrees(t *testing.T, client *policy.MockClient) map[string][]string {
	t.Helper()
	ret := make(map[string][]string)
	for _, req := range client.EvaluateRequests {
		if req.Related == nil {
			continue
		}
		key := req.Meta.Address
		if req.Meta.DeposedKey != "" {
			key += " deposed " + req.Meta.DeposedKey
		}
		key += " " + req.Meta.Operation.String()
		if _, ok := ret[key]; ok {
			t.Fatalf("subject %s evaluated twice", key)
		}
		ret[key] = relTree(req.Related)
	}
	return ret
}

// relTree renders related members, one line per hop and member, nested
// members indented: "RELATIONSHIP COMPLETENESS [DEFERRED...]" for a hop, and
// "ADDRESS [deposed KEY] [old] OPERATION [attrs] [prior]" for a member, with
// attrs and prior when it has attrs and prior attrs.
func relTree(related []*proto.RelatedMembers) []string {
	var lines []string
	var walk func([]*proto.RelatedMembers, string)
	walk = func(hops []*proto.RelatedMembers, indent string) {
		for _, hop := range hops {
			line := []string{hop.Relationship, strings.TrimSuffix(hop.Completeness.String(), "_COMPLETENESS")}
			lines = append(lines, indent+strings.Join(append(line, hop.Deferred...), " "))
			for _, m := range hop.Members {
				line := []string{m.Address}
				if m.DeposedKey != "" {
					line = append(line, "deposed", m.DeposedKey)
				}
				if m.OldObject {
					line = append(line, "old")
				}
				line = append(line, m.Operation.String())
				if m.Attrs != nil {
					line = append(line, "attrs")
				}
				if m.PriorAttrs != nil {
					line = append(line, "prior")
				}
				lines = append(lines, indent+"  "+strings.Join(line, " "))
				walk(m.Related, indent+"    ")
			}
		}
	}
	walk(related, "")
	return lines
}

// relMembers returns the members of every hop of every subject, by subject
// key like relTrees, then by member address.
func relMembers(t *testing.T, client *policy.MockClient) map[string]map[string]*proto.RelatedMember {
	ret := make(map[string]map[string]*proto.RelatedMember)
	for _, req := range client.EvaluateRequests {
		key := req.Meta.Address + " " + req.Meta.Operation.String()
		for _, hop := range req.Related {
			for _, m := range hop.Members {
				if ret[key] == nil {
					ret[key] = make(map[string]*proto.RelatedMember)
				}
				ret[key][m.Address] = m
			}
		}
	}
	return ret
}

// relAttr returns an attribute of encoded attrs, as a string.
func relAttr(t *testing.T, attrs *proto.ResourceAttributes, name string) string {
	t.Helper()
	val, err := ctymsgpack.Unmarshal(attrs.GetRaw(), cty.DynamicPseudoType)
	if err != nil {
		t.Fatalf("invalid attrs: %s", err)
	}
	attr := val.GetAttr(name)
	switch {
	case !attr.IsKnown():
		return "(unknown)"
	case attr.IsNull():
		return "(null)"
	default:
		return attr.AsString()
	}
}

func TestContext2Plan_PolicyRelationships_validation(t *testing.T) {
	tests := map[string]struct {
		def *proto.RelationshipDefinition
		// detail is the detail of the expected error; none when empty.
		detail string
	}{
		"data source": {
			def:    relDefinition("r", "test_info", "test_net", "net_id=id"),
			detail: "test_info is a data source type of registry.terraform.io/hashicorp/test v1.2.3, not a managed resource type. Relationships relate managed resources only.",
		},
		"no such type": {
			def:    relDefinition("r", "test_vm", "test_nope", "net_id=id"),
			detail: "registry.terraform.io/hashicorp/test v1.2.3 has no resource type test_nope.",
		},
		"no such path": {
			def:    relDefinition("r", "test_vm", "test_net", "nope=id"),
			detail: "`nope` of test_vm doesn't exist in the schema of registry.terraform.io/hashicorp/test v1.2.3.",
		},
		"no such nested path": {
			def:    relDefinition("r", "test_vm", "test_net", "nic.nope=id"),
			detail: "`nic.nope` of test_vm doesn't exist in the schema of registry.terraform.io/hashicorp/test v1.2.3.",
		},
		"steps through a map": {
			def:    relDefinition("r", "test_vm", "test_net", "net_id=tags.x"),
			detail: "`tags.x` of test_net steps through `tags`, of type map of string, in the schema of registry.terraform.io/hashicorp/test v1.2.3. A key can't read a map.",
		},
		"ends in a map": {
			def:    relDefinition("r", "test_vm", "test_net", "net_id=tags"),
			detail: "`tags` of test_net has type map of string in the schema of registry.terraform.io/hashicorp/test v1.2.3. A key can't read a map.",
		},
		"ends in nested objects": {
			def:    relDefinition("r", "test_vm", "test_net", "disks=id"),
			detail: "`disks` of test_vm has type list of object in the schema of registry.terraform.io/hashicorp/test v1.2.3. A key must read a primitive or a collection of primitives.",
		},
		"ends in a nested block": {
			def:    relDefinition("r", "test_vm", "test_net", "nic=id"),
			detail: "`nic` of test_vm has type list of object in the schema of registry.terraform.io/hashicorp/test v1.2.3. A key must read a primitive or a collection of primitives.",
		},
		"write-only": {
			def:    relDefinition("r", "test_vm", "test_net", "net_id=token"),
			detail: "`token` of test_net is write-only, of type string, in the schema of registry.terraform.io/hashicorp/test v1.2.3, so it's always null in the plan and the state.",
		},
		"write-only in a nested block": {
			def:    relDefinition("r", "test_vm", "test_net", "nic.key=id"),
			detail: "`nic.key` of test_vm is write-only, of type string, in the schema of registry.terraform.io/hashicorp/test v1.2.3, so it's always null in the plan and the state.",
		},
		"different types": {
			def:    relDefinition("r", "test_vm", "test_net", "disks.size=id"),
			detail: "`disks.size` of test_vm is number, but `id` of test_net is string, in the schema of registry.terraform.io/hashicorp/test v1.2.3. Both paths of a key must read the same primitive type.",
		},
		"valid nested paths": {
			def: relDefinition("r", "test_vm", "test_net", "nic.net_id=id", "net_ids=id&net_set=name"),
		},
		"another provider": {
			def: &proto.RelationshipDefinition{
				ProviderSource: "registry.terraform.io/hashicorp/other",
				Name:           "r",
				KeyResource:    "other_nope",
				ValueResource:  "other_nope",
				Match:          []*proto.MatchAlternative{{Keys: []*proto.MatchKey{{KeyPath: relAttrPath("nope"), ValuePath: relAttrPath("nope")}}}},
			},
		},
	}
	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_net" "a" {
			name = "a"
		}
	`})
	locks := map[addrs.Provider]*depsfile.ProviderLock{
		addrs.NewDefaultProvider("test"): depsfile.NewProviderLock(addrs.NewDefaultProvider("test"), getproviders.MustParseVersion("1.2.3"), nil, nil),
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rels := policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{test.def},
				Usages:      []*proto.RelationshipUsage{relUsage("test_net", relHop("r"))},
			}
			client := relClient(t, rels)
			provider := relationshipsTestProvider()
			_, diags := relContext(t, provider).Plan(mod, states.NewState(), &PlanOpts{
				Mode:          plans.NormalMode,
				PolicyClient:  client,
				ProviderLocks: locks,
			})
			if test.detail == "" {
				tfdiags.AssertNoErrors(t, diags)
				return
			}
			if len(diags) != 1 {
				t.Fatalf("expected one diagnostic, got %s", diags.ErrWithWarnings())
			}
			desc := diags[0].Description()
			if diags[0].Severity() != tfdiags.Error || desc.Summary != `Invalid relationship "r"` || desc.Detail != test.detail {
				t.Errorf("wrong diagnostic\ngot:  %s: %s\nwant: Invalid relationship \"r\": %s", desc.Summary, desc.Detail, test.detail)
			}
			if subject := diags[0].Source().Subject; subject == nil || subject.Filename != "relationships.tfpolicy.hcl" {
				t.Errorf("wrong subject %v", subject)
			}
			// The plan fails before the walk.
			if provider.PlanResourceChangeCalled || len(client.EvaluateRequests) > 0 {
				t.Error("expected the plan to fail before the walk")
			}
		})
	}

	t.Run("apply", func(t *testing.T) {
		provider := relationshipsTestProvider()
		ctx := relContext(t, provider)
		plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{Mode: plans.NormalMode})
		tfdiags.AssertNoErrors(t, diags)

		client := relClient(t, policy.Relationships{Definitions: []*proto.RelationshipDefinition{tests["write-only"].def}})
		_, diags = ctx.Apply(plan, mod, &ApplyOpts{PolicyClient: client, ProviderLocks: locks})
		if !diags.HasErrors() || diags.Err().Error() != `Invalid relationship "r": `+tests["write-only"].detail {
			t.Fatalf("expected the invalid relationship to fail the apply, got %v", diags.ErrWithWarnings())
		}
		if provider.ApplyResourceChangeCalled || len(client.EvaluateRequests) > 0 {
			t.Error("expected the apply to fail before the walk")
		}
	})

	t.Run("version unknown", func(t *testing.T) {
		client := relClient(t, policy.Relationships{Definitions: []*proto.RelationshipDefinition{tests["no such type"].def}})
		_, diags := relContext(t, relationshipsTestProvider()).Plan(mod, states.NewState(), &PlanOpts{
			Mode:         plans.NormalMode,
			PolicyClient: client,
		})
		if want := "registry.terraform.io/hashicorp/test (version unknown) has no resource type test_nope."; !diags.HasErrors() || diags[0].Description().Detail != want {
			t.Fatalf("expected the error %q, got %v", want, diags.ErrWithWarnings())
		}
	})
}

func TestContext2Plan_PolicyRelationships_related(t *testing.T) {
	tests := map[string]struct {
		files map[string]string
		state func(*states.SyncState)
		opts  *PlanOpts
		rels  policy.Relationships
		// planFn replaces the provider's planning when set.
		planFn func(providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse
		// wantErrors is set if the plan fails.
		wantErrors bool
		want       map[string][]string
	}{
		// A plan with every candidate decided has every member.
		"known values": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "v" {
					net_id = "a-id"
				}
				resource "test_vm" "none" {
					net_id = "c-id"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
				relNetState(s, "test_net.b", `{"id":"b-id","name":"b"}`)
			},
			rels: relVMNet(),
			want: map[string][]string{
				"test_net.a NO_OP":    {"vm_net COMPLETE", "  test_vm.v CREATE attrs"},
				"test_net.b NO_OP":    {"vm_net COMPLETE"},
				"test_vm.v CREATE":    {"vm_net COMPLETE", "  test_net.a NO_OP attrs prior"},
				"test_vm.none CREATE": {"vm_net COMPLETE"},
			},
		},
		// A candidate is undecided if no key of some alternative certainly
		// fails. A key with an empty side certainly fails.
		"undecided": {
			files: map[string]string{"main.tf": `
				resource "test_net" "new" {
					name = "new"
				}
				resource "test_vm" "v" {
					net_id = "a-id"
				}
				resource "test_vm" "none" {
					name = "none"
				}
			`},
			rels: policy.Relationships{
				Definitions: relVMNet().Definitions,
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_vm.v CREATE":    {"vm_net PLAN"},
				"test_vm.none CREATE": {"vm_net COMPLETE"},
			},
		},
		// Unknown values relate nothing, but a reference relates the
		// instance to the value it refers to. Unknown values that don't
		// share an origin are undecided.
		"origins": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				locals {
					net_id = test_net.a.id
				}
				resource "test_vm" "v" {
					net_id = local.net_id
				}
				resource "test_vm" "list" {
					net_ids = [test_net.b.id]
				}
			`},
			rels: policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_id=id", "net_ids=id")},
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net")), relUsage("test_net", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_net.a CREATE":   {"vm_net PLAN", "  test_vm.v CREATE attrs"},
				"test_net.b CREATE":   {"vm_net PLAN", "  test_vm.list CREATE attrs"},
				"test_vm.v CREATE":    {"vm_net PLAN", "  test_net.a CREATE attrs"},
				"test_vm.list CREATE": {"vm_net PLAN", "  test_net.b CREATE attrs"},
			},
		},
		// Every key of an alternative must hold.
		"alternatives": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "both" {
					net_id = "a-id"
					zone   = "a"
				}
				resource "test_vm" "one" {
					net_id = "a-id"
					zone   = "b"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
			},
			rels: policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_id=id&zone=name")},
				Usages:      []*proto.RelationshipUsage{relUsage("test_net", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_net.a NO_OP": {"vm_net COMPLETE", "  test_vm.both CREATE attrs"},
			},
		},
		// Instances are related only with the same provider configuration:
		// the same address, or known and equal values. An unknown provider
		// configuration leaves a candidate undecided.
		"provider rule": {
			files: map[string]string{"main.tf": `
				provider "test" {
					region = "a"
				}
				provider "test" {
					alias  = "same"
					region = "a"
				}
				provider "test" {
					alias  = "other"
					region = "b"
				}
				provider "test" {
					alias  = "unknown"
					region = test_net.new.id
				}
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "new" {
					name = "new"
				}
				resource "test_vm" "default" {
					net_id = "a-id"
				}
				resource "test_vm" "same" {
					provider = test.same
					net_id   = "a-id"
				}
				resource "test_vm" "other" {
					provider = test.other
					net_id   = "a-id"
				}
				resource "test_vm" "unknown" {
					provider = test.unknown
					net_id   = "a-id"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
			},
			rels: policy.Relationships{
				Definitions: relVMNet().Definitions,
				Usages:      []*proto.RelationshipUsage{relUsage("test_net", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_net.a NO_OP":    {"vm_net PLAN", "  test_vm.default CREATE attrs", "  test_vm.same CREATE attrs"},
				"test_net.new CREATE": {"vm_net PLAN"},
			},
		},
		// A member gets the members of the children of its hop, walked from
		// it, so the subject is a member of its member's hop.
		"nested hops": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
				}
				resource "test_vm" "w" {
					net_id = test_net.a.id
				}
			`},
			rels: policy.Relationships{
				Definitions: relVMNet().Definitions,
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net", relHop("vm_net")))},
			},
			want: map[string][]string{
				"test_vm.v CREATE": {
					"vm_net COMPLETE",
					"  test_net.a CREATE attrs",
					"    vm_net COMPLETE",
					"      test_vm.v CREATE attrs",
					"      test_vm.w CREATE attrs",
				},
				"test_vm.w CREATE": {
					"vm_net COMPLETE",
					"  test_net.a CREATE attrs",
					"    vm_net COMPLETE",
					"      test_vm.v CREATE attrs",
					"      test_vm.w CREATE attrs",
				},
			},
		},
		"same type": {
			files: map[string]string{"main.tf": `
				resource "test_vm" "a" {
					name = "a"
					zone = "z"
				}
				resource "test_vm" "b" {
					name = "b"
					zone = "z"
				}
			`},
			rels: policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{relDefinition("zone", "test_vm", "test_vm", "zone=zone")},
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("zone"))},
			},
			want: map[string][]string{
				"test_vm.a CREATE": {"zone COMPLETE", "  test_vm.b CREATE attrs"},
				"test_vm.b CREATE": {"zone COMPLETE", "  test_vm.a CREATE attrs"},
			},
		},
		// Deletes, the old objects of replaces and deposed objects are
		// members with their prior values. The replacement of test_net.r is
		// undecided.
		"deletes, replaces and deposed objects": {
			files: map[string]string{"main.tf": `
				resource "test_net" "r" {
					name = "r"
				}
				resource "test_net" "u" {
					name = "new"
				}
				resource "test_vm" "v" {
					net_ids = ["r-id", "gone-id", "deposed-id", "u-id"]
				}
				removed {
					from = test_net.f
					lifecycle {
						destroy = false
					}
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.r", `{"id":"r-id","name":"r"}`)
				relNetState(s, "test_net.u", `{"id":"u-id","name":"old"}`)
				relNetDeposedState(s, "test_net.u", "00000001", `{"id":"deposed-id","name":"older"}`)
				relNetState(s, "test_net.gone", `{"id":"gone-id","name":"gone"}`)
				// Forgotten instances are never members.
				relNetState(s, "test_net.f", `{"id":"r-id","name":"f"}`)
			},
			opts: &PlanOpts{
				Mode:         plans.NormalMode,
				ForceReplace: []addrs.AbsResourceInstance{mustResourceInstanceAddr("test_net.r")},
			},
			rels: policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_ids=id")},
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_vm.v CREATE": {
					"vm_net PLAN",
					"  test_net.gone DELETE prior",
					"  test_net.r old DELETE prior",
					"  test_net.u UPDATE attrs prior",
					"  test_net.u deposed 00000001 DELETE prior",
				},
			},
		},
		// Instances outside the targets are in the state with no change.
		"targeted": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "new"
				}
				resource "test_vm" "v" {
					net_id = "a-id"
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.a", `{"id":"a-id","name":"a"}`)
			},
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_vm.v")},
			},
			rels: relVMNet(),
			want: map[string][]string{
				"test_vm.v CREATE": {"vm_net COMPLETE", "  test_net.a NO_OP attrs"},
			},
		},
		"deferred": {
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "west"
					region = "west"
				}
				resource "test_net" "deferred" {
					name  = "deferred"
					defer = true
				}
				resource "test_net" "partial" {
					for_each = toset([test_vm.a.id])
					name     = each.key
				}
				resource "test_net" "west" {
					provider = test.west
					name     = "west"
					defer    = true
				}
				resource "test_vm" "a" {
					name = "a"
				}
			`},
			opts: &PlanOpts{Mode: plans.NormalMode, DeferralAllowed: true},
			rels: relVMNet(),
			// The deferred test_net.west has a provider configuration that is
			// known to differ.
			want: map[string][]string{
				"test_vm.a CREATE": {"vm_net DEFERRED test_net.deferred test_net.partial[*]"},
			},
		},
		"planning failed": {
			files: map[string]string{"main.tf": `
				resource "test_net" "bad" {
					name = "bad"
				}
				resource "test_vm" "v" {
					name = "v"
				}
			`},
			planFn: func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
				if name := req.Config.GetAttr("name"); name.IsKnown() && name.AsString() == "bad" {
					var resp providers.PlanResourceChangeResponse
					resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("planning failed"))
					return resp
				}
				return testDiffFn(req)
			},
			wantErrors: true,
			rels:       relVMNet(),
			want: map[string][]string{
				"test_vm.v CREATE": {"vm_net ERROR"},
			},
		},
		"state object with another schema version": {
			files: map[string]string{"main.tf": `
				resource "test_net" "old" {
					name = "old"
				}
				resource "test_vm" "v" {
					name = "v"
				}
			`},
			state: func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(mustResourceInstanceAddr("test_net.old"), &states.ResourceInstanceObjectSrc{
					AttrsJSON:     []byte(`{"id":"old-id","name":"old"}`),
					SchemaVersion: 1,
					Status:        states.ObjectReady,
				}, mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`))
			},
			opts: &PlanOpts{
				Mode:    plans.NormalMode,
				Targets: []addrs.Targetable{mustResourceInstanceAddr("test_vm.v")},
			},
			rels: policy.Relationships{
				Definitions: relVMNet().Definitions,
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_vm.v CREATE": {"vm_net ERROR"},
			},
		},
		"no usage": {
			files: map[string]string{"main.tf": `
				resource "test_vm" "v" {
					name = "v"
				}
			`},
			rels: policy.Relationships{Definitions: relVMNet().Definitions},
			want: map[string][]string{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, test.files)
			opts := test.opts
			if opts == nil {
				opts = &PlanOpts{Mode: plans.NormalMode}
			}
			client := relClient(t, test.rels)
			opts.PolicyClient = client
			provider := relationshipsTestProvider()
			if test.planFn != nil {
				provider.PlanResourceChangeFn = test.planFn
			}
			var state *states.State
			if test.state != nil {
				state = states.BuildState(test.state)
			}
			_, diags := relContext(t, provider).Plan(mod, relState(state), opts)
			if test.wantErrors != diags.HasErrors() {
				t.Fatalf("wrong errors: %v", diags.ErrWithWarnings())
			}
			if diff := cmp.Diff(test.want, relTrees(t, client)); diff != "" {
				t.Errorf("wrong related members (-want +got):\n%s", diff)
			}
		})
	}
}

// The values of members are those of the candidates table: the planned value
// and the prior value of a change, and the state value of an instance in
// the state, with sensitive values redacted.
func TestContext2Plan_PolicyRelationships_memberValues(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_net" "u" {
			name   = "new"
			secret = "s"
		}
		resource "test_net" "t" {
			name = "t"
		}
		resource "test_vm" "v" {
			net_ids = ["u-id", "t-id"]
		}
	`})
	state := states.BuildState(func(s *states.SyncState) {
		relNetState(s, "test_net.u", `{"id":"u-id","name":"old","secret":"s"}`)
		relNetState(s, "test_net.t", `{"id":"t-id","name":"old-t"}`)
	})
	client := relClient(t, policy.Relationships{
		Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_ids=id")},
		Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net"))},
	})
	relPlan(t, mod, state, &PlanOpts{
		Mode:         plans.NormalMode,
		PolicyClient: client,
		Targets:      []addrs.Targetable{mustResourceInstanceAddr("test_vm.v"), mustResourceInstanceAddr("test_net.u")},
	}, relationshipsTestProvider())

	members := relMembers(t, client)["test_vm.v CREATE"]
	u, tm := members["test_net.u"], members["test_net.t"]
	if u == nil || tm == nil {
		t.Fatalf("missing members: %v", members)
	}
	got := []string{
		relAttr(t, u.Attrs, "name"), relAttr(t, u.PriorAttrs, "name"),
		relAttr(t, tm.Attrs, "name"),
	}
	if diff := cmp.Diff([]string{"new", "old", "old-t"}, got); diff != "" {
		t.Errorf("wrong values (-want +got):\n%s", diff)
	}
	for _, attrs := range []*proto.ResourceAttributes{u.Attrs, u.PriorAttrs} {
		var redacted []string
		for _, path := range attrs.GetRedactedPaths() {
			redacted = append(redacted, path.Steps[0].GetAttributeName())
		}
		sort.Strings(redacted)
		if diff := cmp.Diff([]string{"secret"}, redacted); diff != "" {
			t.Errorf("wrong redacted paths (-want +got):\n%s", diff)
		}
	}
	if u.Type != "test_net" || u.ProviderType != "test" || u.ProviderSource != relTestSource || u.ModulePath != "" {
		t.Errorf("wrong metadata: %v", u)
	}
}

func TestContext2Plan_PolicyRelationships_query(t *testing.T) {
	mod := testModuleInline(t, map[string]string{
		"main.tf": `
			terraform {
				required_providers {
					test = {
						source = "hashicorp/test"
						version = "1.0.0"
					}
				}
			}
		`,
		"main.tfquery.hcl": `
			list "test_resource" "test1" {
				provider = test
				include_resource = true

				config {
					filter = {
						attr = "foo"
					}
				}
			}
		`,
	}, configs.MatchQueryFiles())

	provider := testProvider("test")
	provider.GetProviderSchemaResponse = getListProviderSchemaResp()
	provider.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
		return providers.ListResourceResponse{Result: cty.ObjectVal(map[string]cty.Value{
			"data": cty.TupleVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
				"identity":     cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("i-1")}),
				"display_name": cty.StringVal("Instance 1"),
				"state":        cty.ObjectVal(map[string]cty.Value{"instance_type": cty.StringVal("ami-1")}),
			})}),
			"config": request.Config.GetAttr("config"),
		})}
	}

	client := relClient(t, policy.Relationships{
		Definitions: []*proto.RelationshipDefinition{relDefinition("self", "test_resource", "test_resource", "id=id")},
		Usages:      []*proto.RelationshipUsage{relUsage("test_resource", relHop("self"))},
	})
	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(provider),
		},
	})
	_, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{
		Mode:         plans.NormalMode,
		SetVariables: testInputValuesUnset(mod.Module.Variables),
		Query:        true,
		PolicyClient: client,
	})
	tfdiags.AssertNoDiagnostics(t, diags)

	if len(client.EvaluateRequests) == 0 {
		t.Fatal("expected the query results to be evaluated")
	}
	for _, req := range client.EvaluateRequests {
		if req.Related != nil {
			t.Errorf("expected no related members for %s, got %v", req.Meta.Address, req.Related)
		}
	}
}

func TestContext2Apply_PolicyRelationships_related(t *testing.T) {
	tests := map[string]struct {
		files map[string]string
		state func(*states.SyncState)
		opts  *PlanOpts
		rels  policy.Relationships
		// applyFn wraps the provider's apply when set.
		applyFn func(apply func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse) func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse
		// wantErrors is set if the apply fails.
		wantErrors bool
		want       map[string][]string
	}{
		// At apply, the final values relate instances.
		"applied values": {
			files: map[string]string{"main.tf": `
				resource "test_net" "a" {
					name = "a"
				}
				resource "test_net" "b" {
					name = "b"
				}
				resource "test_vm" "v" {
					net_id = test_net.a.id
				}
				resource "test_vm" "w" {
					net_ids = ["b-id"]
				}
			`},
			rels: policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_id=id", "net_ids=id")},
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net")), relUsage("test_net", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_net.a CREATE": {"vm_net COMPLETE", "  test_vm.v CREATE attrs"},
				"test_net.b CREATE": {"vm_net COMPLETE", "  test_vm.w CREATE attrs"},
				"test_vm.v CREATE":  {"vm_net COMPLETE", "  test_net.a CREATE attrs"},
				"test_vm.w CREATE":  {"vm_net COMPLETE", "  test_net.b CREATE attrs"},
			},
		},
		"deletes, replaces, deposed objects and the state": {
			files: map[string]string{"main.tf": `
				resource "test_net" "r" {
					name = "r"
				}
				resource "test_net" "u" {
					name = "new"
				}
				resource "test_net" "outside" {
					name = "outside"
				}
				resource "test_vm" "v" {
					net_ids = ["r-id", "gone-id", "deposed-id", "u-id", "outside-id"]
				}
			`},
			state: func(s *states.SyncState) {
				relNetState(s, "test_net.r", `{"id":"r-id","name":"r"}`)
				relNetState(s, "test_net.u", `{"id":"u-id","name":"old"}`)
				relNetDeposedState(s, "test_net.u", "00000001", `{"id":"deposed-id","name":"older"}`)
				relNetState(s, "test_net.gone", `{"id":"gone-id","name":"gone"}`)
				relNetState(s, "test_net.outside", `{"id":"outside-id","name":"outside"}`)
			},
			opts: &PlanOpts{
				Mode:         plans.NormalMode,
				ForceReplace: []addrs.AbsResourceInstance{mustResourceInstanceAddr("test_net.r")},
				Targets: []addrs.Targetable{
					mustResourceInstanceAddr("test_vm.v"),
					mustResourceInstanceAddr("test_net.r"),
					mustResourceInstanceAddr("test_net.u"),
					mustResourceInstanceAddr("test_net.gone"),
				},
			},
			rels: policy.Relationships{
				Definitions: []*proto.RelationshipDefinition{relDefinition("vm_net", "test_vm", "test_net", "net_ids=id")},
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net"))},
			},
			// The new object of test_net.r has the id r-id after apply.
			want: map[string][]string{
				"test_vm.v CREATE": {
					"vm_net COMPLETE",
					"  test_net.gone DELETE prior",
					"  test_net.outside NO_OP attrs",
					"  test_net.r CREATE attrs",
					"  test_net.r old DELETE prior",
					"  test_net.u UPDATE attrs prior",
					"  test_net.u deposed 00000001 DELETE prior",
				},
			},
		},
		"apply failed": {
			files: map[string]string{"main.tf": `
				resource "test_net" "bad" {
					name = "bad"
				}
				resource "test_vm" "v" {
					name = "v"
				}
			`},
			applyFn: func(apply func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse) func(providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
				return func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
					if name := req.PlannedState.GetAttr("name"); !name.IsNull() && name.AsString() == "bad" {
						var resp providers.ApplyResourceChangeResponse
						resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("apply failed"))
						return resp
					}
					return apply(req)
				}
			},
			wantErrors: true,
			rels: policy.Relationships{
				Definitions: relVMNet().Definitions,
				Usages:      []*proto.RelationshipUsage{relUsage("test_vm", relHop("vm_net"))},
			},
			want: map[string][]string{
				"test_vm.v CREATE": {"vm_net ERROR"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mod := testModuleInline(t, test.files)
			opts := test.opts
			if opts == nil {
				opts = &PlanOpts{Mode: plans.NormalMode}
			}
			provider := relationshipsTestProvider()
			if test.applyFn != nil {
				provider.ApplyResourceChangeFn = test.applyFn(provider.ApplyResourceChangeFn)
			}
			var state *states.State
			if test.state != nil {
				state = states.BuildState(test.state)
			}
			ctx := relContext(t, provider)
			plan, diags := ctx.Plan(mod, relState(state), opts)
			tfdiags.AssertNoErrors(t, diags)

			client := relClient(t, test.rels)
			_, diags = ctx.Apply(plan, mod, &ApplyOpts{PolicyClient: client})
			if test.wantErrors != diags.HasErrors() {
				t.Fatalf("wrong errors: %v", diags.ErrWithWarnings())
			}
			if diff := cmp.Diff(test.want, relTrees(t, client)); diff != "" {
				t.Errorf("wrong related members (-want +got):\n%s", diff)
			}
		})
	}
}

// At apply, a member has its final state value.
func TestContext2Apply_PolicyRelationships_memberValues(t *testing.T) {
	mod := testModuleInline(t, map[string]string{"main.tf": `
		resource "test_net" "a" {
			name = "a"
		}
		resource "test_vm" "v" {
			net_id = test_net.a.id
		}
	`})
	provider := relationshipsTestProvider()
	ctx := relContext(t, provider)
	plan, diags := ctx.Plan(mod, states.NewState(), &PlanOpts{Mode: plans.NormalMode})
	tfdiags.AssertNoErrors(t, diags)

	client := relClient(t, relVMNet())
	_, diags = ctx.Apply(plan, mod, &ApplyOpts{PolicyClient: client})
	tfdiags.AssertNoErrors(t, diags)

	m := relMembers(t, client)["test_vm.v CREATE"]["test_net.a"]
	if m == nil {
		t.Fatal("expected test_net.a to be related to test_vm.v")
	}
	if got := relAttr(t, m.Attrs, "id"); got != "a-id" {
		t.Errorf("wrong id %q, want a-id", got)
	}
	if m.PriorAttrs != nil {
		t.Error("expected no prior attrs for a create")
	}
}
