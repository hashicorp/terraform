// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/lang/simplerefs"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

type PolicyCallbackManager struct {
	WalkOperation walkOperation
	Schema        providers.GetProviderSchemaResponse
	Config        *configs.Config

	// resources is a map of resource addresses to their policy resources.
	resources addrs.Map[addrs.AbsResourceInstance, *PolicyResource]
}

func NewPolicyCallbackManager(walkOperation walkOperation, schema providers.GetProviderSchemaResponse, config *configs.Config) *PolicyCallbackManager {
	return &PolicyCallbackManager{
		WalkOperation: walkOperation,
		Schema:        schema,
		Config:        config,
	}
}

// PolicyResource co-locates the data required for the relationship analysis for a single resource
type PolicyResource struct {
	Addr       addrs.AbsResourceInstance
	ConfigBody hcl.Body
	Schema     *configschema.Block
	Value      cty.Value
}

func (cb *PolicyCallbackManager) RelatedResourcesCallback(ctx EvalContext, subjectAddr addrs.AbsResourceInstance, val cty.Value) func(context.Context, *callback.RelationshipBlock) (callback.RelatedResource, error) {
	return func(_ context.Context, blk *callback.RelationshipBlock) (callback.RelatedResource, error) {
		related, err := cb.GetRelatedResources(ctx, subjectAddr, blk, val)
		return related, err
	}
}

// GetRelatedResources returns the related resources for the given target resource type and connection.
func (cb *PolicyCallbackManager) GetRelatedResources(ctx EvalContext, subjectAddr addrs.AbsResourceInstance, blk *callback.RelationshipBlock, val cty.Value) (callback.RelatedResource, error) {
	found := make([]callback.RelatedResource, 0)
	partial := false
	var err error
	policyGraph := ctx.PolicyGraph()
	subjectResource := policyGraph.GetResource(subjectAddr)

	// Consider an example where the terraform config is:
	// resource "aws_s3_bucket" "example" {
	//   bucket = "my-bucket"
	// }
	// resource "aws_s3_bucket_acl" "example" {
	//   bucket = aws_s3_bucket.example.id
	// }
	// and the relationship block pair is
	// { sourceAttribute: "id", relatedAttribute: "bucket" }

	for addr, related := range policyGraph.resourceMap.Iter() {
		relatedAddr := addr.ConfigResource()
		if relatedAddr.Resource.Type != blk.RelatedType {
			continue
		}

		// Skip the subject resource, i.e aws_s3_bucket.example
		if relatedAddr.Equal(subjectAddr.ConfigResource()) {
			continue
		}

		// Deferred candidates make the overall answer incomplete.
		if ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{relatedAddr}) {
			partial = true
			continue
		}

		// If the current iteration is for aws_s3_bucket_acl.example, we will
		// check for the given related attribute pair to match aws_s3_bucket.example.
		// We do that by checking if the related attribute (e.g. bucket) is a literal value
		// or a simple traversal.
		// If it is a literal value, we check if it matches relationship.QueryAttributes.
		// If it is a traversal, we check if the traversal points to aws_s3_bucket.example.id.
		resourceValue := related.Value
		matched := cb.Match(ctx, subjectResource, related, blk)
		if matched.IsWhollyKnown() && matched.True() {
			resourceValue, _ = related.Value.UnmarkDeep()

			// If the resource matched and the relationship has a nested block,
			// this is a multi-hop relationship: recurse with the matched
			// candidate as the new subject and collapse the recursion to the
			// terminal targets of the deepest hop. An outer candidate that
			// reaches no terminal (a broken chain) therefore contributes
			// nothing to related.<name>, which is the correct multi-hop
			// existence semantic. When there is no nested block, the matched
			// candidate itself is the terminal target.
			if blk.Nested != nil {
				nestedRes, nestedErr := cb.GetRelatedResources(ctx, related.Addr, blk.Nested, resourceValue)
				if nestedErr != nil {
					err = nestedErr
					continue
				}
				found = append(found, nestedRes.Related...)
				partial = partial || nestedRes.Partial
			} else {
				found = append(found, callback.RelatedResource{Value: resourceValue})
			}
		}
		partial = partial || !matched.IsWhollyKnown()
	}

	return callback.RelatedResource{
		Related: found,
		Partial: partial,
		Value:   val,
	}, err
}

