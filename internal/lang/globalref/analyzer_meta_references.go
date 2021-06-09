package globalref

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"
)

// MetaReferences inspects the configuration to find the references contained
// within the most specific object that the given address refers to.
//
// This finds only the direct references in that object, not any indirect
// references from those. This is a building block for some other Analyzer
// functions that can walk through multiple levels of reference.
//
// If the given reference refers to something that doesn't exist in the
// configuration we're analyzing then MetaReferences will return no
// meta-references at all, which is indistinguishable from an existing
// object that doesn't refer to anything.
func (a *Analyzer) MetaReferences(ref Reference) []Reference {
	// This function is aiming to encapsulate the fact that a reference
	// is actually quite a complex notion which includes both a specific
	// object the reference is to, where each distinct object type has
	// a very different representation in the configuration, and then
	// also potentially an attribute or block within the definition of that
	// object. Our goal is to make all of these different situations appear
	// mostly the same to the caller, in that all of them can be reduced to
	// a set of references regardless of which expression or expressions we
	// derive those from.

	moduleAddr := ref.ModuleAddr()
	remaining := ref.LocalRef.Remaining

	// Our first task then is to select an appropriate implementation based
	// on which address type the reference refers to.
	switch targetAddr := ref.LocalRef.Subject.(type) {
	case addrs.InputVariable:
		return a.metaReferencesInputVariable(moduleAddr, targetAddr, remaining)
	case addrs.AbsModuleCallOutput:
		return a.metaReferencesOutputValue(moduleAddr, targetAddr, remaining)
	case addrs.ModuleCallInstance:
		return a.metaReferencesModuleCall(moduleAddr, targetAddr, remaining)
	case addrs.ModuleCall:
		// TODO: It isn't really correct to say that a reference to a module
		// call is a reference to its no-key instance. Really what we want to
		// say here is that it's a reference to _all_ instances, or to an
		// instance with an unknown key, but we don't have any representation
		// of that. For the moment it's pretty immaterial since most of our
		// other analysis ignores instance keys anyway, but maybe we'll revisit
		// this latter to distingish these two cases better.
		return a.metaReferencesModuleCall(moduleAddr, targetAddr.Instance(addrs.NoKey), remaining)
	case addrs.CountAttr, addrs.ForEachAttr:
		if resourceAddr, ok := ref.ResourceAddr(); ok {
			return a.metaReferencesCountOrEach(resourceAddr)
		}
		return nil
	case addrs.ResourceInstance:
		return a.metaReferencesResourceInstance(moduleAddr, targetAddr, remaining)
	case addrs.Resource:
		// TODO: It isn't really correct to say that a reference to a resource
		// is a reference to its no-key instance. Really what we want to say
		// here is that it's a reference to _all_ instances, or to an instance
		// with an unknown key, but we don't have any representation of that.
		// For the moment it's pretty immaterial since most of our other
		// analysis ignores instance keys anyway, but maybe we'll revisit this
		// latter to distingish these two cases better.
		return a.metaReferencesResourceInstance(moduleAddr, targetAddr.Instance(addrs.NoKey), remaining)
	default:
		// For anything we don't explicitly support we'll just return no
		// references. This includes the reference types that don't really
		// refer to configuration objects at all, like "path.module",
		// and so which cannot possibly generate any references.
		return nil
	}
}

func (a *Analyzer) metaReferencesInputVariable(calleeAddr addrs.ModuleInstance, addr addrs.InputVariable, remain hcl.Traversal) []Reference {
	if calleeAddr.IsRoot() {
		// A root module variable definition can never refer to anything,
		// because it conceptually exists outside of any module.
		return nil
	}

	callerAddr, callAddr := calleeAddr.Call()

	// We need to find the module call inside the caller module.
	callerCfg := a.ModuleConfig(callerAddr)
	if callerCfg == nil {
		return nil
	}
	call := callerCfg.ModuleCalls[callAddr.Name]
	if call == nil {
		return nil
	}

	// Now we need to look for an attribute matching the variable name inside
	// the module block body.
	body := call.Config
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: addr.Name},
		},
	}
	// We don't check for errors here because we'll make a best effort to
	// analyze whatever partial result HCL is able to extract.
	content, _, _ := body.PartialContent(schema)
	attr := content.Attributes[addr.Name]
	if attr == nil {
		return nil
	}
	refs, _ := lang.ReferencesInExpr(attr.Expr)
	return absoluteRefs(callerAddr, refs)
}

