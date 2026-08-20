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
	Addr   addrs.AbsResourceInstance
	Body   hcl.Body
	Schema *configschema.Block
	Value  cty.Value
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

			// If the resource matched, and the relationship block has a block itself,
			// we recursively get the related resources
			var relatedRes callback.RelatedResource
			if blk.Nested != nil {
				relatedRes, err = cb.GetRelatedResources(ctx, related.Addr, blk.Nested, resourceValue)
				if err != nil {
					continue
				}
			} else {
				relatedRes = callback.RelatedResource{Value: resourceValue}
			}

			found = append(found, relatedRes)
		}
		partial = partial || !matched.IsWhollyKnown()
	}

	return callback.RelatedResource{
		Related: found,
		Partial: partial,
		Value:   val,
	}, err
}

func (c *PolicyCallbackManager) Match(ctx EvalContext, subject, related *PolicyResource, conn *callback.RelationshipBlock) cty.Value {
	// we will return unknown if we cannot determine whether the resource matches
	unknown := cty.UnknownVal(cty.Bool)

	currentValue := subject.Value

	// if there is no related body. What to do?
	if related.Body == nil {
		return unknown
	}

	// First try to match by values
	if !conn.QueryAttributes.IsNull() {
		filterMap := conn.QueryAttributes.AsValueMap()
		matches, _ := resourceMatchesFilter(related.Addr.ConfigResource(), related.Schema, filterMap, related.Value)
		if matches {
			return cty.True
		}
	}

	// Parse the resource config as a simple body that contains only attributes that are either
	// simple traversals or literal values.
	relatedBody, diags := hclsyntax.ParseSimpleBody(related.Body)
	if diags.HasErrors() {
		return unknown
	}

	for _, pair := range conn.AttributePairs {
		// If the current resource is null or does not have the source attribute,
		// we cannot compare the literal to the current value.
		if !currentValue.Type().IsObjectType() || !currentValue.Type().HasAttribute(pair.SubjectAttribute) {
			// TODO: Is this unknown or false?
			return unknown
		}

		relatedTraversal, _ := hclsyntax.ParseTraversalAbs([]byte(pair.RelatedAttribute), "", hcl.InitialPos)
		// get the attribute's expression from the body
		path, _ := traversalToPath(relatedTraversal)
		relatedExpr, found := getAttributeFromBody(relatedBody, path, related.Schema)
		if !found {
			// related attribute or block not found. Then it is not a match.
			return cty.False
		}

		// If the related expression is not a plain traversal, it cannot be compared structurally,
		// so we assume it to be unknown if the related attribute expression is not a plain traversal.
		if !relatedExpr.IsTraversal() {
			return unknown
		}

		// Walk the reference tree to resolve the related attribute reference to a
		// resource attribute reference.
		relatedRef, refDiags := globalref.ParseRef(related.Addr.Module, relatedExpr.Traversal)
		if refDiags.HasErrors() {
			log.Printf("[TRACE] global ref parse error: %s", refDiags.Err())
			return unknown
		}
		tree := ctx.ResourceAttrRefGraph()
		attrRef, found := tree.ResolveReference(relatedRef)
		if !found {
			return unknown
		}

		// Compare the resolved attribute reference to the source reference, including
		// the module instance where both are resolved.
		sourceRef := &globalref.Reference{
			ContainerAddr: subject.Addr.Module,
			LocalRef: &addrs.Reference{
				Subject:   subject.Addr.Resource,
				Remaining: hcl.Traversal{hcl.TraverseAttr{Name: pair.SubjectAttribute}},
			},
		}

		if !equalRef(sourceRef, attrRef) {
			srcStr := sourceRef.DebugString()
			resStr := attrRef.DebugString()
			log.Printf("[TRACE] global ref comparison failed: source=%s resolved=%s", srcStr, resStr)
			return unknown
		}
	}

	return cty.True
}

func equalRef(ref *globalref.Reference, other *globalref.Reference) bool {
	if ref == nil || other == nil {
		return false
	}
	if ref.ContainerAddr == nil || other.ContainerAddr == nil {
		return false
	}
	if !addrs.Equivalent(ref.ContainerAddr, other.ContainerAddr) {
		return false
	}

	localRef1 := ref.LocalRef
	localRef2 := other.LocalRef
	if !addrs.Equivalent(localRef1.Subject, localRef2.Subject) {
		return false
	}
	if len(localRef1.Remaining) != len(localRef2.Remaining) {
		return false
	}
	for i := range localRef1.Remaining {
		ref := localRef1.Remaining[i]
		otherRef := localRef2.Remaining[i]
		refAttr, ok := ref.(hcl.TraverseAttr)
		if !ok {
			return false
		}
		otherRefAttr, ok := otherRef.(hcl.TraverseAttr)
		if !ok {
			return false
		}
		if refAttr.Name != otherRefAttr.Name {
			return false
		}
	}
	return true
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
