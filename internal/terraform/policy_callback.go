// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"iter"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

type PolicyCallbackManager struct {
	WalkOperation walkOperation
	Schema        providers.GetProviderSchemaResponse
	Config        *configs.Config
	Source        ConnectingResource
}

type ConnectingResource struct {
	Addr   addrs.AbsResourceInstance
	Body   hcl.Body
	Schema *configschema.Block
	Value  cty.Value
}

// GetRelatedResources returns the related resources for the given target resource type and connection.
func (cb *PolicyCallbackManager) GetRelatedResources(ctx EvalContext, target string, conn *callback.ConnectedBlock, val cty.Value) (callback.RelatedResource, error) {
	found := make([]callback.RelatedResource, 0)
	partial := false
	var err error

	// Consider an example where the terraform config is:
	// resource "aws_s3_bucket" "example" {
	//   bucket = "my-bucket"
	// }
	// resource "aws_s3_bucket_acl" "example" {
	//   bucket = aws_s3_bucket.example.id
	// }
	// and the related attribute pair is
	// { sourceAttribute: "id", relatedAttribute: "bucket" }
	cb.Config.DeepEach(func(cfg *configs.Config) {
		for _, resource := range cfg.Module.ManagedResources {
			if resource.Type != target {
				continue
			}
			relatedAddr := resource.Addr().InModule(cfg.Path)

			// Skip the resource currently under evaluation, i.e aws_s3_bucket.example
			if relatedAddr.Equal(cb.Source.Addr.ConfigResource()) {
				continue
			}

			// Deferred candidates make the overall answer incomplete.
			if ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{relatedAddr}) {
				partial = true
				continue
			}

			var resourcesSeq iter.Seq2[addrs.AbsResourceInstance, cty.Value]
			if cb.WalkOperation == walkApply {
				state := ctx.State()
				resourceSchema := cb.Schema.SchemaForResourceAddr(relatedAddr.Resource)
				// During apply, read the matching objects from state.
				seq := states.ReadEachConfigResourceInstance(state, relatedAddr, func(inst *states.ResourceInstance) (cty.Value, bool) {
					if inst.Current == nil {
						return cty.NilVal, false
					}
					decoded, err := inst.Current.Decode(resourceSchema)
					if err != nil || decoded == nil {
						return cty.NilVal, false
					}
					return decoded.Value, true
				})

				resourcesSeq = func(yield func(addrs.AbsResourceInstance, cty.Value) bool) {
					for addr, value := range seq {
						yield(addr, value)
					}
				}
			} else {
				// During plan, return the matching planned objects.
				resourcesSeq = func(yield func(addrs.AbsResourceInstance, cty.Value) bool) {
					for change := range plans.ReadInstancesForConfigResource(ctx.Changes(), relatedAddr) {
						yield(change.Addr, change.After)
					}
				}
			}

			// If the current iteration is for aws_s3_bucket_acl.example, we will
			// check for the given related attribute pair to match aws_s3_bucket.example.
			// We do that by checking if the related attribute (e.g. bucket) is a literal value
			// or a simple traversal. If it is a literal value, we check if it matches the source attribute
			// in aws_s3_bucket.example.
			// If it is a traversal, we check if the traversal points to the source attribute.
			for addr, resourceValue := range resourcesSeq {
				resourceSchema := cb.Schema.SchemaForResourceAddr(relatedAddr.Resource)
				related := ConnectingResource{
					Addr:   addr,
					Body:   resource.Config,
					Schema: resourceSchema.Body,
					Value:  resourceValue,
				}
				matched := cb.Match(ctx, &cb.Source, &related, conn)
				if matched.IsWhollyKnown() && matched.True() {
					resourceValue, _ = resourceValue.UnmarkDeep()

					cbCtx := &PolicyCallbackManager{
						WalkOperation: cb.WalkOperation,
						Schema:        cb.Schema,
						Config:        cb.Config,
						Source:        related,
					}
					// If the resource matched, and the connected block has a block itself,
					// we recursively get the related resources
					var relatedRes callback.RelatedResource
					if conn.Nested != nil {
						target = conn.Nested.TargetType
						relatedRes, err = cbCtx.GetRelatedResources(ctx, target, conn.Nested, resourceValue)
						if err != nil {
							return
						}
					} else {
						relatedRes = callback.RelatedResource{Value: resourceValue}
					}

					found = append(found, relatedRes)
				}
				partial = partial || !matched.IsWhollyKnown()
			}
		}
	})

	return callback.RelatedResource{
		Related: found,
		Partial: partial,
		Value:   val,
	}, err
}