func (a *Analyzer) metaReferencesOutputValue(callerAddr addrs.ModuleInstance, addr addrs.AbsModuleCallOutput, remain hcl.Traversal) []Reference {
	calleeAddr := callerAddr.Child(addr.Call.Call.Name, addr.Call.Key)

	// We need to find the output value declaration inside the callee module.
	calleeCfg := a.ModuleConfig(calleeAddr)
	if calleeCfg == nil {
		return nil
	}

	oc := calleeCfg.Outputs[addr.Name]
	if oc == nil {
		return nil
	}

	// We don't check for errors here because we'll make a best effort to
	// analyze whatever partial result HCL is able to extract.
	refs, _ := lang.ReferencesInExpr(oc.Expr)
	return absoluteRefs(calleeAddr, refs)
}

func (a *Analyzer) metaReferencesModuleCall(callerAddr addrs.ModuleInstance, addr addrs.ModuleCallInstance, remain hcl.Traversal) []Reference {
	calleeAddr := callerAddr.Child(addr.Call.Name, addr.Key)

	// What we're really doing here is just rolling up all of the references
	// from all of this module's output values.
	calleeCfg := a.ModuleConfig(calleeAddr)
	if calleeCfg == nil {
		return nil
	}

	var ret []Reference
	for name := range calleeCfg.Outputs {
		outputAddr := addrs.AbsModuleCallOutput{
			Call: addr,
			Name: name,
		}
		moreRefs := a.metaReferencesOutputValue(callerAddr, outputAddr, nil)
		ret = append(ret, moreRefs...)
	}
	return ret
}

func (a *Analyzer) metaReferencesCountOrEach(resourceAddr addrs.AbsResource) []Reference {
	return a.ReferencesFromResourceRepetition(resourceAddr)
}

