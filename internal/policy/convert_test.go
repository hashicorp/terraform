// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	protobuf "google.golang.org/protobuf/proto"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

func setupDefinition(name, keyType, valueType string) *proto.Relationship {
	path := func(name string) *proto.AttributePath {
		return &proto.AttributePath{Steps: []*proto.AttributePath_Step{{
			Selector: &proto.AttributePath_Step_AttributeName{AttributeName: name},
		}}}
	}
	return &proto.Relationship{
		Name: name, KeyType: keyType, ValueType: valueType,
		Matches: []*proto.Relationship_Match{{Keys: []*proto.Relationship_Key{{KeyPath: path("id"), ValuePath: path("parent_id")}}}},
	}
}

func TestConfigFromProto(t *testing.T) {
	rel := func(name, keyType, valueType string, nested ...Relationship) Relationship {
		if nested == nil {
			nested = []Relationship{}
		}
		return Relationship{Name: name, KeyType: keyType, ValueType: valueType,
			Matches: []Match{{{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("parent_id")}}}, Relationships: nested}
	}
	for name, tc := range map[string]struct {
		capabilities *proto.PolicySetupResponse_ServerCapabilities
		definitions  int
		want         map[string][]Relationship
	}{
		"nil capabilities": {},
		"unused definition": {capabilities: &proto.PolicySetupResponse_ServerCapabilities{
			Relationships: []*proto.Relationship{setupDefinition("unused", "a", "b")},
		}, definitions: 1, want: map[string][]Relationship{}},
		"finite shared trees": {capabilities: &proto.PolicySetupResponse_ServerCapabilities{
			Relationships: []*proto.Relationship{setupDefinition("a_b", "a", "b"), setupDefinition("b_c", "b", "c"), setupDefinition("b_b", "b", "b")},
			ResourceRelationships: []*proto.ResourceRelationships{
				{ResourceType: "a", References: []*proto.ResourceRelationships_Reference{{Relationship: "a_b", References: []*proto.ResourceRelationships_Reference{
					{Relationship: "b_c"},
					{Relationship: "b_b", References: []*proto.ResourceRelationships_Reference{{Relationship: "b_b", References: []*proto.ResourceRelationships_Reference{{Relationship: "b_c"}}}}},
				}}}},
				{ResourceType: "b", References: []*proto.ResourceRelationships_Reference{{Relationship: "a_b", References: []*proto.ResourceRelationships_Reference{
					{Relationship: "a_b", References: []*proto.ResourceRelationships_Reference{{Relationship: "b_c"}}},
				}}}},
			},
		}, definitions: 3, want: map[string][]Relationship{
			"a": {rel("a_b", "a", "b", rel("b_c", "b", "c"), rel("b_b", "b", "b", rel("b_b", "b", "b", rel("b_c", "b", "c"))))},
			"b": {rel("a_b", "a", "b", rel("a_b", "a", "b", rel("b_c", "b", "c")))},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			if name == "finite shared trees" {
				wire, err := protobuf.Marshal(tc.capabilities)
				if err != nil {
					t.Fatal(err)
				}
				tc.capabilities = new(proto.PolicySetupResponse_ServerCapabilities)
				if err := protobuf.Unmarshal(wire, tc.capabilities); err != nil {
					t.Fatal(err)
				}
			}
			config, err := configFromProto(tc.capabilities)
			if err != nil || config == nil || len(config.Definitions) != tc.definitions {
				t.Fatalf("config=%#v err=%v", config, err)
			}
			if diff := cmp.Diff(tc.want, config.Relationships, ctydebug.CmpOptions); diff != "" {
				t.Fatalf("wrong resolved trees (-want +got):\n%s", diff)
			}
			for _, definition := range config.Definitions {
				if len(definition.Relationships) != 0 {
					t.Fatal("shared definitions acquired usage subtrees")
				}
			}
			if name == "finite shared trees" {
				config.Relationships["a"][0].Relationships[0].Name = "changed"
				if config.Relationships["b"][0].Relationships[0].Relationships[0].Name != "b_c" {
					t.Fatal("usage subtrees share mutable storage")
				}
			}
		})
	}
}

func TestConfigFromProtoInvalidRelationships(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(*proto.PolicySetupResponse_ServerCapabilities)
		want   string
	}{
		"empty name": {func(c *proto.PolicySetupResponse_ServerCapabilities) { c.Relationships[0].Name = "" }, "definition has no name"},
		"duplicate definition": {func(c *proto.PolicySetupResponse_ServerCapabilities) {
			c.Relationships = append(c.Relationships, c.Relationships[0])
		}, "duplicate relationship definition"},
		"empty endpoint": {func(c *proto.PolicySetupResponse_ServerCapabilities) { c.Relationships[0].KeyType = "" }, "endpoint types must not be empty"},
		"unused malformed definition": {func(c *proto.PolicySetupResponse_ServerCapabilities) {
			c.Relationships = append(c.Relationships, setupDefinition("unused", "x", "y"))
			c.Relationships[1].Matches[0].Keys[0].KeyPath = nil
		}, "relationship \"unused\": invalid key path"},
		"unset path selector": {func(c *proto.PolicySetupResponse_ServerCapabilities) {
			c.Relationships[0].Matches[0].Keys[0].KeyPath.Steps[0].Selector = nil
		}, "missing attribute path selector"},
		"duplicate resource type": {func(c *proto.PolicySetupResponse_ServerCapabilities) {
			c.ResourceRelationships = append(c.ResourceRelationships, c.ResourceRelationships[0])
		}, "duplicate relationship usage tree"},
		"undefined nested reference": {func(c *proto.PolicySetupResponse_ServerCapabilities) {
			c.ResourceRelationships[0].References[0].References = []*proto.ResourceRelationships_Reference{{Relationship: "unknown"}}
		}, "undefined relationship \"unknown\""},
		"wrong nested endpoint": {func(c *proto.PolicySetupResponse_ServerCapabilities) {
			c.Relationships = append(c.Relationships, setupDefinition("a_c", "a", "c"))
			c.ResourceRelationships[0].References[0].References = []*proto.ResourceRelationships_Reference{{Relationship: "a_c"}}
		}, "does not connect to resource type \"b\""},
	} {
		t.Run(name, func(t *testing.T) {
			capabilities := &proto.PolicySetupResponse_ServerCapabilities{
				Relationships:         []*proto.Relationship{setupDefinition("a_b", "a", "b")},
				ResourceRelationships: []*proto.ResourceRelationships{{ResourceType: "a", References: []*proto.ResourceRelationships_Reference{{Relationship: "a_b"}}}},
			}
			tc.change(capabilities)
			config, err := configFromProto(capabilities)
			if err == nil || !strings.Contains(err.Error(), tc.want) || config != nil {
				t.Fatalf("config=%#v err=%v, want nil and %q", config, err, tc.want)
			}
		})
	}
}