func (c *PolicyCallbackManager) Match(ctx EvalContext, source, related *ConnectingResource, conn *callback.ConnectedBlock) cty.Value {
	// we will return unknown if we cannot determine whether the resource matches
	unknown := cty.UnknownVal(cty.Bool)

	currentValue := source.Value
	relatedValue := related.Value

	// if there is no related body. What to do?
	if related.Body == nil {
		return unknown
	}

	// Parse the resource config as a simple body that contains only attributes that are either
	// simple traversals or literal values.
	cfg, ok := related.Body.(*hclsyntax.Body)
	if !ok {
		return unknown
	}
	relatedBody, parseDiags := cfg.AsSimpleBody()
	if parseDiags.HasErrors() {
		return unknown
	}

	for _, pair := range conn.AttributePairs {
		// If the current resource is null or does not have the source attribute,
		// we cannot compare the literal to the current value.
		if !currentValue.Type().IsObjectType() || !currentValue.Type().HasAttribute(pair.SourceAttribute) {
			// TODO: Is this unknown or false?
			return unknown
		}

		// The changeset supercedes config, so we check it first.
		// If we have enough information to verify equality, we can compare the related attribute
		// to the source attribute directly, without re-evaluating the related attribute expression.
		relatedValue, relatedTraversal, foundRelated := lookupValue(relatedValue, pair.RelatedAttribute)
		if foundRelated {
			sourceValue, _, foundSource := lookupValue(currentValue, pair.SourceAttribute)
			if !foundSource {
				return unknown
			}
			equals := relatedValue.Equals(sourceValue)
			if equals.IsKnown() {
				if equals.False() { // we can return early if the values do not match
					return cty.False
				}

				// otherwise, the values match, so we continue to the next pair
				continue
			}
		}

		// get the attribute's expression from the body
		path, _ := traversalToPath(relatedTraversal)
		relatedExpr, found := getAttributeFromBody(relatedBody, path, related.Schema)
		if !found {
			// related attribute or block not found
			return unknown
		}

		// If the expression is a literal, try a direct comparison against
		// source value so we can make an early decision if the values do not match.
		if relatedExpr.IsLiteral() && relatedExpr.Expr != nil {
			litVal, litDiags := relatedExpr.Expr.Value(nil)
			if !litDiags.HasErrors() {
				litVal, _ = litVal.UnmarkDeep()
				sourceValue := currentValue.GetAttr(pair.SourceAttribute)
				sourceValue, _ = sourceValue.UnmarkDeep()
				equals := litVal.Equals(sourceValue)
				if equals.IsKnown() {
					if equals.False() {
						return cty.False
					}
					continue
				}
			}
			return unknown
		}

		// Anything more complex than a plain traversal cannot be compared structurally,
		// so we assume it to be unknown if the related attribute expression is not a plain traversal.
		if relatedExpr.Kind == hclsyntax.AttributeKindOther {
			return unknown
		}

		// Walk the reference tree to resolve the related attribute reference to a
		// resource attribute reference.
		relatedRef, refDiags := globalref.ParseRef(related.Addr.Module, relatedExpr.Traversal)
		if refDiags.HasErrors() {
			log.Printf("[TRACE] global ref parse error: %s", refDiags.Err())
			return unknown
		}
		tree := ctx.ResourceAttrRefTree()
		attrRef, found := tree.ResolveReference(relatedRef)
		if !found {
			return unknown
		}

		// Compare the resolved attribute reference to the source reference, including
		// the module instance where both are resolved.
		sourceRef := &globalref.Reference{
			ContainerAddr: c.Source.Addr.Module,
			LocalRef: &addrs.Reference{
				Subject:   c.Source.Addr.Resource,
				Remaining: hcl.Traversal{hcl.TraverseAttr{Name: pair.SourceAttribute}},
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

// func getConnections(ctx EvalContext, conn *callback.Connection, connectable *Connectable) {

// 	// Parse the resource config as a simple body that contains only attributes that are either
// 	// simple traversals or literal values.
// 	cfg, ok := connectable.body.(*hclsyntax.Body)
// 	if !ok {
// 		continue
// 	}
// 	relatedBody, parseDiags := cfg.AsSimpleBody()
// 	if parseDiags.HasErrors() {
// 		partial = true
// 		continue
// 	}

// 	// If the current iteration is for aws_s3_bucket_acl.example, we will
// 	// check for the given related attribute pair to match aws_s3_bucket.example.
// 	// We do that by checking if the related attribute (e.g. bucket) is a literal value
// 	// or a simple traversal. If it is a literal value, we check if it matches the source attribute
// 	// in aws_s3_bucket.example.
// 	// If it is a traversal, we check if the traversal points to the source attribute.
// 	for addr, resourceValue := range resourcesSeq {
// 		resourceSchema := schema.SchemaForResourceAddr(connectable.addr.Resource)
// 		matched := relatedResourceMatchesPairs(ctx, relatedBody, currentAddr, addr, resourceValue, currentAttrs, conn, resourceSchema.Body)
// 		if matched.IsWhollyKnown() && matched.True() {
// 			resourceValue, _ = resourceValue.UnmarkDeep()
// 			found = append(found, resourceValue)
// 		}
// 		partial = partial || !matched.IsWhollyKnown()
// 	}
// }

func relatedResourceMatchesPairs(evalCtx EvalContext, body *hclsyntax.SimpleBody, current, related addrs.AbsResourceInstance, relatedValue, currentValue cty.Value, conn *callback.ConnectedBlock, resourceSchema *configschema.Block) cty.Value {
	// we will return unknown if we cannot determine whether the resource matches
	unknown := cty.UnknownVal(cty.Bool)

	for _, pair := range conn.AttributePairs {
		// If the current resource is null or does not have the source attribute,
		// we cannot compare the literal to the current value.
		if !currentValue.Type().IsObjectType() || !currentValue.Type().HasAttribute(pair.SourceAttribute) {
			// TODO: Is this unknown or false?
			return unknown
		}

		// The changeset supercedes config, so we check it first.
		// If we have enough information to verify equality, we can compare the related attribute
		// to the source attribute directly, without re-evaluating the related attribute expression.
		relatedValue, relatedTraversal, foundRelated := lookupValue(relatedValue, pair.RelatedAttribute)
		if foundRelated {
			sourceValue, _, foundSource := lookupValue(currentValue, pair.SourceAttribute)
			if !foundSource {
				return unknown
			}
			equals := relatedValue.Equals(sourceValue)
			if equals.IsKnown() {
				if equals.False() { // we can return early if the values do not match
					return cty.False
				}

				// otherwise, the values match, so we continue to the next pair
				continue
			}
		}

		// get the attribute's expression from the body
		path, _ := traversalToPath(relatedTraversal)
		relatedExpr, found := getAttributeFromBody(body, path, resourceSchema)
		if !found {
			// related attribute or block not found
			return unknown
		}

		// If the expression is a literal, try a direct comparison against
		// source value so we can make an early decision if the values do not match.
		if relatedExpr.IsLiteral() && relatedExpr.Expr != nil {
			litVal, litDiags := relatedExpr.Expr.Value(nil)
			if !litDiags.HasErrors() {
				litVal, _ = litVal.UnmarkDeep()
				sourceValue := currentValue.GetAttr(pair.SourceAttribute)
				sourceValue, _ = sourceValue.UnmarkDeep()
				equals := litVal.Equals(sourceValue)
				if equals.IsKnown() {
					if equals.False() {
						return cty.False
					}
					continue
				}
			}
			return unknown
		}

		// Anything more complex than a plain traversal cannot be compared structurally,
		// so we assume it to be unknown if the related attribute expression is not a plain traversal.
		if relatedExpr.Kind == hclsyntax.AttributeKindOther {
			return unknown
		}

		// Walk the reference tree to resolve the related attribute reference to a
		// resource attribute reference.
		relatedRef, refDiags := globalref.ParseRef(related.Module, relatedExpr.Traversal)
		if refDiags.HasErrors() {
			log.Printf("[TRACE] global ref parse error: %s", refDiags.Err())
			return unknown
		}
		tree := evalCtx.ResourceAttrRefTree()
		attrRef, found := tree.ResolveReference(relatedRef)
		if !found {
			return unknown
		}

		// Compare the resolved attribute reference to the source reference, including
		// the module instance where both are resolved.
		sourceRef := &globalref.Reference{
			ContainerAddr: current.Module,
			LocalRef: &addrs.Reference{
				Subject:   current.Resource,
				Remaining: hcl.Traversal{hcl.TraverseAttr{Name: pair.SourceAttribute}},
			},
		}

		if !equalRef(sourceRef, attrRef) {
			srcStr := sourceRef.DebugString()
			resStr := attrRef.DebugString()
			log.Printf("[TRACE] global ref comparison failed: source=%s resolved=%s", srcStr, resStr)
			return unknown
		}
	}

	// We found a match at this level, now descend into the next level.
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

func lookupValue(val cty.Value, attr string) (cty.Value, hcl.Traversal, bool) {
	traversal, diags := hclsyntax.ParseTraversalAbs([]byte(attr), "", hcl.Pos{Line: 1, Column: 1})
	if diags != nil {
		log.Println("[DEBUG] Error parsing traversal: ", diags)
		return val, nil, false
	}
	if val.Type().HasAttribute(traversal.RootName()) {
		path, _ := traversalToPath(traversal)
		val, _ := path.Apply(val)
		val, _ = val.UnmarkDeep()
		return val, traversal, true
	}

	return val, nil, false
}

// getAttributeFromBody looks up an attribute expression inside a parsed restricted body
// tree using a block/attribute path.
//
// A block instance is addressed by its block type followed by each of its
// labels. Once an attribute is selected, any remaining path steps are resolved
// recursively through object and tuple constructor expressions stored in the
// returned RestrictedAttribute.
//
// For example, given:
//
//	resource "aws_vpc" "foo" {
//	  config = {
//	    vpc_id = local.ids["primary"]
//	  }
//	}
//
// the path to the nested "vpc_id" expression is:
//
//	resource.aws_vpc.foo.config.vpc_id
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

		// If it is not a block, then it should have already been handled as an attribute
		blk := resourceSchema.BlockTypes[step.Name]
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

		if blk.Nesting == configschema.NestingList {
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

			step, ok := remaining[0].(cty.IndexStep)
			if !ok {
				return attr, false
			}

			idx, _ := step.Key.AsBigFloat().Int64()
			current := currentBlock[idx]
			remaining = remaining[1:]

			final, ok := getAttributeFromBody(current.Body, remaining, &blk.Block)
			if ok {
				return final, true
			}
		}
	default:
		return attr, false
	}

	return attr, false
}
