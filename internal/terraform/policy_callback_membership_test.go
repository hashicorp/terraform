// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/policy/callback"
)

// TestPolicyCallbackManager_Match_Membership verifies expansion-connector
// provenance (M10e): a collection-valued connector expression (a tuple of
// references) matches a candidate that is genuinely among its members, and does
// not match a candidate that is only a hardcoded string in the list or absent.
func TestPolicyCallbackManager_Match_Membership(t *testing.T) {
	subjectSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"subnet_ids": {Type: cty.List(cty.String), Optional: true},
		},
	}

	subjectAddr := mustResourceInstanceAddr("aws_db_subnet_group.this")

	// subject.subnet_ids = [aws_subnet.a.id, aws_subnet.b.id, "subnet-hardcoded"]
	subject := &PolicyResource{
		Addr:       subjectAddr,
		ConfigBody: mustSimpleBody(t, `subnet_ids = [aws_subnet.a.id, aws_subnet.b.id, "subnet-hardcoded"]`),
		Schema:     subjectSchema,
		Value: cty.ObjectVal(map[string]cty.Value{
			"subnet_ids": cty.ListVal([]cty.Value{cty.StringVal("subnet-a")}),
		}),
	}

	conn := &callback.RelationshipBlock{
		SubjectType: "aws_db_subnet_group",
		RelatedType: "aws_subnet",
		Direction:   callback.DirectionOutbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "subnet_ids", RelatedAttribute: "id"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
	}

	cases := map[string]struct {
		candidate string
		wantTrue  bool
	}{
		"member a":            {candidate: "aws_subnet.a", wantTrue: true},
		"member b":            {candidate: "aws_subnet.b", wantTrue: true},
		"not referenced":      {candidate: "aws_subnet.c", wantTrue: false},
		"hardcoded not a ref": {candidate: "aws_subnet.hardcoded", wantTrue: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			related := &PolicyResource{
				Addr:   mustResourceInstanceAddr(tc.candidate),
				Schema: &configschema.Block{},
				Value:  cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("subnet-x")}),
				// nil ConfigBody: the inbound direction is not evaluable, so the
				// outbound (membership) direction decides.
			}
			ctx := &MockEvalContext{ReferenceTreeValue: resourceAttrRefGraph()}
			mgr := &PolicyCallbackManager{}
			got := mgr.Match(ctx, subject, related, conn)
			if tc.wantTrue {
				if !got.IsKnown() || !got.True() {
					t.Fatalf("expected membership match true for %s, got %#v", tc.candidate, got)
				}
			} else {
				if got.IsKnown() && got.True() {
					t.Fatalf("expected no membership match for %s, got true", tc.candidate)
				}
			}
		})
	}
}
