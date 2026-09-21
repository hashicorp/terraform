// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package simplerefs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/globalref"
)

func resourceAttrGraph() *ReferenceGraph {
	return NewReferenceGraph(func(ref *globalref.Reference) bool {
		_, ok := ref.ResourceAttr()
		return ok
	})
}

func exprFor(t *testing.T, src string) hcl.Expression {
	t.Helper()
	f, diags := hclsyntax.ParseConfig([]byte("x = "+src), "t.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("parse %q: %s", src, diags.Error())
	}
	return f.Body.(*hclsyntax.Body).Attributes["x"].Expr
}

func localRef(name string) *globalref.Reference {
	return &globalref.Reference{
		ContainerAddr: addrs.RootModuleInstance,
		LocalRef:      &addrs.Reference{Subject: addrs.LocalValue{Name: name}},
	}
}

// TestSetReference_NonSimpleExpressions verifies that SetReference records a
// source when a non-simple expression provably resolves to a single homogeneous
// resource reference, records a constant marker for a pure constant, and leaves
// undecidable expressions unrecorded (so consumers treat them as unknown).
func TestSetReference_NonSimpleExpressions(t *testing.T) {
	subnetRef := func(g *ReferenceGraph) (*globalref.Reference, bool) {
		// aws_subnet.this.id, as a reference to resolve against.
		return g.ResolveReference(&globalref.Reference{
			ContainerAddr: addrs.RootModuleInstance,
			LocalRef: &addrs.Reference{
				Subject: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_subnet",
					Name: "this",
				},
				Remaining: hcl.Traversal{hcl.TraverseAttr{Name: "id"}},
			},
		})
	}
	_ = subnetRef

	tests := []struct {
		name string
		src  string
		// outcome: "ref" -> resolves to aws_subnet.this.id; "literal" ->
		// resolves-to-constant; "unknown" -> unrecorded.
		outcome string
	}{
		{"splat-element", `element(aws_subnet.this[*].id, 0)`, "ref"},
		{"one-splat", `one(aws_subnet.this[*].id)`, "ref"},
		{"homogeneous-conditional", `var.flag ? aws_subnet.this[0].id : aws_subnet.this[1].id`, "ref"},
		{"pure-constant", `"subnet-abc"`, "literal"},
		{"ref-vs-literal", `var.flag ? aws_subnet.this[0].id : "nothing"`, "unknown"},
		{"heterogeneous", `coalesce(aws_subnet.this.id, aws_subnet.other.id)`, "unknown"},
		{"opaque", `format("%s", aws_subnet.this.id)`, "unknown"},

		// Composition through the graph-population path.
		{"composed-valid", `one(flatten([concat(aws_subnet.this[*].id, aws_subnet.this[*].id)]))`, "ref"},
		{"composed-heterogeneous", `element(concat(aws_subnet.this[*].id, aws_subnet.other[*].id), 0)`, "unknown"},
		{"composed-unsupported-inner", `try(aws_subnet.this.id, upper(aws_subnet.this.id))`, "unknown"},
		{"composed-unsupported-control-arg", `element(aws_subnet.this[*].id, length(aws_subnet.other))`, "ref"},

		// for expressions and object literals through the graph-population path.
		{"list-for", `[for s in aws_subnet.this : s.id]`, "ref"},
		{"for-then-element", `element([for s in aws_subnet.this : s.id], 0)`, "ref"},
		{"object-value-ref", `{ x = aws_subnet.this[0].id }`, "ref"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := resourceAttrGraph()
			src := localRef("out")
			g.SetReference(src, exprFor(t, tc.src), addrs.RootModuleInstance)

			resolved, ok := g.ResolveReference(src)
			literal := g.ResolvesToLiteral(src)

			switch tc.outcome {
			case "ref":
				if !ok {
					t.Fatalf("expected %q to resolve to a resource reference, got unresolved", tc.src)
				}
				ra, _ := resolved.ResourceAttr()
				cr := ra.Resource.ConfigResource()
				if cr.Resource.Type != "aws_subnet" || cr.Resource.Name != "this" {
					t.Fatalf("expected aws_subnet.this, got %s", resolved.DebugString())
				}
			case "literal":
				if ok {
					t.Fatalf("expected %q to be a constant, but it resolved to %s", tc.src, resolved.DebugString())
				}
				if !literal {
					t.Fatalf("expected %q to be recorded as a constant", tc.src)
				}
			case "unknown":
				if ok {
					t.Fatalf("expected %q to be unrecorded, but it resolved to %s", tc.src, resolved.DebugString())
				}
				if literal {
					t.Fatalf("expected %q to be unrecorded (not a constant)", tc.src)
				}
			}
		})
	}
}

func traversalFor(t *testing.T, src string) hcl.Traversal {
	t.Helper()
	tr, diags := hclsyntax.ParseTraversalAbs([]byte(src), "t.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("parse traversal %q: %s", src, diags.Error())
	}
	return tr
}

// TestResolveReference_ThroughExpandedModuleOutput verifies that a reference
// into a count/for_each-expanded module output resolves to the terminal
// resource. The output is recorded under the fully-expanded module instance
// (module.buckets[0]), while the connector references it at config-address
// granularity (module.buckets.bucket_id, index stripped); resolution must
// bridge the two by normalizing instance keys away.
func TestResolveReference_ThroughExpandedModuleOutput(t *testing.T) {
	g := resourceAttrGraph()

	modInst := addrs.RootModuleInstance.Child("buckets", addrs.IntKey(0))
	outRef := &globalref.Reference{
		ContainerAddr: modInst,
		LocalRef:      &addrs.Reference{Subject: addrs.OutputValue{Name: "bucket_id"}},
	}
	g.SetReference(outRef, exprFor(t, "aws_s3_bucket.test.id"), modInst)

	lookup, diags := globalref.ParseRef(addrs.RootModuleInstance, traversalFor(t, "module.buckets.bucket_id"))
	if diags.HasErrors() {
		t.Fatalf("parse lookup ref: %s", diags.Err())
	}

	resolved, ok := g.ResolveReference(lookup)
	if !ok {
		t.Fatal("expected reference through expanded module output to resolve to a resource")
	}
	ra, ok := resolved.ResourceAttr()
	if !ok {
		t.Fatalf("resolved ref is not a resource attribute: %s", resolved.DebugString())
	}
	if got, want := ra.Resource.Resource.Resource.Type, "aws_s3_bucket"; got != want {
		t.Errorf("resolved resource type = %q, want %q", got, want)
	}
	if got, want := ra.Resource.Resource.Resource.Name, "test"; got != want {
		t.Errorf("resolved resource name = %q, want %q", got, want)
	}
	if got, want := ra.Resource.Module.Module().String(), "module.buckets"; got != want {
		t.Errorf("resolved resource module = %q, want %q", got, want)
	}
}
