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
		// want is the expected cty.Bool result; set wantUnknown for unknown.
		want        cty.Value
		wantUnknown bool
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
			name: "reference to a different resource does not match",
			conn: structuralConn,
			src:  `bucket = aws_s3_bucket.other.id`,
			want: cty.False,
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

			if tc.wantUnknown {
				if got.IsKnown() {
					t.Fatalf("expected unknown, got %#v", got)
				}
				return
			}
			if !got.IsKnown() {
				t.Fatalf("expected known %#v, got unknown", tc.want)
			}
			if !got.RawEquals(tc.want) {
				t.Fatalf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}

// TestPolicyCallbackManager_Match_Expansion verifies that structural provenance
// detection works when the subject and/or target are expanded via count/for_each
// (on the resource or an enclosing module call). Detection is config-address
// granular, so a genuine reference matches regardless of instance keys or dynamic
// index expressions, while a hardcoded literal still fails.
func TestPolicyCallbackManager_Match_Expansion(t *testing.T) {
	relatedSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bucket": {Type: cty.String, Optional: true},
		},
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

	tests := []struct {
		name        string
		subjectAddr string
		relatedAddr string
		src         string
		want        cty.Value
	}{
		{
			name:        "expanded resource via count.index",
			subjectAddr: "aws_s3_bucket.this[2]",
			relatedAddr: "aws_s3_bucket_acl.this[2]",
			src:         `bucket = aws_s3_bucket.this[count.index].id`,
			want:        cty.True,
		},
		{
			name:        "expanded resource via each.key",
			subjectAddr: `aws_s3_bucket.this["prod"]`,
			relatedAddr: `aws_s3_bucket_acl.this["prod"]`,
			src:         `bucket = aws_s3_bucket.this[each.key].id`,
			want:        cty.True,
		},
		{
			name:        "expanded resource via literal key",
			subjectAddr: "aws_s3_bucket.this[0]",
			relatedAddr: "aws_s3_bucket_acl.this[0]",
			src:         `bucket = aws_s3_bucket.this[0].id`,
			want:        cty.True,
		},
		{
			name:        "expanded resource with hardcoded literal does not match",
			subjectAddr: "aws_s3_bucket.this[0]",
			relatedAddr: "aws_s3_bucket_acl.this[0]",
			src:         `bucket = "my-bucket"`,
			want:        cty.False,
		},
		{
			name:        "subject and target inside an expanded module call",
			subjectAddr: `module.team["a"].aws_s3_bucket.this[0]`,
			relatedAddr: `module.team["a"].aws_s3_bucket_acl.this[0]`,
			src:         `bucket = aws_s3_bucket.this[count.index].id`,
			want:        cty.True,
		},
		{
			name:        "reference to a different expanded resource does not match",
			subjectAddr: "aws_s3_bucket.this[0]",
			relatedAddr: "aws_s3_bucket_acl.this[0]",
			src:         `bucket = aws_s3_bucket.other[count.index].id`,
			want:        cty.False,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			subjectAddr := mustResourceInstanceAddr(tc.subjectAddr)
			relatedAddr := mustResourceInstanceAddr(tc.relatedAddr)

			subject := &PolicyResource{
				Addr:   subjectAddr,
				Schema: &configschema.Block{},
				Value:  cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("my-bucket")}),
			}
			related := &PolicyResource{
				Addr:       relatedAddr,
				ConfigBody: mustSimpleBody(t, tc.src),
				Schema:     relatedSchema,
				Value:      cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")}),
			}

			ctx := &MockEvalContext{ReferenceTreeValue: resourceAttrRefGraph()}
			mgr := &PolicyCallbackManager{}

			got := mgr.Match(ctx, subject, related, conn)
			if !got.IsKnown() {
				t.Fatalf("expected known %#v, got unknown", tc.want)
			}
			if !got.RawEquals(tc.want) {
				t.Fatalf("expected %#v, got %#v", tc.want, got)
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

// TestPolicyCallbackManager_Match_ThroughLiteralLocal verifies that a connector
// reference laundered through a local (or module output) that holds a hardcoded
// literal is a definitive non-match (cty.False), not unknown — so a coincidental
// value cannot evade provenance by hiding behind a local/output.
func TestPolicyCallbackManager_Match_ThroughLiteralLocal(t *testing.T) {
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

	// local.bucket_id = "my-bucket" (a hardcoded literal).
	graph := resourceAttrRefGraph()
	localRef := &globalref.Reference{
		ContainerAddr: addrs.RootModuleInstance,
		LocalRef:      &addrs.Reference{Subject: addrs.LocalValue{Name: "bucket_id"}},
	}
	localExpr := mustSimpleBody(t, `bucket_id = "my-bucket"`).(*hclsyntax.Body).Attributes["bucket_id"].Expr
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
	if !got.IsKnown() || !got.False() {
		t.Fatalf("expected literal-laundered reference to be a definitive non-match (false), got %#v", got)
	}
}

// TestPolicyCallbackManager_Match_ThroughChainedLiteralLocal verifies that a
// constant laundered through MORE THAN ONE intermediate (local.outer ->
// local.inner -> "literal") still resolves to a definitive non-match, because
// SetReference propagates constant-ness through the graph the same way it
// propagates resource references.
func TestPolicyCallbackManager_Match_ThroughChainedLiteralLocal(t *testing.T) {
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

	graph := resourceAttrRefGraph()
	// Record inner first (as the walk would): local.inner = "my-bucket".
	innerRef := &globalref.Reference{
		ContainerAddr: addrs.RootModuleInstance,
		LocalRef:      &addrs.Reference{Subject: addrs.LocalValue{Name: "inner"}},
	}
	innerExpr := mustSimpleBody(t, `inner = "my-bucket"`).(*hclsyntax.Body).Attributes["inner"].Expr
	graph.SetReference(innerRef, innerExpr, addrs.RootModuleInstance)
	// local.outer = local.inner (a pass-through to the constant).
	outerRef := &globalref.Reference{
		ContainerAddr: addrs.RootModuleInstance,
		LocalRef:      &addrs.Reference{Subject: addrs.LocalValue{Name: "outer"}},
	}
	outerExpr := mustSimpleBody(t, `outer = local.inner`).(*hclsyntax.Body).Attributes["outer"].Expr
	graph.SetReference(outerRef, outerExpr, addrs.RootModuleInstance)

	ctx := &MockEvalContext{ReferenceTreeValue: graph}
	mgr := &PolicyCallbackManager{}

	related := &PolicyResource{
		Addr:       relatedAddr,
		ConfigBody: mustSimpleBody(t, `bucket = local.outer`),
		Schema:     relatedSchema,
		Value:      cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")}),
	}

	got := mgr.Match(ctx, subject, related, conn)
	if !got.IsKnown() || !got.False() {
		t.Fatalf("expected chained literal-laundered reference to be a definitive non-match (false), got %#v", got)
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

// TestPolicyCallbackManager_GetRelatedResources_MultiHop verifies multi-hop
// provenance through a nested RelationshipBlock: an inbound chain
// res_a <- res_b <- res_c (each hop's target references the previous hop's
// resource). The result exposes the terminal (res_c) targets, and an outer
// candidate whose chain does not reach a terminal (a broken chain) is pruned.
func TestPolicyCallbackManager_GetRelatedResources_MultiHop(t *testing.T) {
	parentSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"parent": {Type: cty.String, Optional: true},
		},
	}

	subjectAddr := mustResourceInstanceAddr("res_a.this")
	b1Addr := mustResourceInstanceAddr("res_b.this")     // references subject (hop 1)
	b2Addr := mustResourceInstanceAddr("res_b.other")    // references subject (hop 1) but has no hop-2 target
	c1Addr := mustResourceInstanceAddr("res_c.this")     // references res_b.this (hop 2, terminal)

	subjectVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("a1")})
	c1Val := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("c1"), "parent": cty.StringVal("b1")})

	pg := newPolicySubgraph()
	pg.resourceMap.Put(subjectAddr, &PolicyResource{
		Addr:   subjectAddr,
		Schema: &configschema.Block{},
		Value:  subjectVal,
	})
	pg.resourceMap.Put(b1Addr, &PolicyResource{
		Addr:       b1Addr,
		ConfigBody: mustSimpleBody(t, `parent = res_a.this.id`),
		Schema:     parentSchema,
		Value:      cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("b1"), "parent": cty.StringVal("a1")}),
	})
	pg.resourceMap.Put(b2Addr, &PolicyResource{
		Addr:       b2Addr,
		ConfigBody: mustSimpleBody(t, `parent = res_a.this.id`),
		Schema:     parentSchema,
		Value:      cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("b2"), "parent": cty.StringVal("a1")}),
	})
	pg.resourceMap.Put(c1Addr, &PolicyResource{
		Addr:       c1Addr,
		ConfigBody: mustSimpleBody(t, `parent = res_b.this.id`),
		Schema:     parentSchema,
		Value:      c1Val,
	})

	ctx := &MockEvalContext{
		ReferenceTreeValue: resourceAttrRefGraph(),
		PolicyGraphValue:   pg,
		DeferralsState:     deferring.NewDeferred(false),
	}

	conn := &callback.RelationshipBlock{
		SubjectType: "res_a",
		RelatedType: "res_b",
		Direction:   callback.DirectionInbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "id", RelatedAttribute: "parent"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
		Nested: &callback.RelationshipBlock{
			SubjectType: "res_b",
			RelatedType: "res_c",
			Direction:   callback.DirectionInbound,
			AttributePairs: []callback.RelatedAttributePair{
				{SubjectAttribute: "id", RelatedAttribute: "parent"},
			},
			QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
		},
	}

	mgr := &PolicyCallbackManager{}
	got, err := mgr.GetRelatedResources(ctx, subjectAddr, conn, subjectVal)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got.Partial {
		t.Fatalf("expected non-partial result, got partial")
	}
	// Only the terminal res_c.this should surface; res_b.other's broken chain
	// (no res_c references it) is pruned.
	if len(got.Related) != 1 {
		t.Fatalf("expected exactly 1 terminal related resource, got %d: %#v", len(got.Related), got.Related)
	}
	if !got.Related[0].Value.RawEquals(c1Val) {
		t.Fatalf("expected terminal value %#v, got %#v", c1Val, got.Related[0].Value)
	}
}