// matchOutcome is the result of checking one reference direction of a
// relationship connector. It is a four-valued logic so the two directions can
// be combined without a "not evaluable" direction (a missing config body)
// spuriously forcing the overall result to unknown.
type matchOutcome int

const (
	// matchNA means this direction could not be evaluated at all (the config
	// body it needs is unavailable). It is ignored when combining directions.
	matchNA matchOutcome = iota
	matchTrue
	matchFalse
	matchUnknown
)

// Match reports whether the related candidate is genuinely connected to the
// subject by a config reference on the connector's attribute pairs.
//
// A connector is an *undirected* reference edge: the reference may live on
// either resource. We therefore check both directions and OR the outcomes:
//   - Inbound: the target (related) config references the subject
//     (e.g. aws_s3_bucket_acl.bucket = aws_s3_bucket.this.id).
//   - Outbound: the subject config references the target (related)
//     (e.g. aws_instance.subnet_id = aws_subnet.this.id).
//
// This makes provenance hold regardless of which side holds the reference, and
// lets multi-hop chains verify each hop structurally in either direction
// (Terraform Core's GetRelatedResources recursion calls Match per hop).
func (c *PolicyCallbackManager) Match(ctx EvalContext, subject, related *PolicyResource, conn *callback.RelationshipBlock) cty.Value {
	// Value fallback (direction-agnostic): a non-null QueryAttributes filter
	// matches the candidate by value, bypassing the reference comparison.
	if !conn.QueryAttributes.IsNull() {
		filterMap := conn.QueryAttributes.AsValueMap()
		if matches, _ := resourceMatchesFilter(related.Addr.ConfigResource(), related.Schema, filterMap, related.Value); matches {
			return cty.True
		}
	}

	inbound := c.checkDirection(ctx, subject, related, conn, false)
	if inbound == matchTrue {
		return cty.True
	}
	outbound := c.checkDirection(ctx, subject, related, conn, true)
	return combineMatch(inbound, outbound)
}

// checkDirection verifies a single reference direction of the connector. When
// outbound is false (inbound) it parses the related (target) config and expects
// each connector's related attribute to reference the subject's subject
// attribute. When outbound is true it parses the subject config and expects the
// subject attribute to reference the related (target) attribute. All attribute
// pairs must match (logical AND).
func (c *PolicyCallbackManager) checkDirection(ctx EvalContext, subject, related *PolicyResource, conn *callback.RelationshipBlock, outbound bool) matchOutcome {
	bodyResource := related
	if outbound {
		bodyResource = subject
	}
	if bodyResource.ConfigBody == nil {
		return matchNA
	}

	bodySimple, diags := hclsyntax.ParseSimpleBody(bodyResource.ConfigBody)
	if diags.HasErrors() {
		return matchUnknown
	}

	for _, pair := range conn.AttributePairs {
		// Assign the attribute-on-body and the attribute-on-target for this
		// direction. Inbound: body=related.RelatedAttribute -> subject.SubjectAttribute.
		// Outbound: body=subject.SubjectAttribute -> related.RelatedAttribute.
		bodyAttr, targetAttr := pair.RelatedAttribute, pair.SubjectAttribute
		targetResource := subject
		if outbound {
			bodyAttr, targetAttr = pair.SubjectAttribute, pair.RelatedAttribute
			targetResource = related
		}

		// The target must actually have the attribute we compare against.
		if !targetResource.Value.Type().IsObjectType() || !targetResource.Value.Type().HasAttribute(targetAttr) {
			return matchUnknown
		}

		bodyTraversal, _ := hclsyntax.ParseTraversalAbs([]byte(bodyAttr), "", hcl.InitialPos)
		path, _ := traversalToPath(bodyTraversal)
		bodyExpr, found := getAttributeFromBody(bodySimple, path, bodyResource.Schema)
		if !found {
			// The connector attribute is not set in this config, so it cannot
			// carry a reference in this direction.
			return matchFalse
		}

		// Compare at config-address granularity (see equalRef).
		targetRef := &globalref.Reference{
			ContainerAddr: targetResource.Addr.Module,
			LocalRef: &addrs.Reference{
				Subject:   targetResource.Addr.Resource,
				Remaining: hcl.Traversal{hcl.TraverseAttr{Name: targetAttr}},
			},
		}

		switch c.connectorRefOutcome(ctx, bodyResource.Addr.Module, bodyExpr, targetRef) {
		case matchFalse:
			return matchFalse
		case matchUnknown:
			return matchUnknown
		}
		// matchTrue: this pair holds; continue to the next pair (AND semantics).
	}

	return matchTrue
}

