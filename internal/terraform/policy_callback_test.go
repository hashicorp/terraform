// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/lang/simplerefs"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy/callback"
)

// resourceAttrRefGraph builds a ReferenceGraph using the same terminal selector
// as production (a resource-attribute reference is a terminal).
func resourceAttrRefGraph() *simplerefs.ReferenceGraph {
	return simplerefs.NewReferenceGraph(func(ref *globalref.Reference) bool {
		_, ok := ref.ResourceAttr()
		return ok
	})
}

// mustSimpleBody parses a fragment of HCL into a real hclsyntax body so that
// hclsyntax.ParseSimpleBody (which type-asserts the concrete hclsyntax types)
// can analyze it.
func mustSimpleBody(t *testing.T, src string) hcl.Body {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("failed to parse config body: %s", diags.Error())
	}
	return f.Body
}

// TestPolicyCallbackManager_Match exercises the structural (reference-based)
// matcher that backs require_reference relationships. The key guarantee is that
// a hardcoded value that merely coincides with the subject's attribute is NOT a
// match, while a genuine config reference is.
func TestPolicyCallbackManager_Match(t *testing.T) {
	relatedSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bucket": {Type: cty.String, Optional: true},
		},
	}

	subjectAddr := mustResourceInstanceAddr("aws_s3_bucket.example")
	relatedAddr := mustResourceInstanceAddr("aws_s3_bucket_acl.example")

	subject := &PolicyResource{
		Addr:   subjectAddr,
		Schema: &configschema.Block{},
		Value: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("my-bucket"),
		}),
	}

	// A structural connector: the related resource's "bucket" attribute is
	// expected to reference the subject's "id".
	structuralConn := &callback.RelationshipBlock{
		SubjectType: "aws_s3_bucket",
		RelatedType: "aws_s3_bucket_acl",
		Direction:   callback.DirectionInbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "id", RelatedAttribute: "bucket"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
	}

	newRelated := func(src string) *PolicyResource {
		return &PolicyResource{
			Addr:       relatedAddr,
			ConfigBody: mustSimpleBody(t, src),
			Schema:     relatedSchema,
			Value: cty.ObjectVal(map[string]cty.Value{
				"bucket": cty.StringVal("my-bucket"),
			}),
		}
	}

	tests := []struct {
		name string
		conn *callback.RelationshipBlock
		src  string
		// want is the expected cty.Bool result; use cty.NilVal to mean "unknown".
		want cty.Value
	}{
		{
			name: "genuine reference to subject matches",
			conn: structuralConn,
			src:  `bucket = aws_s3_bucket.example.id`,
			want: cty.True,
		},
		{
			name: "hardcoded coincidental literal does not match",
			conn: structuralConn,
			src:  `bucket = "my-bucket"`,
			want: cty.False,
		},
		{
			name: "reference to a different resource is unknown (no positive match)",
			conn: structuralConn,
			src:  `bucket = aws_s3_bucket.other.id`,
			want: cty.UnknownVal(cty.Bool),
		},
		{
			name: "value fallback matches when query_attributes provided",
			conn: &callback.RelationshipBlock{
				SubjectType: "aws_s3_bucket",
				RelatedType: "aws_s3_bucket_acl",
				Direction:   callback.DirectionInbound,
				AttributePairs: []callback.RelatedAttributePair{
					{SubjectAttribute: "id", RelatedAttribute: "bucket"},
				},
				QueryAttributes: cty.ObjectVal(map[string]cty.Value{
					"bucket": cty.StringVal("my-bucket"),
				}),
			},
			src:  `bucket = "my-bucket"`,
			want: cty.True,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &MockEvalContext{
				ReferenceTreeValue: resourceAttrRefGraph(),
			}
			mgr := &PolicyCallbackManager{}

			got := mgr.Match(ctx, subject, newRelated(tc.src), tc.conn)

			switch {
			case tc.want == cty.UnknownVal(cty.Bool):
				if got.IsKnown() {
					t.Fatalf("expected unknown, got %#v", got)
				}
			default:
				if !got.IsKnown() {
					t.Fatalf("expected known %#v, got unknown", tc.want)
				}
				if !got.RawEquals(tc.want) {
					t.Fatalf("expected %#v, got %#v", tc.want, got)
				}
			}
		})
	}
}