// TestPolicyCallbackManager_Match_NonSimpleExpressions exercises the decidable
// subset of non-simple connector expressions (splats, reference-preserving
// functions, homogeneous conditionals) that should be recovered into a
// definite match/non-match, and confirms the genuinely undecidable ones
// (value-conditional selection among differing outcomes) stay unknown.
func TestPolicyCallbackManager_Match_NonSimpleExpressions(t *testing.T) {
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

	const (
		wantTrue = iota
		wantFalse
		wantUnknown
	)
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"splat-element", `bucket = element(aws_s3_bucket.example[*].id, 0)`, wantTrue},
		{"one-splat", `bucket = one(aws_s3_bucket.example[*].id)`, wantTrue},
		{"try-homogeneous", `bucket = try(aws_s3_bucket.example[0].id, aws_s3_bucket.example[1].id)`, wantTrue},
		{"conditional-homogeneous", `bucket = var.flag ? aws_s3_bucket.example[0].id : aws_s3_bucket.example[1].id`, wantTrue},
		{"dynamic-index", `bucket = aws_s3_bucket.example[var.i].id`, wantTrue},
		{"concat-flatten-homogeneous", `bucket = element(flatten([aws_s3_bucket.example[*].id]), 0)`, wantTrue},
		{"different-resource", `bucket = element(aws_s3_bucket.other[*].id, 0)`, wantFalse},
		{"conditional-ref-vs-literal", `bucket = var.flag ? aws_s3_bucket.example.id : "nothing"`, wantUnknown},
		{"conditional-ref-vs-diff", `bucket = var.flag ? aws_s3_bucket.example.id : aws_s3_bucket.other.id`, wantUnknown},
		{"coalesce-heterogeneous", `bucket = coalesce(aws_s3_bucket.example.id, aws_s3_bucket.other.id)`, wantUnknown},
		{"opaque-function", `bucket = format("%s", aws_s3_bucket.example.id)`, wantUnknown},

		// Composition: valid functions nested arbitrarily still resolve, and a
		// single unsupported (opaque) value-contributing sub-expression anywhere
		// forces the whole thing to unknown.
		{"composed-valid-nested", `bucket = one(flatten([concat(aws_s3_bucket.example[*].id, aws_s3_bucket.example[*].id)]))`, wantTrue},
		{"composed-valid-deep", `bucket = element(distinct(concat(aws_s3_bucket.example[*].id, aws_s3_bucket.example[*].id)), 0)`, wantTrue},
		{"composed-heterogeneous", `bucket = element(concat(aws_s3_bucket.example[*].id, aws_s3_bucket.other[*].id), 0)`, wantUnknown},
		{"composed-unsupported-inner", `bucket = try(aws_s3_bucket.example.id, upper(aws_s3_bucket.example.id))`, wantUnknown},
		{"composed-unsupported-outer", `bucket = upper(element(aws_s3_bucket.example[*].id, 0))`, wantUnknown},
		{"composed-unsupported-control-arg", `bucket = element(aws_s3_bucket.example[*].id, length(aws_s3_bucket.other))`, wantTrue},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &MockEvalContext{ReferenceTreeValue: resourceAttrRefGraph()}
			mgr := &PolicyCallbackManager{}
			related := &PolicyResource{
				Addr:       relatedAddr,
				ConfigBody: mustSimpleBody(t, tc.src),
				Schema:     relatedSchema,
				Value:      cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")}),
			}
			got := mgr.Match(ctx, subject, related, conn)
			switch tc.want {
			case wantUnknown:
				if got.IsKnown() {
					t.Fatalf("expected unknown, got %#v", got)
				}
			case wantTrue:
				if !got.IsKnown() || !got.True() {
					t.Fatalf("expected true, got %#v", got)
				}
			case wantFalse:
				if !got.IsKnown() || !got.False() {
					t.Fatalf("expected false, got %#v", got)
				}
			}
		})
	}
}