func (a *Analyzer) metaReferencesResourceInstance(moduleAddr addrs.ModuleInstance, addr addrs.ResourceInstance, remain hcl.Traversal) []Reference {
	modCfg := a.ModuleConfig(moduleAddr)
	if modCfg == nil {
		return nil
	}

	rc := modCfg.ResourceByAddr(addr.Resource)
	if rc == nil {
		return nil
	}

	// In valid cases we should have the schema for this resource type
	// available. In invalid cases we might be dealing with partial information,
	// and so the schema might be nil so we won't be able to return reference
	// information for this particular situation.
	providerSchema := a.providerSchemas[rc.Provider]
	if providerSchema == nil {
		return nil
	}
	resourceTypeSchema, _ := providerSchema.SchemaForResourceAddr(addr.Resource)
	if resourceTypeSchema == nil {
		return nil
	}

	// When analyzing the resource configuration to look for references, we'll
	// make a best effort to narrow down to only a particular sub-portion of
	// the configuration by following the remaining traversal steps. In the
	// ideal case this will lead us to a specific expression, but as a
	// compromise it might lead us to some nested blocks where at least we
	// can limit our searching only to those.
	bodies := []hcl.Body{rc.Config}
	var exprs []hcl.Expression
	schema := resourceTypeSchema
	var steppingThrough *configschema.NestedBlock
	var steppingThroughType string
	nextStep := func(newBodies []hcl.Body, newExprs []hcl.Expression) {
		// We append exprs but replace bodies because exprs represent extra
		// expressions we collected on the path, such as dynamic block for_each,
		// which can potentially contribute to the final evalcontext, but
		// bodies never contribute any values themselves, and instead just
		// narrow down where we're searching.
		bodies = newBodies
		exprs = append(exprs, newExprs...)
		steppingThrough = nil
		steppingThroughType = ""
		// Caller must also update "schema" if necessary.
	}
	traverseInBlock := func(name string) ([]hcl.Body, []hcl.Expression) {
		if attr := schema.Attributes[name]; attr != nil {
			// When we reach a specific attribute we can't traverse any deeper, because attributes are the leaves of the schema.
			schema = nil
			return traverseAttr(bodies, name)
		} else if blockType := schema.BlockTypes[name]; blockType != nil {
			// We need to take a different action here depending on
			// the nesting mode of the block type. Some require us
			// to traverse in two steps in order to select a specific
			// child block, while others we can just step through
			// directly.
			switch blockType.Nesting {
			case configschema.NestingSingle, configschema.NestingGroup:
				// There should be only zero or one blocks of this
				// type, so we can traverse in only one step.
				schema = &blockType.Block
				return traverseNestedBlockSingle(bodies, name)
			case configschema.NestingMap, configschema.NestingList, configschema.NestingSet:
				steppingThrough = blockType
				return bodies, exprs // Preserve current selections for the second step
			default:
				// The above should be exhaustive, but just in case
				// we add something new in future we'll bail out
				// here and conservatively return everything under
				// the current traversal point.
				schema = nil
				return nil, nil
			}
		}

		// We'll get here if the given name isn't in the schema at all. If so,
		// there's nothing else to be done here.
		schema = nil
		return nil, nil
	}
Steps:
	for _, step := range remain {
		// If we filter out all of our bodies before we finish traversing then
		// we know we won't find anything else, because all of our subsequent
		// traversal steps won't have any bodies to search.
		if len(bodies) == 0 {
			return nil
		}
		// If we no longer have a schema then that suggests we've
		// traversed as deep as what the schema covers (e.g. we reached
		// a specific attribute) and so we'll stop early, assuming that
		// any remaining steps are traversals into an attribute expression
		// result.
		if schema == nil {
			break
		}

		switch step := step.(type) {

		case hcl.TraverseAttr:
			switch {
			case steppingThrough != nil:
				// If we're stepping through a NestingMap block then
				// it's valid to use attribute syntax to select one of
				// the blocks by its label. Other nesting types require
				// TraverseIndex, so can never be valid.
				if steppingThrough.Nesting != configschema.NestingMap {
					nextStep(nil, nil) // bail out
					continue
				}
				nextStep(traverseNestedBlockMap(bodies, steppingThroughType, step.Name))
				schema = &steppingThrough.Block
			default:
				nextStep(traverseInBlock(step.Name))
				if schema == nil {
					// traverseInBlock determined that we've traversed as
					// deep as we can with reference to schema, so we'll
					// stop here and just process whatever's selected.
					break Steps
				}
			}
		case hcl.TraverseIndex:
			switch {
			case steppingThrough != nil:
				switch steppingThrough.Nesting {
				case configschema.NestingMap:
					keyVal, err := convert.Convert(step.Key, cty.String)
					if err != nil { // Invalid traversal, so can't have any refs
						nextStep(nil, nil) // bail out
						continue
					}
					nextStep(traverseNestedBlockMap(bodies, steppingThroughType, keyVal.AsString()))
					schema = &steppingThrough.Block
				case configschema.NestingList:
					idxVal, err := convert.Convert(step.Key, cty.Number)
					if err != nil { // Invalid traversal, so can't have any refs
						nextStep(nil, nil) // bail out
						continue
					}
					var idx int
					err = gocty.FromCtyValue(idxVal, &idx)
					if err != nil { // Invalid traversal, so can't have any refs
						nextStep(nil, nil) // bail out
						continue
					}
					nextStep(traverseNestedBlockList(bodies, steppingThroughType, idx))
					schema = &steppingThrough.Block
				default:
					// Note that NestingSet ends up in here because we don't
					// actually allow traversing into set-backed block types,
					// and so such a reference would be invalid.
					nextStep(nil, nil) // bail out
					continue
				}
			default:
				// When indexing the contents of a block directly we always
				// interpret the key as a string representing an attribute
				// name.
				nameVal, err := convert.Convert(step.Key, cty.String)
				if err != nil { // Invalid traversal, so can't have any refs
					nextStep(nil, nil) // bail out
					continue
				}
				nextStep(traverseInBlock(nameVal.AsString()))
				if schema == nil {
					// traverseInBlock determined that we've traversed as
					// deep as we can with reference to schema, so we'll
					// stop here and just process whatever's selected.
					break Steps
				}
			}
		default:
			// We shouldn't get here, because the above cases are exhaustive
			// for all of the relative traversal types, but we'll be robust in
			// case HCL adds more in future and just pretend the traversal
			// ended a bit early if so.
			break Steps
		}
	}

	if steppingThrough != nil {
		// If we ended in the middle of "stepping through" then we'll conservatively
		// use the bodies of _all_ nested blocks of the type we were stepping
		// through, because the recipient of this value could refer to any
		// of them dynamically.
		var labelNames []string
		if steppingThrough.Nesting == configschema.NestingMap {
			labelNames = []string{"key"}
		}
		blocks := findBlocksInBodies(bodies, steppingThroughType, labelNames)
		for _, block := range blocks {
			bodies, exprs = blockParts(block)
		}
	}

	if len(bodies) == 0 && len(exprs) == 0 {
		return nil
	}

	var refs []*addrs.Reference
	for _, expr := range exprs {
		moreRefs, _ := lang.ReferencesInExpr(expr)
		refs = append(refs, moreRefs...)
	}
	if schema != nil {
		for _, body := range bodies {
			moreRefs, _ := lang.ReferencesInBlock(body, schema)
			refs = append(refs, moreRefs...)
		}
	}
	return absoluteRefs(addr.Absolute(moduleAddr), refs)
}

