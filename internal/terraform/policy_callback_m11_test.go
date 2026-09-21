// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/policy/callback"
)

// TestPolicyCallbackManager_Undecidable_Records verifies the M11 classifier: a
// structurally undecidable connector expression (an opaque function, a value
// that may reference more than one resource, or a reference mixed with a
// constant) is recorded with the exact source location and a reason, while a
// genuine single reference records nothing. The recorded locations drive the
// closed-by-default plan-halt in node_policy_resource.
func TestPolicyCallbackManager_Undecidable_Records(t *testing.T) {
	subjectSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"subnet_id": {Type: cty.String, Optional: true},
		},
	}

	subjectAddr := mustResourceInstanceAddr("aws_network_interface.this")
	relatedAddr := mustResourceInstanceAddr("aws_subnet.a")

	conn := &callback.RelationshipBlock{
		SubjectType: "aws_network_interface",
		RelatedType: "aws_subnet",
		Direction:   callback.DirectionOutbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "subnet_id", RelatedAttribute: "id"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
	}

	related := &PolicyResource{
		Addr:   relatedAddr,
		Schema: &configschema.Block{},
		Value:  cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("subnet-a")}),
		// nil ConfigBody: the inbound direction is not evaluable, so only the
		// subject's (outbound) expression is classified.
	}

	cases := map[string]struct {
		body       string
		wantRecord bool
		wantReason string
	}{
		"opaque_function": {
			body:       `subnet_id = upper(aws_subnet.a.id)`,
			wantRecord: true,
			wantReason: "unsupported function",
		},
		"two_distinct_resources": {
			body:       `subnet_id = true ? aws_subnet.a.id : aws_subnet.b.id`,
			wantRecord: true,
			wantReason: "more than one resource",
		},
		"reference_mixed_with_constant": {
			body:       `subnet_id = true ? aws_subnet.a.id : "subnet-hardcoded"`,
			wantRecord: true,
			wantReason: "reference or a constant",
		},
		"single_reference_ok": {
			body:       `subnet_id = aws_subnet.a.id`,
			wantRecord: false,
		},
		"hardcoded_literal_ok": {
			// A pure constant is a decided non-reference (deny), not undecidable.
			body:       `subnet_id = "subnet-hardcoded"`,
			wantRecord: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			subject := &PolicyResource{
				Addr:       subjectAddr,
				ConfigBody: mustSimpleBody(t, tc.body),
				Schema:     subjectSchema,
				Value:      cty.ObjectVal(map[string]cty.Value{"subnet_id": cty.StringVal("subnet-a")}),
			}
			ctx := &MockEvalContext{ReferenceTreeValue: resourceAttrRefGraph()}
			mgr := &PolicyCallbackManager{}

			var acc []undecidableRef
			mgr.matchWithAcc(ctx, subject, related, conn, &acc)

			if tc.wantRecord {
				if len(acc) != 1 {
					t.Fatalf("expected 1 undecidable record, got %d: %+v", len(acc), acc)
				}
				if acc[0].attr != "subnet_id" {
					t.Errorf("recorded attr = %q, want %q", acc[0].attr, "subnet_id")
				}
				if !strings.Contains(acc[0].reason, tc.wantReason) {
					t.Errorf("recorded reason = %q, want it to contain %q", acc[0].reason, tc.wantReason)
				}
				if acc[0].rng.Filename != "test.tf" {
					t.Errorf("recorded range filename = %q, want test.tf (an exact source location)", acc[0].rng.Filename)
				}
			} else if len(acc) != 0 {
				t.Fatalf("expected no undecidable records, got %d: %+v", len(acc), acc)
			}
		})
	}
}
