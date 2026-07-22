// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"
	"iter"
	"log"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func evaluatePolicies(ctx EvalContext, target addrs.AbsResourceInstance, config *configs.Resource, attrs, priorAttrs cty.Value, meta *proto.PolicyEvaluateResourceRequest_ResourceMetadata, callbacks callback.Functions) policy.EvaluationResponse {
	// We want a per-resource parent span so we can reason about the evaluation of individual
	// resources in the trace
	evalCtx := ctx.StopCtx()
	if phaseSpan := ctx.PolicyGraph().span; phaseSpan != nil {
		evalCtx = trace.ContextWithSpan(evalCtx, phaseSpan)
	}

	result := ctx.PolicyClient().EvaluateResource(evalCtx, policy.EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]{
		Target:     target.Resource.Resource.Type,
		Attrs:      policy.CtyToPolicyValue(attrs),
		PriorAttrs: policy.CtyToPolicyValue(priorAttrs),
		Meta:       meta,
		Callbacks:  callbacks,
	})

	// Do a nil check because orphaned resources do not have a config, so we can't provide source information
	// for such errors.
	if config != nil {
		result = result.WithLocalRange(config.DeclRange.Ptr())
	}

	return result
}

func getResourcesForPolicyCallback(ctx EvalContext, walkOperation walkOperation, provider providers.Interface, schema providers.GetProviderSchemaResponse, config *configs.Config) func(callbackCtx context.Context, target string, attrs cty.Value) ([]cty.Value, bool, error) {
	return func(c context.Context, target string, attrs cty.Value) ([]cty.Value, bool, error) {
		_, span := tracer().Start(c, "policy.callback.getResources", trace.WithAttributes(
			attribute.String("policy.callback.getResources.type", target),
		))
		defer span.End()

		found := make([]cty.Value, 0)
		var filterMap map[string]cty.Value
		if !attrs.IsNull() {
			filterMap = attrs.AsValueMap()
		}
		var isPartialResult bool
		config.DeepEach(func(c *configs.Config) {
			state := ctx.State()
			for _, resource := range c.Module.ManagedResources {
				if resource.Type != target {
					continue
				}
				addr := resource.Addr().InModule(c.Path)
				schema := schema.SchemaForResourceAddr(addr.Resource)

				// Before checking the data to see if there is a match, check if there is a deferral for this address.
				//
				// If there is a deferral, we can't use the data to determine if there is a match so we'll indicate
				// the callback return is a partial result.
				deferred := ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{addr})
				if deferred {
					isPartialResult = true
					continue
				}

				// Now we implement a generator function that yields resource instances
				// from either the state or the config, depending on the walk operation.
				var resourcesSeq iter.Seq[cty.Value]
				if walkOperation == walkApply {
					// Read each config resource instance from the state, decoding it into a cty.Value
					seq := states.ReadEachConfigResourceInstance(state, addr, func(inst *states.ResourceInstance) (cty.Value, bool) {
						if inst.Current == nil {
							return cty.NilVal, false
						}
						rsc, err := inst.Current.Decode(schema)
						if err != nil {
							log.Printf("[ERROR] getresources: failed to decode resource %q: %v", addr, err)
							return cty.NilVal, false
						}
						return rsc.Value, true
					})
					resourcesSeq = func(yield func(cty.Value) bool) {
						for _, value := range seq {
							yield(value)
						}
					}
				} else {
					// Read each config resource change from the plan, returning the corresponding cty.Value
					resourcesSeq = func(yield func(cty.Value) bool) {
						for change := range plans.ReadInstancesForConfigResource(ctx.Changes(), addr) {
							yield(change.After)
						}
					}
				}

				for resource := range resourcesSeq {
					matched, unknown := resourceMatchesFilter(addr, schema.Body, filterMap, resource)
					if matched {
						resource, _ = resource.UnmarkDeep()
						found = append(found, resource)
						continue
					}

					// If the filtered attribute for matching is unknown for this resource instance,
					// we can't determine whether it matches, so we'll mark the whole callback result as incomplete.
					// We still continue to the next resource instance, so that we return all known objects as well.
					isPartialResult = isPartialResult || unknown
				}
			}
		})
		span.SetAttributes(attribute.String("policy.callback.getResources.result_count", fmt.Sprintf("%d", len(found))))
		return found, isPartialResult, nil
	}
}

func getDataSourceForPolicyCallback(ctx EvalContext, provider providers.Interface, schema providers.GetProviderSchemaResponse) func(callbackCtx context.Context, datasource string, attrs cty.Value) (cty.Value, bool, error) {
	return func(c context.Context, target string, attrs cty.Value) (cty.Value, bool, error) {
		_, span := tracer().Start(c, "policy.callback.getDataSource", trace.WithAttributes(
			attribute.String("policy.callback.getDataSource.type", target),
		))
		defer span.End()
		if datasource, ok := schema.DataSources[target]; ok {
			configVal, err := datasource.Body.CoerceValue(attrs)
			if err != nil {
				return cty.NilVal, false, fmt.Errorf("invalid attributes for %q: %w", target, err)
			}

			validateResp := provider.ValidateDataResourceConfig(providers.ValidateDataResourceConfigRequest{
				TypeName: target,
				Config:   configVal,
			})
			if err := validateResp.Diagnostics.Err(); err != nil {
				return cty.NilVal, false, fmt.Errorf("failed to validate data source configuration: %s", err)
			}

			meta := cty.NilVal
			if schema.ProviderMeta.Body != nil {
				meta = cty.NullVal(schema.ProviderMeta.Body.ImpliedType())
			}

			readResp := provider.ReadDataSource(providers.ReadDataSourceRequest{
				TypeName:           target,
				Config:             configVal,
				ClientCapabilities: ctx.ClientCapabilities(),
				ProviderMeta:       meta,
			})
			if err := readResp.Diagnostics.Err(); err != nil {
				return cty.NilVal, false, fmt.Errorf("failed to read data source: %s", err)
			}

			// If the data source indicates deferral (which would be unlikely here), we need to pass that info back to the caller
			deferred := false
			if readResp.Deferred != nil {
				// If we don't support deferrals, but the provider reports a deferral, we should emit an error.
				if !ctx.Deferrals().DeferralAllowed() {
					return cty.NilVal, false, fmt.Errorf("failed to read data source: "+
						"The provider signaled a deferred action for %s, but in this context deferrals are disabled. "+
						"This is a bug in the provider, please file an issue with the provider developers.", target)
				}

				deferred = true
			}

			return readResp.State, deferred, nil
		}
		return cty.NilVal, false, fmt.Errorf("no data source found for %s", target)
	}
}