func traverseAttr(bodies []hcl.Body, name string) ([]hcl.Body, []hcl.Expression) {
	if len(bodies) == 0 {
		return nil, nil
	}
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: name},
		},
	}
	// We can find at most one expression per body, because attribute names
	// are always unique within a body.
	retExprs := make([]hcl.Expression, 0, len(bodies))
	for _, body := range bodies {
		content, _, _ := body.PartialContent(schema)
		if attr := content.Attributes[name]; attr != nil && attr.Expr != nil {
			retExprs = append(retExprs, attr.Expr)
		}
	}
	return nil, retExprs
}

func traverseNestedBlockSingle(bodies []hcl.Body, typeName string) ([]hcl.Body, []hcl.Expression) {
	if len(bodies) == 0 {
		return nil, nil
	}

	blocks := findBlocksInBodies(bodies, typeName, nil)
	var retBodies []hcl.Body
	var retExprs []hcl.Expression
	for _, block := range blocks {
		moreBodies, moreExprs := blockParts(block)
		retBodies = append(retBodies, moreBodies...)
		retExprs = append(retExprs, moreExprs...)
	}
	return retBodies, retExprs
}

func traverseNestedBlockMap(bodies []hcl.Body, typeName string, key string) ([]hcl.Body, []hcl.Expression) {
	if len(bodies) == 0 {
		return nil, nil
	}

	blocks := findBlocksInBodies(bodies, typeName, []string{"key"})
	var retBodies []hcl.Body
	var retExprs []hcl.Expression
	for _, block := range blocks {
		switch block.Type {
		case "dynamic":
			// For dynamic blocks we allow the key to be chosen dynamically
			// and so we'll just conservatively include all dynamic block
			// bodies. However, we need to also look for references in some
			// arguments of the dynamic block itself.
			argExprs, contentBody := dynamicBlockParts(block.Body)
			retExprs = append(retExprs, argExprs...)
			if contentBody != nil {
				retBodies = append(retBodies, contentBody)
			}
		case typeName:
			if len(block.Labels) == 1 && block.Labels[0] == key && block.Body != nil {
				retBodies = append(retBodies, block.Body)
			}
		}
	}
	return retBodies, retExprs
}