// connectorRefOutcome decides, for a single connector attribute pair, whether
// the body expression genuinely references the target resource attribute. It
// reduces the connector expression to the set of resource-attribute references
// it can yield (handling plain traversals, index/expansion steps, splats,
// conditionals, and allowlisted reference-preserving functions), resolves each
// through the reference graph, and applies the decidability rule:
//   - a single homogeneous resource reference, no constant branch -> compare to
//     the target (matchTrue / matchFalse);
//   - only constant branches -> matchFalse (a hardcoded value is not a
//     reference);
//   - two or more distinct references, a reference mixed with a constant, or an
//     unresolved/opaque expression -> matchUnknown (undecidable, defer).
func (c *PolicyCallbackManager) connectorRefOutcome(ctx EvalContext, bodyModule addrs.ModuleInstance, bodyExpr hclsyntax.SimpleAttribute, targetRef *globalref.Reference) matchOutcome {
	if bodyExpr.IsLiteral() {
		return matchFalse
	}

	var traversals []hcl.Traversal
	hasNonRef := false
	if bodyExpr.IsTraversal() {
		traversals = []hcl.Traversal{bodyExpr.Traversal}
	} else {
		var ok bool
		traversals, hasNonRef, ok = simplerefs.DecomposeTraversals(bodyExpr.Expr)
		if !ok {
			return matchUnknown
		}
	}

	tree := ctx.ResourceAttrRefGraph()
	var resolved []*globalref.Reference
	for _, tr := range traversals {
		ref, refDiags := globalref.ParseRef(bodyModule, tr)
		if refDiags.HasErrors() {
			log.Printf("[TRACE] global ref parse error: %s", refDiags.Err())
			return matchUnknown
		}
		attrRef, ok := tree.ResolveReference(ref)
		if ok {
			resolved = append(resolved, attrRef)
			continue
		}
		// A reference that resolves through the graph to a constant is a
		// definitive non-reference branch (a hardcoded value laundered through a
		// local/output); anything else unresolved is genuinely unknown.
		if tree.ResolvesToLiteral(ref) {
			hasNonRef = true
			continue
		}
		return matchUnknown
	}

	distinct := dedupRefs(resolved)
	if hasNonRef {
		if len(distinct) == 0 {
			return matchFalse
		}
		return matchUnknown
	}
	switch len(distinct) {
	case 0:
		return matchUnknown
	case 1:
		if equalRef(targetRef, distinct[0]) {
			return matchTrue
		}
		return matchFalse
	default:
		// Multiple distinct resources reachable: the value could come from any
		// of them, so we cannot decide whether it is the target.
		return matchUnknown
	}
}

// dedupRefs returns the input references de-duplicated at config-address
// granularity.
func dedupRefs(refs []*globalref.Reference) []*globalref.Reference {
	out := make([]*globalref.Reference, 0, len(refs))
	for _, r := range refs {
		seen := false
		for _, o := range out {
			if equalRef(r, o) {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, r)
		}
	}
	return out
}

// combineMatch ORs the outcomes of the two reference directions into a
// three-valued cty.Bool. A direction that could not be evaluated (matchNA) is
// ignored so it never forces the result to unknown; unknown only wins over a
// definitive false.
func combineMatch(a, b matchOutcome) cty.Value {
	if a == matchTrue || b == matchTrue {
		return cty.True
	}
	sawFalse := false
	for _, o := range [...]matchOutcome{a, b} {
		switch o {
		case matchUnknown:
			return cty.UnknownVal(cty.Bool)
		case matchFalse:
			sawFalse = true
		}
	}
	if sawFalse {
		return cty.False
	}
	// Both directions were not-applicable: we genuinely cannot tell.
	return cty.UnknownVal(cty.Bool)
}