func relatedResourcesForPolicyCallback(ctx EvalContext, walkOperation walkOperation, schema providers.GetProviderSchemaResponse, config *configs.Config, currentAddr addrs.AbsResourceInstance, currentAttrs cty.Value) func(context.Context, string, []callback.RelatedAttributePair) ([]cty.Value, bool, error) {
	return func(_ context.Context, target string, pairs []callback.RelatedAttributePair) ([]cty.Value, bool, error) {
		found := make([]cty.Value, 0)
		partial := false

		// Consider an example where the terraform config is:
		// resource "aws_s3_bucket" "example" {
		//   bucket = "my-bucket"
		// }
		// resource "aws_s3_bucket_acl" "example" {
		//   bucket = aws_s3_bucket.example.id
		// }
		// and the related attribute pair is
		// { sourceAttribute: "id", relatedAttribute: "bucket" }
		config.DeepEach(func(cfg *configs.Config) {
			for _, resource := range cfg.Module.ManagedResources {
				if resource.Type != target {
					continue
				}
				relatedAddr := resource.Addr().InModule(cfg.Path)

				// Skip the resource currently under evaluation, i.e aws_s3_bucket.example
				if relatedAddr.Equal(currentAddr.ConfigResource()) {
					continue
				}

				// Deferred candidates make the overall answer incomplete.
				if ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{relatedAddr}) {
					partial = true
					continue
				}

				// Parse the resource config as a simple body that contains only attributes that are either
				// simple traversals or literal values.
				cfg, ok := resource.Config.(*hclsyntax.Body)
				if !ok {
					continue
				}
				relatedBody, parseDiags := cfg.AsSimpleBody()
				if parseDiags.HasErrors() {
					partial = true
					continue
				}

				var resourcesSeq iter.Seq2[addrs.AbsResourceInstance, cty.Value]
				if walkOperation == walkApply {
					state := ctx.State()
					resourceSchema := schema.SchemaForResourceAddr(relatedAddr.Resource)
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
					resourceSchema := schema.SchemaForResourceAddr(relatedAddr.Resource)
					matched := relatedResourceMatchesPairs(ctx, relatedBody, currentAddr, addr, resourceValue, currentAttrs, pairs, resourceSchema.Body)
					if matched.IsWhollyKnown() && matched.True() {
						resourceValue, _ = resourceValue.UnmarkDeep()
						found = append(found, resourceValue)
					}
					partial = partial || !matched.IsWhollyKnown()
				}
			}
		})

		return found, partial, nil
	}
}

// resourceMatchesFilter returns whether the given resource matches the given filter attributes and/or if the filter attributes are unknown for the resource.
func resourceMatchesFilter(addr addrs.ConfigResource, schema *configschema.Block, filterAttrs map[string]cty.Value, resource cty.Value) (matches, unknown bool) {
	if resource.IsNull() {
		// if the resource is null, then it doesn't match anything
		return false, false
	}
	if len(filterAttrs) == 0 {
		// if the filter is null, then match everything
		return true, false
	}

	sawUnknown := false

	attrTypes := resource.Type().AttributeTypes()
	for name, attr := range filterAttrs {
		if _, ok := attrTypes[name]; !ok {
			return false, false
		}

		equals := attr.Equals(resource.GetAttr(name))
		if !equals.IsKnown() {
			// If the filtered attribute for matching is unknown for this resource instance, we
			// can't determine whether it matches, so we track that we saw an unknown attribute and continue to check other attributes.
			// This lets false matches take precedence over the unknown result, since we can still determine that the resource does not match.
			sawUnknown = true
			continue
		}

		if equals.False() {
			log.Printf("[DEBUG] attribute %q does not match in resource %q", name, addr.String())
			// an attribute mismatch means we can't match this resource
			return false, false
		}
	}

	// We saw an unknown attribute, and no other attributes mismatched, so we can't determine whether the resource matches.
	if sawUnknown {
		return false, true
	}

	return true, false
}

func relatedResourceMatchesPairs(evalCtx EvalContext, body *hclsyntax.SimpleBody, current, related addrs.AbsResourceInstance, relatedValue, currentValue cty.Value, pairs []callback.RelatedAttributePair, resourceSchema *configschema.Block) cty.Value {
	// we will return unknown if we cannot determine whether the resource matches
	unknown := cty.UnknownVal(cty.Bool)

	for _, pair := range pairs {
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