func traverseNestedBlockList(bodies []hcl.Body, typeName string, idx int) ([]hcl.Body, []hcl.Expression) {
	if len(bodies) == 0 {
		return nil, nil
	}

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: typeName, LabelNames: nil},
			{Type: "dynamic", LabelNames: []string{"type"}},
		},
	}
	var retBodies []hcl.Body
	var retExprs []hcl.Expression
	for _, body := range bodies {
		content, _, _ := body.PartialContent(schema)
		blocks := content.Blocks

		// A tricky aspect of this scenario is that if there are any "dynamic"
		// blocks then we can't statically predict how many concrete blocks they
		// will generate, and so consequently we can't predict the indices of
		// any statically-defined blocks that might appear after them.
		firstDynamic := -1 // -1 means "no dynamic blocks"
		for i, block := range blocks {
			if block.Type == "dynamic" {
				firstDynamic = i
				break
			}
		}

		switch {
		case firstDynamic >= 0 && idx >= firstDynamic:
			// This is the unfortunate case where the selection could be
			// any of the blocks from firstDynamic onwards, and so we
			// need to conservatively include all of them in our result.
			for _, block := range blocks[firstDynamic:] {
				moreBodies, moreExprs := blockParts(block)
				retBodies = append(retBodies, moreBodies...)
				retExprs = append(retExprs, moreExprs...)
			}
		default:
			// This is the happier case where we can select just a single
			// static block based on idx. Note that this one is guaranteed
			// to never be dynamic but we're using blockParts here just
			// for consistency.
			moreBodies, moreExprs := blockParts(blocks[idx])
			retBodies = append(retBodies, moreBodies...)
			retExprs = append(retExprs, moreExprs...)
		}
	}

	return retBodies, retExprs
}

func findBlocksInBodies(bodies []hcl.Body, typeName string, labelNames []string) []*hcl.Block {
	// We need to look for both static blocks of the given type, and any
	// dynamic blocks whose label gives the expected type name.
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: typeName, LabelNames: labelNames},
			{Type: "dynamic", LabelNames: []string{"type"}},
		},
	}
	var blocks []*hcl.Block
	for _, body := range bodies {
		// We ignore errors here because we'll just make a best effort to analyze
		// whatever partial result HCL returns in that case.
		content, _, _ := body.PartialContent(schema)

		for _, block := range content.Blocks {
			switch block.Type {
			case "dynamic":
				if len(block.Labels) != 1 { // Invalid
					continue
				}
				if block.Labels[0] == typeName {
					blocks = append(blocks, block)
				}
			case typeName:
				blocks = append(blocks, block)
			}
		}
	}

	// NOTE: The caller still needs to check for dynamic vs. static in order
	// to do further processing. The callers above all aim to encapsulate
	// that.
	return blocks
}

func blockParts(block *hcl.Block) ([]hcl.Body, []hcl.Expression) {
	switch block.Type {
	case "dynamic":
		exprs, contentBody := dynamicBlockParts(block.Body)
		var bodies []hcl.Body
		if contentBody != nil {
			bodies = []hcl.Body{contentBody}
		}
		return bodies, exprs
	default:
		if block.Body == nil {
			return nil, nil
		}
		return []hcl.Body{block.Body}, nil
	}
}

func dynamicBlockParts(body hcl.Body) ([]hcl.Expression, hcl.Body) {
	if body == nil {
		return nil, nil
	}

	// This is a subset of the "dynamic" block schema defined by the HCL
	// dynblock extension, covering only the two arguments that are allowed
	// to be arbitrary expressions possibly referring elsewhere.
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "for_each"},
			{Name: "labels"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "content"},
		},
	}
	content, _, _ := body.PartialContent(schema)
	var exprs []hcl.Expression
	if len(content.Attributes) != 0 {
		exprs = make([]hcl.Expression, 0, len(content.Attributes))
	}
	for _, attr := range content.Attributes {
		if attr.Expr != nil {
			exprs = append(exprs, attr.Expr)
		}
	}
	var contentBody hcl.Body
	for _, block := range content.Blocks {
		if block != nil && block.Type == "content" && block.Body != nil {
			contentBody = block.Body
		}
	}
	return exprs, contentBody
}