// TestPolicyCallbackManager_Match_LanguageCoverage walks the Terraform
// expression language (per the public "Expressions" docs) category by category
// and asserts that every form maps to a SOUND provenance outcome: a supported
// shape resolves to a definite reference/non-reference, and every other form is
// unknown. The critical invariant is that no expression is ever mis-classified
// as a definite match/deny when it isn't — anything not provably a single
// resource reference or a pure constant must defer.
//
// References that require the reference graph to resolve (var/local/module
// output) are unknown here because this test uses an empty graph; their
// resolution is covered by the graph-population tests and e2e. The subject is
// aws_s3_bucket.example.id (a managed resource attribute).
func TestPolicyCallbackManager_Match_LanguageCoverage(t *testing.T) {
	relatedSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"bucket": {Type: cty.String, Optional: true},
		},
	}
	subject := &PolicyResource{
		Addr:   mustResourceInstanceAddr("aws_s3_bucket.example"),
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

	const (
		wantTrue = iota
		wantFalse
		wantUnknown
	)
	tests := []struct {
		category string
		name     string
		src      string
		want     int
	}{
		// --- Types and Values (literals) ---
		{"types", "string-literal", `bucket = "my-bucket"`, wantFalse},
		{"types", "number-literal", `bucket = 42`, wantFalse},
		{"types", "bool-literal", `bucket = true`, wantFalse},
		{"types", "null-literal", `bucket = null`, wantFalse},
		{"types", "tuple-of-literals", `bucket = ["a", "b"]`, wantFalse},
		{"types", "tuple-with-ref", `bucket = [aws_s3_bucket.example.id]`, wantTrue},
		{"types", "object-with-ref-value", `bucket = { x = aws_s3_bucket.example.id }`, wantTrue},
		{"types", "object-heterogeneous", `bucket = { x = aws_s3_bucket.example.id, y = aws_s3_bucket.other.id }`, wantUnknown},
		{"types", "object-of-literals", `bucket = { x = "a" }`, wantFalse},

		// --- Strings and Templates ---
		{"strings", "single-interpolation", `bucket = "${aws_s3_bucket.example.id}"`, wantTrue},
		{"strings", "interpolated-template", `bucket = "prefix-${aws_s3_bucket.example.id}"`, wantUnknown},
		{"strings", "heredoc-literal", "bucket = <<-EOT\n  static\n  EOT\n", wantFalse},

		// --- References to Named Values ---
		{"references", "resource-attr", `bucket = aws_s3_bucket.example.id`, wantTrue},
		{"references", "resource-indexed", `bucket = aws_s3_bucket.example[0].id`, wantTrue},
		{"references", "data-source", `bucket = data.aws_s3_bucket.example.id`, wantFalse},
		{"references", "count-index", `bucket = count.index`, wantUnknown},
		{"references", "each-key", `bucket = each.key`, wantUnknown},
		{"references", "self", `bucket = self.id`, wantUnknown},
		{"references", "path-module", `bucket = path.module`, wantUnknown},
		{"references", "terraform-workspace", `bucket = terraform.workspace`, wantUnknown},
		{"references", "var", `bucket = var.name`, wantUnknown},
		{"references", "local", `bucket = local.name`, wantUnknown},
		{"references", "module-output", `bucket = module.m.name`, wantUnknown},

		// --- Operators ---
		{"operators", "arithmetic", `bucket = aws_s3_bucket.example.count + 1`, wantUnknown},
		{"operators", "equality", `bucket = aws_s3_bucket.example.id == "x"`, wantUnknown},
		{"operators", "comparison", `bucket = aws_s3_bucket.example.size > 1`, wantUnknown},
		{"operators", "logical-and", `bucket = aws_s3_bucket.example.enabled && true`, wantUnknown},
		{"operators", "unary-not", `bucket = !aws_s3_bucket.example.enabled`, wantUnknown},

		// --- Function Calls ---
		{"functions", "supported", `bucket = element(aws_s3_bucket.example[*].id, 0)`, wantTrue},
		{"functions", "unsupported", `bucket = md5(aws_s3_bucket.example.id)`, wantUnknown},

		// --- Conditional Expressions ---
		{"conditionals", "homogeneous", `bucket = var.flag ? aws_s3_bucket.example[0].id : aws_s3_bucket.example[1].id`, wantTrue},
		{"conditionals", "ref-vs-literal", `bucket = var.flag ? aws_s3_bucket.example.id : "x"`, wantUnknown},

		// --- For Expressions ---
		{"for", "list-for", `bucket = [for s in aws_s3_bucket.example : s.id]`, wantTrue},
		{"for", "list-for-indexed", `bucket = [for i, s in aws_s3_bucket.example : s.id]`, wantTrue},
		{"for", "list-for-different", `bucket = [for s in aws_s3_bucket.example : aws_s3_bucket.other.id]`, wantFalse},
		{"for", "list-for-opaque", `bucket = [for s in aws_s3_bucket.example : upper(s.id)]`, wantUnknown},
		{"for", "object-for", `bucket = { for s in aws_s3_bucket.example : s.id => s.id }`, wantTrue},
		{"for", "for-then-element", `bucket = element([for s in aws_s3_bucket.example : s.id], 0)`, wantTrue},

		// --- Splat Expressions ---
		{"splat", "attr-splat", `bucket = element(aws_s3_bucket.example[*].id, 0)`, wantTrue},
		{"splat", "full-splat", `bucket = element(aws_s3_bucket.example[*].id, 0)`, wantTrue},
	}

	for _, tc := range tests {
		t.Run(tc.category+"/"+tc.name, func(t *testing.T) {
			ctx := &MockEvalContext{ReferenceTreeValue: resourceAttrRefGraph()}
			mgr := &PolicyCallbackManager{}
			related := &PolicyResource{
				Addr:       mustResourceInstanceAddr("aws_s3_bucket_acl.example"),
				ConfigBody: mustSimpleBody(t, tc.src),
				Schema:     relatedSchema,
				Value:      cty.ObjectVal(map[string]cty.Value{"bucket": cty.StringVal("my-bucket")}),
			}
			got := mgr.Match(ctx, subject, related, conn)
			switch tc.want {
			case wantUnknown:
				if got.IsKnown() {
					t.Fatalf("expected unknown, got %#v", got)
				}
			case wantTrue:
				if !got.IsKnown() || !got.True() {
					t.Fatalf("expected true, got %#v", got)
				}
			case wantFalse:
				if !got.IsKnown() || !got.False() {
					t.Fatalf("expected false, got %#v", got)
				}
			}
		})
	}
}

