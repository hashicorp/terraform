package globalref

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// walkBlock walks through the block following the given traversal. If it finds
// any invalid steps in the traversal, the walk is halted and the traversal
// until that point is returned. This effectively validates and corrects the
// given traversal.
func walkBlock(block *configschema.Block, traversal hcl.Traversal) hcl.Traversal {
	if len(traversal) == 0 {
		// then we've reached the end of the traversal, so we won't keep trying
		// to walk through the block
		return nil
	}

	// within a block the next step should always be a string attribute, but we
	// want to be lenient.

	current, ok := coerceToAttribute(traversal[0])
	if !ok {
		return nil
	}

	if attr, ok := current.(hcl.TraverseAttr); ok {
		// this is the valid case, we're expecting an attribute that's going to
		// point to an attribute or a block

		if attr, ok := block.Attributes[attr.Name]; ok {
			return append(hcl.Traversal{current}, walkAttribute(attr, traversal[1:])...)
		}

		if block, ok := block.BlockTypes[attr.Name]; ok {
			return append(hcl.Traversal{current}, walkBlockType(block, traversal[1:])...)
		}
	}

	// if nothing was triggered, this was an invalid reference so we'll cut the
	// traversal here and return nothing.

	return nil
}

// walkBlock walks through the block following the given traversal. If it finds
// any invalid steps in the traversal, the walk is halted and the traversal
// until that point is returned. This effectively validates and corrects the
// given traversal.
func walkBlockType(block *configschema.NestedBlock, traversal hcl.Traversal) hcl.Traversal {
	if len(traversal) == 0 {
		return nil
	}

	switch block.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		// we don't expect an intermediary step for single or group nested
		// blocks, so we can just keep going.
		return walkBlock(&block.Block, traversal)
	case configschema.NestingList:
		// this should be an index type, but we'll tolerate an attribute as long
		// as it's a number
		current, ok := coerceToIntegerIndex(traversal[0])
		if !ok {
			return nil
		}
		return append(hcl.Traversal{current}, walkBlock(&block.Block, traversal[1:])...)
	case configschema.NestingMap:
		// this should be an index type, but we'll tolerate an attribute
		current, ok := coerceToStringIndex(traversal[0])
		if !ok {
			return nil
		}
		return append(hcl.Traversal{current}, walkBlock(&block.Block, traversal[1:])...)
	case configschema.NestingSet:
		return nil // can't reference into a set
	}

	// above switch should have been exhaustive
	return nil
}

// walkAttribute walks through the attribute following the given traversal. If
// it finds any invalid steps in the traversal, the walk is haled and the
// traversal until that point is returned. This effectively validates and
// corrects the given traversal.
func walkAttribute(attr *configschema.Attribute, traversal hcl.Traversal) hcl.Traversal {
	if len(traversal) == 0 {
		return nil
	}

	if attr.NestedType != nil {
		return walkNestedAttribute(attr.NestedType, traversal)
	}

	return walkType(attr.Type, traversal)
}

func walkNestedAttributes(attrs map[string]*configschema.Attribute, traversal hcl.Traversal) hcl.Traversal {
	if len(traversal) == 0 {
		return nil
	}

	current, ok := coerceToAttribute(traversal[0])
	if !ok {
		return nil
	}

	if attr, ok := attrs[current.(hcl.TraverseAttr).Name]; ok {
		return append(hcl.Traversal{current}, walkAttribute(attr, traversal[1:])...)
	}

	return nil
}

func walkNestedAttribute(attr *configschema.Object, traversal hcl.Traversal) hcl.Traversal {
	if len(traversal) == 0 {
		return nil
	}

	switch attr.Nesting {
	case configschema.NestingSingle, configschema.NestingGroup:
		return walkNestedAttributes(attr.Attributes, traversal)
	case configschema.NestingList:
		// this should be an index type, but we'll tolerate an attribute as long
		// as it's a number
		current, ok := coerceToIntegerIndex(traversal[0])
		if !ok {
			return nil
		}
		return append(hcl.Traversal{current}, walkNestedAttributes(attr.Attributes, traversal[1:])...)
	case configschema.NestingMap:
		// this should be an index type, but we'll tolerate an attribute
		current, ok := coerceToStringIndex(traversal[0])
		if !ok {
			return nil
		}
		return append(hcl.Traversal{current}, walkNestedAttributes(attr.Attributes, traversal[1:])...)
	case configschema.NestingSet:
		return nil // can't reference into a set
	}

	// above switch should have been exhaustive
	return nil
}