// TestPolicyCallbackManager_Match_ThroughLocal verifies that a connector
// expression that flows through a local value still resolves to the subject via
// the reference graph.
func TestPolicyCallbackManager_Match_ThroughLocal(t *testing.T) {
	relatedSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bucket": {Type: cty.String, Optional: true},
		},
	}

	subjectAddr := mustResourceInstanceAddr("aws_s3_bucket.example")
	relatedAddr := mustResourceInstanceAddr("aws_s3_bucket_acl.example")

	subject := &PolicyResource{
		Addr:   subjectAddr,
		Schema: &configschema.Block{},
		Value:  cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("my-bucket")}),
	}

	conn := &callback.RelationshipBlock{
		SubjectType: "aws_s3_bucket",
		RelatedType: "aws_s3_bucket_acl",
		Direction:   callback.DirectionInbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "id", RelatedAttribute: "bucket"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
	}

	// Populate the graph so that local.bucket_id -> aws_s3_bucket.example.id.
	graph := resourceAttrRefGraph()
	localRef := &globalref.Reference{
		ContainerAddr: addrs.RootModuleInstance,
		LocalRef: &addrs.Reference{
			Subject: addrs.LocalValue{Name: "bucket_id"},
		},
	}
	localExpr := mustSimpleBody(t, `bucket_id = aws_s3_bucket.example.id`).(*hclsyntax.Body).Attributes["bucket_id"].Expr
	graph.SetReference(localRef, localExpr, addrs.RootModuleInstance)

	ctx := &MockEvalContext{ReferenceTreeValue: graph}
	mgr := &PolicyCallbackManager{}

	related := &PolicyResource{
		Addr:       relatedAddr,
		ConfigBody: mustSimpleBody(t, `bucket = local.bucket_id`),
		Schema:     relatedSchema,
		Value:      cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")}),
	}

	got := mgr.Match(ctx, subject, related, conn)
	if !got.IsKnown() || !got.True() {
		t.Fatalf("expected match through local to be true, got %#v", got)
	}
}

// TestPolicyCallbackManager_GetRelatedResources verifies that resolution
// iterates the resource map, includes only genuinely-referencing candidates,
// skips the subject itself, and flattens matched values into the result.
func TestPolicyCallbackManager_GetRelatedResources(t *testing.T) {
	relatedSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bucket": {Type: cty.String, Optional: true},
		},
	}

	subjectAddr := mustResourceInstanceAddr("aws_s3_bucket.example")
	matchAddr := mustResourceInstanceAddr("aws_s3_bucket_acl.example")
	coincidentalAddr := mustResourceInstanceAddr("aws_s3_bucket_acl.other")

	subjectVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("my-bucket")})

	pg := newPolicySubgraph()
	pg.resourceMap.Put(subjectAddr, &PolicyResource{
		Addr:   subjectAddr,
		Schema: &configschema.Block{},
		Value:  subjectVal,
	})
	// A genuine reference to the subject — should be included.
	matchedVal := cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")})
	pg.resourceMap.Put(matchAddr, &PolicyResource{
		Addr:       matchAddr,
		ConfigBody: mustSimpleBody(t, `bucket = aws_s3_bucket.example.id`),
		Schema:     relatedSchema,
		Value:      matchedVal,
	})
	// A hardcoded coincidental literal — should be excluded.
	pg.resourceMap.Put(coincidentalAddr, &PolicyResource{
		Addr:       coincidentalAddr,
		ConfigBody: mustSimpleBody(t, `bucket = "my-bucket"`),
		Schema:     relatedSchema,
		Value:      cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")}),
	})

	ctx := &MockEvalContext{
		ReferenceTreeValue: resourceAttrRefGraph(),
		PolicyGraphValue:   pg,
		DeferralsState:     deferring.NewDeferred(false),
	}

	conn := &callback.RelationshipBlock{
		SubjectType: "aws_s3_bucket",
		RelatedType: "aws_s3_bucket_acl",
		Direction:   callback.DirectionInbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "id", RelatedAttribute: "bucket"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
	}

	mgr := &PolicyCallbackManager{}
	got, err := mgr.GetRelatedResources(ctx, subjectAddr, conn, subjectVal)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got.Partial {
		t.Fatalf("expected non-partial result, got partial")
	}
	if len(got.Related) != 1 {
		t.Fatalf("expected exactly 1 related resource, got %d", len(got.Related))
	}
	if !got.Related[0].Value.RawEquals(matchedVal) {
		t.Fatalf("expected matched value %#v, got %#v", matchedVal, got.Related[0].Value)
	}
}