// equalRef reports whether two references point at the same resource attribute
// at **config-address granularity**: the module *config* path plus the resource
// type.name plus the attribute path, with module-call instance keys and resource
// instance keys (and expansion index expressions such as count.index/each.key,
// which are dropped before parsing) normalized away.
//
// This granularity is what makes provenance detection work uniformly across
// module boundaries and expansion (count/for_each on either the resource or an
// enclosing module call): a genuine reference to the subject resource is
// detected regardless of which specific instances are involved. Instance-key
// precision (e.g. requiring acl[0] to reference bucket[0] specifically) is an
// explicit non-goal of this prototype.
func equalRef(ref *globalref.Reference, other *globalref.Reference) bool {
	if ref == nil || other == nil {
		return false
	}
	refAttr, ok := ref.ResourceAttr()
	if !ok {
		return false
	}
	otherAttr, ok := other.ResourceAttr()
	if !ok {
		return false
	}
	if !refAttr.Resource.ConfigResource().Equal(otherAttr.Resource.ConfigResource()) {
		return false
	}
	return refAttr.Attr.Equals(otherAttr.Attr)
}

// getAttributeFromBody looks up an attribute expression inside a parsed simple body
// tree using a block/attribute path.
func getAttributeFromBody(simpleBody *hclsyntax.SimpleBody, path cty.Path, resourceSchema *configschema.Block) (hclsyntax.SimpleAttribute, bool) {
	var attr hclsyntax.SimpleAttribute
	if len(path) == 0 {
		return attr, false
	}

	remaining := path[1:]
	switch step := path[0].(type) {
	case cty.GetAttrStep:
		// terminating condition
		if len(path) == 1 {
			attr, ok := simpleBody.Attributes[step.Name]
			return attr, ok
		}

		blk := resourceSchema.BlockTypes[step.Name]
		// If it is not a block, then it should have already been handled as an attribute
		if blk == nil {
			return attr, false
		}
		// If the block is expected to be a single block, we can just
		// get the first block and treat it as such
		if blk.Nesting == configschema.NestingSingle || blk.Nesting == configschema.NestingGroup {
			if len(simpleBody.Blocks) == 0 {
				return attr, false
			}
			final, ok := getAttributeFromBody(simpleBody.Blocks[0].Body, remaining, &blk.Block)
			return final, ok
		}

		// Now we treat other kinds of repeated blocks
		blocks := make(map[string][]hclsyntax.SimpleBlock)
		// group the blocks by type
		for _, block := range simpleBody.Blocks {
			if _, ok := blocks[block.Type]; !ok {
				blocks[block.Type] = make([]hclsyntax.SimpleBlock, 0, len(simpleBody.Blocks))
			}
			blocks[block.Type] = append(blocks[block.Type], block)
		}

		currentBlock, ok := blocks[step.Name]
		if !ok {
			return attr, false
		}
		// if the block is a repeated block, then the next step
		// has to be an index step.
		if len(remaining) == 0 {
			return attr, false
		}
		indexStep, ok := remaining[0].(cty.IndexStep)
		if !ok {
			return attr, false
		}

		if blk.Nesting == configschema.NestingList {
			idx, _ := indexStep.Key.AsBigFloat().Int64()
			current := currentBlock[idx]
			remaining = remaining[1:]

			final, ok := getAttributeFromBody(current.Body, remaining, &blk.Block)
			if ok {
				return final, true
			}
		} else if blk.Nesting == configschema.NestingMap {
			for _, block := range currentBlock {
				if block.Labels[0] == indexStep.Key.AsString() {
					remaining = remaining[1:]
					final, ok := getAttributeFromBody(block.Body, remaining, &blk.Block)
					return final, ok
				}
			}
		}
	default:
		return attr, false
	}

	return attr, false
}