func walkType(t cty.Type, traversal hcl.Traversal) hcl.Traversal {
	if len(traversal) == 0 {
		return nil
	}

	switch {
	case t.IsPrimitiveType():
		return nil // can't traverse into primitives
	case t.IsListType():
		current, ok := coerceToIntegerIndex(traversal[0])
		if !ok {
			return nil
		}
		return append(hcl.Traversal{current}, walkType(t.ElementType(), traversal[1:])...)
	case t.IsMapType():
		current, ok := coerceToStringIndex(traversal[0])
		if !ok {
			return nil
		}
		return append(hcl.Traversal{current}, walkType(t.ElementType(), traversal[1:])...)
	case t.IsSetType():
		return nil // can't traverse into sets
	case t.IsObjectType():
		current, ok := coerceToAttribute(traversal[0])
		if !ok {
			return nil
		}

		key := current.(hcl.TraverseAttr).Name
		if !t.HasAttribute(key) {
			return nil
		}
		return append(hcl.Traversal{current}, walkType(t.AttributeType(key), traversal[1:])...)
	case t.IsTupleType():
		current, ok := coerceToIntegerIndex(traversal[0])
		if !ok {
			return nil
		}

		key := current.(hcl.TraverseIndex).Key
		if !key.IsKnown() {
			return nil // we can't keep traversing if the index is unknown
		}

		bf := key.AsBigFloat()
		if !bf.IsInt() {
			return nil
		}

		ix, _ := bf.Int64()
		types := t.TupleElementTypes()
		if ix < 0 || ix >= int64(len(types)) {
			return nil
		}

		return append(hcl.Traversal{current}, walkType(types[int(ix)], traversal[1:])...)
	}

	return nil // the above should have been exhaustive
}

// coerceToAttribute converts the provided step into a hcl.TraverseAttr if
// possible.
func coerceToAttribute(step hcl.Traverser) (hcl.Traverser, bool) {

	switch step := step.(type) {
	case hcl.TraverseIndex:
		if !step.Key.IsKnown() {
			// this is the only failure case here - we can't put unknown values
			// into attributes so we return false
			return step, false
		}

		name, err := convert.Convert(step.Key, cty.String)
		if err != nil {
			// integers should be able to be converted into strings, so this
			// should never happen
			return step, false
		}

		// all else being good, package this up as an attribute and keep
		// going

		return hcl.TraverseAttr{
			Name:     name.AsString(),
			SrcRange: step.SrcRange,
		}, true
	case hcl.TraverseAttr:
		return step, true
	}

	// we'll just give up if we see any other types, but we shouldn't see
	// anything except index and attribute traversals by this point.
	return step, false
}

// coerceToStringIndex converts the provided step into a string-valued
// hcl.TraverseIndex if possible.
func coerceToStringIndex(step hcl.Traverser) (hcl.Traverser, bool) {
	switch step := step.(type) {
	case hcl.TraverseIndex:
		key, err := convert.Convert(step.Key, cty.String)
		if err != nil {
			return step, false
		}

		return hcl.TraverseIndex{
			Key:      key,
			SrcRange: step.SrcRange,
		}, true
	case hcl.TraverseAttr:
		return hcl.TraverseIndex{
			Key:      cty.StringVal(step.Name),
			SrcRange: step.SrcRange,
		}, true
	}

	// we'll just give up if we see any other types, but we shouldn't see
	// anything except index and attribute traversals by this point.
	return step, false
}

// coerceToIntegerIndex converts the provided step into an int-valued
// hcl.TraverseIndex if possible.
func coerceToIntegerIndex(step hcl.Traverser) (hcl.Traverser, bool) {
	switch step := step.(type) {
	case hcl.TraverseIndex:
		key, err := convert.Convert(step.Key, cty.Number)
		if err != nil {
			return step, false
		}

		return hcl.TraverseIndex{
			Key:      key,
			SrcRange: step.SrcRange,
		}, true
	case hcl.TraverseAttr:
		number, err := cty.ParseNumberVal(step.Name)
		if err != nil {
			return step, false
		}

		return hcl.TraverseIndex{
			Key:      number,
			SrcRange: step.SrcRange,
		}, true
	}

	// we'll just give up if we see any other types, but we shouldn't see
	// anything except index and attribute traversals by this point.
	return step, false
}