// TestPolicyCallbackManager_Match_Outbound verifies that Match detects an
// outbound reference, where the subject config references the target
// (aws_instance.subnet_id = aws_subnet.this.id), not just the inbound case.
func TestPolicyCallbackManager_Match_Outbound(t *testing.T) {
	subjectSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"subnet_id": {Type: cty.String, Optional: true},
		},
	}
	subjectAddr := mustResourceInstanceAddr("aws_instance.this")
	relatedAddr := mustResourceInstanceAddr("aws_subnet.this")

	related := &PolicyResource{
		Addr:       relatedAddr,
		ConfigBody: mustSimpleBody(t, ``),
		Schema:     &configschema.Block{},
		Value:      cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("subnet-1")}),
	}

	conn := &callback.RelationshipBlock{
		SubjectType: "aws_instance",
		RelatedType: "aws_subnet",
		Direction:   callback.DirectionOutbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "subnet_id", RelatedAttribute: "id"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
	}

	tests := []struct {
		name string
		src  string
		want cty.Value
	}{
		{
			name: "subject references target matches",
			src:  `subnet_id = aws_subnet.this.id`,
			want: cty.True,
		},
		{
			name: "hardcoded literal does not match",
			src:  `subnet_id = "subnet-1"`,
			want: cty.False,
		},
		{
			name: "subject references a different target does not match",
			src:  `subnet_id = aws_subnet.other.id`,
			want: cty.False,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			subject := &PolicyResource{
				Addr:       subjectAddr,
				ConfigBody: mustSimpleBody(t, tc.src),
				Schema:     subjectSchema,
				Value:      cty.ObjectVal(map[string]cty.Value{"subnet_id": cty.StringVal("subnet-1")}),
			}
			ctx := &MockEvalContext{ReferenceTreeValue: resourceAttrRefGraph()}
			mgr := &PolicyCallbackManager{}
			got := mgr.Match(ctx, subject, related, conn)
			if !got.IsKnown() {
				t.Fatalf("expected known %#v, got unknown", tc.want)
			}
			if !got.RawEquals(tc.want) {
				t.Fatalf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}

// TestPolicyCallbackManager_GetRelatedResources_MultiHopOutbound verifies the
// canonical outbound multi-hop chain aws_instance -> aws_subnet -> aws_vpc,
// where each resource references the next (outbound at every hop), resolves to
// the terminal aws_vpc through the nested block.
func TestPolicyCallbackManager_GetRelatedResources_MultiHopOutbound(t *testing.T) {
	subnetSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"vpc_id": {Type: cty.String, Optional: true},
		},
	}

	instanceAddr := mustResourceInstanceAddr("aws_instance.this")
	subnetAddr := mustResourceInstanceAddr("aws_subnet.this")
	vpcAddr := mustResourceInstanceAddr("aws_vpc.this")

	instanceVal := cty.ObjectVal(map[string]cty.Value{"subnet_id": cty.StringVal("subnet-1")})
	vpcVal := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("vpc-1")})

	pg := newPolicySubgraph()
	pg.resourceMap.Put(instanceAddr, &PolicyResource{
		Addr:       instanceAddr,
		ConfigBody: mustSimpleBody(t, `subnet_id = aws_subnet.this.id`),
		Schema: &configschema.Block{Attributes: map[string]*configschema.Attribute{
			"subnet_id": {Type: cty.String, Optional: true},
		}},
		Value: instanceVal,
	})
	pg.resourceMap.Put(subnetAddr, &PolicyResource{
		Addr:       subnetAddr,
		ConfigBody: mustSimpleBody(t, `vpc_id = aws_vpc.this.id`),
		Schema:     subnetSchema,
		Value:      cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("subnet-1"), "vpc_id": cty.StringVal("vpc-1")}),
	})
	pg.resourceMap.Put(vpcAddr, &PolicyResource{
		Addr:       vpcAddr,
		ConfigBody: mustSimpleBody(t, ``),
		Schema:     &configschema.Block{},
		Value:      vpcVal,
	})

	ctx := &MockEvalContext{
		ReferenceTreeValue: resourceAttrRefGraph(),
		PolicyGraphValue:   pg,
		DeferralsState:     deferring.NewDeferred(false),
	}

	conn := &callback.RelationshipBlock{
		SubjectType: "aws_instance",
		RelatedType: "aws_subnet",
		Direction:   callback.DirectionOutbound,
		AttributePairs: []callback.RelatedAttributePair{
			{SubjectAttribute: "subnet_id", RelatedAttribute: "id"},
		},
		QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
		Nested: &callback.RelationshipBlock{
			SubjectType: "aws_subnet",
			RelatedType: "aws_vpc",
			Direction:   callback.DirectionOutbound,
			AttributePairs: []callback.RelatedAttributePair{
				{SubjectAttribute: "vpc_id", RelatedAttribute: "id"},
			},
			QueryAttributes: cty.NullVal(cty.DynamicPseudoType),
		},
	}

	mgr := &PolicyCallbackManager{}
	got, err := mgr.GetRelatedResources(ctx, instanceAddr, conn, instanceVal)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got.Partial {
		t.Fatalf("expected non-partial result, got partial")
	}
	if len(got.Related) != 1 {
		t.Fatalf("expected exactly 1 terminal (aws_vpc), got %d: %#v", len(got.Related), got.Related)
	}
	if !got.Related[0].Value.RawEquals(vpcVal) {
		t.Fatalf("expected terminal vpc %#v, got %#v", vpcVal, got.Related[0].Value)
	}
}
