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
	protobuf "google.golang.org/protobuf/proto"
)

const relTestSource = "registry.terraform.io/hashicorp/test"

// relDefinition returns a definition of the test provider. Each match is a
// list of keys joined by "&", each key a key path and a value path joined by
// "=", like "net_id=id&zone=name".
func relDefinition(name, keyType, valueType string, matches ...string) *proto.Relationship {
	def := &proto.Relationship{
		ProviderSource: relTestSource,
		Name:           name,
		KeyType:        keyType,
		ValueType:      valueType,
		DeclRange: &proto.Range{
			Filename: "relationships.tfpolicy.hcl",
			Start:    &proto.Position{Line: 1, Column: 1},
			End:      &proto.Position{Line: 1, Column: 20, Byte: 19},
		},
	}
	for _, match := range matches {
		var keys []*proto.Relationship_Key
		for _, key := range strings.Split(match, "&") {
			kp, vp, _ := strings.Cut(key, "=")
			keys = append(keys, &proto.Relationship_Key{KeyPath: relAttrPath(kp), ValuePath: relAttrPath(vp)})
		}
		def.Matches = append(def.Matches, &proto.Relationship_Match{Keys: keys})
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

// relUsage returns the tree of a subject type of the test provider, whose
// nodes relRelationships inlines.
func relUsage(typ string, hops ...*proto.Relationship) *proto.RelationshipTree {
	return &proto.RelationshipTree{ProviderSource: relTestSource, ResourceType: typ, Relationships: hops}
}

// relHop returns a node of a tree that follows the named definition.
func relHop(relationship string, hops ...*proto.Relationship) *proto.Relationship {
	return &proto.Relationship{Name: relationship, Relationships: hops}
}

// relRelationships returns the trees of the given usages, with the named
// definitions inlined in their nodes, as Setup returns them.
func relRelationships(defs []*proto.Relationship, usages ...*proto.RelationshipTree) policy.Relationships {
	byName := make(map[string]*proto.Relationship)
	for _, def := range defs {
		byName[def.Name] = def
	}
	var inline func([]*proto.Relationship) []*proto.Relationship
	inline = func(hops []*proto.Relationship) []*proto.Relationship {
		var ret []*proto.Relationship
		for _, hop := range hops {
			node := protobuf.Clone(byName[hop.Name]).(*proto.Relationship)
			node.Relationships = inline(hop.Relationships)
			ret = append(ret, node)
		}
		return ret
	}
	var rels policy.Relationships
	for _, usage := range usages {
		rels.Trees = append(rels.Trees, &proto.RelationshipTree{
			ProviderSource: usage.ProviderSource,
			ResourceType:   usage.ResourceType,
			Relationships:  inline(usage.Relationships),
		})
	}
	return rels
}

func relVMNetDefs() []*proto.Relationship {
	return []*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_id=id")}
}

// relVMNet relates a test_vm to the test_net whose id is its net_id, and both
// types use it.
func relVMNet() policy.Relationships {
	return relRelationships(relVMNetDefs(), relUsage("test_vm", relHop("vm_net")), relUsage("test_net", relHop("vm_net")))
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

// relSubject returns the key of an evaluated subject: its address,
// " deposed KEY" for a deposed object, " old" for the old object of a
// replace, and its operation.
func relSubject(meta *proto.PolicyEvaluateResourceRequest_ResourceMetadata) string {
	key := meta.Address
	if meta.DeposedKey != "" {
		key += " deposed " + meta.DeposedKey
	}
	if meta.OldObject {
		key += " old"
	}
	return key + " " + meta.Operation.String()
}

// relResolve returns the related resource with the given identifier, as the
// plugin fetches it, and fails the test if there is none.
func relResolve(t *testing.T, client *policy.MockClient, id string) *proto.GetResourcesByIdentifierResponse_Resource {
	t.Helper()
	resp := client.GetResourcesByIdentifier(t.Context(), []string{id})
	if len(resp.Diagnostics) > 0 || len(resp.Resources) != 1 {
		t.Fatalf("can't resolve %q: %v", id, resp.Diagnostics)
	}
	return resp.Resources[0]
}

// relTrees returns the relations of every evaluated subject that has any,
// rendered by relTree, by subject key (relSubject).
func relTrees(t *testing.T, client *policy.MockClient) map[string][]string {
	t.Helper()
	ret := make(map[string][]string)
	for _, req := range client.EvaluateRequests {
		if req.Relationships == nil {
			continue
		}
		key := relSubject(req.Meta)
		if _, ok := ret[key]; ok {
			t.Fatalf("subject %s evaluated twice", key)
		}
		ret[key] = relTree(t, client, req.Relationships)
	}
	return ret
}

// relTree renders relations, one line per relation and related resource,
// nested relations indented: "RELATIONSHIP COMPLETENESS [DEFERRED...]" for a
// relation, and "ADDRESS [deposed KEY] [old] OPERATION [attrs] [prior]" for
// a resource, with attrs and prior when it has attrs and prior attrs.
// Resources are sorted by their lines.
func relTree(t *testing.T, client *policy.MockClient, relations []*proto.Relation) []string {
	t.Helper()
	var walk func([]*proto.Relation, string) []string
	walk = func(relations []*proto.Relation, indent string) []string {
		var lines []string
		for _, rel := range relations {
			line := []string{rel.Relationship, strings.TrimSuffix(rel.Completeness.String(), "_COMPLETENESS")}
			lines = append(lines, indent+strings.Join(append(line, rel.Deferred...), " "))
			var blocks [][]string
			for id, related := range rel.Resources {
				r := relResolve(t, client, id)
				line := []string{r.Metadata.Address}
				if r.Metadata.DeposedKey != "" {
					line = append(line, "deposed", r.Metadata.DeposedKey)
				}
				if r.Metadata.OldObject {
					line = append(line, "old")
				}
				if related.Operation != r.Metadata.Operation {
					t.Errorf("%s has operation %s in the relation, but %s in its metadata", r.Metadata.Address, related.Operation, r.Metadata.Operation)
				}
				line = append(line, related.Operation.String())
				if r.Attrs != nil {
					line = append(line, "attrs")
				}
				if r.PriorAttrs != nil {
					line = append(line, "prior")
				}
				block := []string{indent + "  " + strings.Join(line, " ")}
				blocks = append(blocks, append(block, walk(related.Relationships, indent+"    ")...))
			}
			sort.Slice(blocks, func(i, j int) bool { return blocks[i][0] < blocks[j][0] })
			for _, block := range blocks {
				lines = append(lines, block...)
			}
		}
		return lines
	}
	return walk(relations, "")
}

// relMembers returns the related resources of every relation of every
// subject, by subject key like relTrees, then by address.
func relMembers(t *testing.T, client *policy.MockClient) map[string]map[string]*proto.GetResourcesByIdentifierResponse_Resource {
	t.Helper()
	ret := make(map[string]map[string]*proto.GetResourcesByIdentifierResponse_Resource)
	for _, req := range client.EvaluateRequests {
		key := relSubject(req.Meta)
		for _, rel := range req.Relationships {
			for id := range rel.Resources {
				if ret[key] == nil {
					ret[key] = make(map[string]*proto.GetResourcesByIdentifierResponse_Resource)
				}
				r := relResolve(t, client, id)
				ret[key][r.Metadata.Address] = r
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
		def *proto.Relationship
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
			def: &proto.Relationship{
				ProviderSource: "registry.terraform.io/hashicorp/other",
				Name:           "r",
				KeyType:        "other_nope",
				ValueType:      "other_nope",
				Matches:        []*proto.Relationship_Match{{Keys: []*proto.Relationship_Key{{KeyPath: relAttrPath("nope"), ValuePath: relAttrPath("nope")}}}},
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
			rels := relRelationships([]*proto.Relationship{test.def}, relUsage("test_net", relHop("r")))
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

		client := relClient(t, relRelationships([]*proto.Relationship{tests["write-only"].def}, relUsage("test_net", relHop("r"))))
		_, diags = ctx.Apply(plan, mod, &ApplyOpts{PolicyClient: client, ProviderLocks: locks})
		if !diags.HasErrors() || diags.Err().Error() != `Invalid relationship "r": `+tests["write-only"].detail {
			t.Fatalf("expected the invalid relationship to fail the apply, got %v", diags.ErrWithWarnings())
		}
		if provider.ApplyResourceChangeCalled || len(client.EvaluateRequests) > 0 {
			t.Error("expected the apply to fail before the walk")
		}
	})

	t.Run("version unknown", func(t *testing.T) {
		client := relClient(t, relRelationships([]*proto.Relationship{tests["no such type"].def}, relUsage("test_net", relHop("r"))))
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
			rels: relRelationships(relVMNetDefs(), relUsage("test_vm", relHop("vm_net"))),
			want: map[string][]string{
				"test_vm.v CREATE":    {"vm_net PLAN"},
				"test_vm.none CREATE": {"vm_net COMPLETE"},
			},
		},
		// Unknown values relate nothing, but a plain reference relates the
		// instance to the value it refers to. Unknown values that don't
		// share an origin are undecided: a collection constructor like
		// [test_net.b.id] gives no origin.
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
			rels: relRelationships([]*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_id=id", "net_ids=id")}, relUsage("test_vm", relHop("vm_net")), relUsage("test_net", relHop("vm_net"))),
			want: map[string][]string{
				"test_net.a CREATE":   {"vm_net PLAN", "  test_vm.v CREATE attrs"},
				"test_net.b CREATE":   {"vm_net PLAN"},
				"test_vm.v CREATE":    {"vm_net PLAN", "  test_net.a CREATE attrs"},
				"test_vm.list CREATE": {"vm_net PLAN"},
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
			rels: relRelationships([]*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_id=id&zone=name")}, relUsage("test_net", relHop("vm_net"))),
			want: map[string][]string{
				"test_net.a NO_OP": {"vm_net COMPLETE", "  test_vm.both CREATE attrs"},
			},
		},
		// Instances are related only with the same provider configuration
		// address. Configurations aren't compared by value, so equal and
		// unknown values relate nothing.
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
			rels: relRelationships(relVMNetDefs(), relUsage("test_net", relHop("vm_net"))),
			want: map[string][]string{
				"test_net.a NO_OP":    {"vm_net COMPLETE", "  test_vm.default CREATE attrs"},
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
			rels: relRelationships(relVMNetDefs(), relUsage("test_vm", relHop("vm_net", relHop("vm_net")))),
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
			rels: relRelationships([]*proto.Relationship{relDefinition("zone", "test_vm", "test_vm", "zone=zone")}, relUsage("test_vm", relHop("zone"))),
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
			rels: relRelationships([]*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_ids=id")}, relUsage("test_vm", relHop("vm_net"))),
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
			rels: relRelationships(relVMNetDefs(), relUsage("test_vm", relHop("vm_net"))),
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
			rels: relRelationships(relVMNetDefs()),
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
	client := relClient(t, relRelationships([]*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_ids=id")}, relUsage("test_vm", relHop("vm_net"))))
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
	if u.Type != "test_net" || u.Metadata.ProviderType != "test" || u.Metadata.ProviderSource != relTestSource || u.Metadata.ModulePath != "" || u.Metadata.Operation != proto.Operation_UPDATE {
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

	client := relClient(t, relRelationships([]*proto.Relationship{relDefinition("self", "test_resource", "test_resource", "id=id")}, relUsage("test_resource", relHop("self"))))
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
		if req.Relationships != nil {
			t.Errorf("expected no related members for %s, got %v", req.Meta.Address, req.Relationships)
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
			rels: relRelationships([]*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_id=id", "net_ids=id")}, relUsage("test_vm", relHop("vm_net")), relUsage("test_net", relHop("vm_net"))),
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
			rels: relRelationships([]*proto.Relationship{relDefinition("vm_net", "test_vm", "test_net", "net_ids=id")}, relUsage("test_vm", relHop("vm_net"))),
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
			rels:       relRelationships(relVMNetDefs(), relUsage("test_vm", relHop("vm_net"))),
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
