package configschema

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// CoerceValue attempts to force the given value to conform to the type
// implied by the receiever, while also applying the same validation and
// transformation rules that would be applied by the decoder specification
// returned by method DecoderSpec.
//
// This is useful in situations where a configuration must be derived from
// an already-decoded value. It is always better to decode directly from
// configuration where possible since then source location information is
// still available to produce diagnostics, but in special situations this
// function allows a compatible result to be obtained even if the
// configuration objects are not available.
//
// If the given value cannot be converted to conform to the receiving schema
// then an error is returned describing one of possibly many problems. This
// error may be a cty.PathError indicating a position within the nested
// data structure where the problem applies.
func (b *Block) CoerceValue(in cty.Value) (cty.Value, error) {
	var path cty.Path
	return b.coerceValue(in, path)
}

func (b *Block) coerceValue(in cty.Value, path cty.Path) (cty.Value, error) {
	switch {
	case in.IsNull():
		return cty.NullVal(b.ImpliedType()), nil
	case !in.IsKnown():
		return cty.UnknownVal(b.ImpliedType()), nil
	}

	ty := in.Type()
	if !ty.IsObjectType() {
		return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("an object is required")
	}

	for name := range ty.AttributeTypes() {
		if _, defined := b.Attributes[name]; defined {
			continue
		}
		if _, defined := b.BlockTypes[name]; defined {
			continue
		}
		return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("unexpected attribute %q", name)
	}

	attrs := make(map[string]cty.Value)

	for name, attrS := range b.Attributes {
		var val cty.Value
		switch {
		case ty.HasAttribute(name):
			val = in.GetAttr(name)
		case attrS.Computed:
			val = cty.UnknownVal(attrS.Type)
		case attrS.Optional:
			val = cty.NullVal(attrS.Type)
		default:
			return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q is required", name)
		}

		val, err := attrS.coerceValue(val, append(path, cty.GetAttrStep{Name: name}))
		if err != nil {
			return cty.UnknownVal(b.ImpliedType()), err
		}

		attrs[name] = val
	}
	for typeName, blockS := range b.BlockTypes {
		switch blockS.Nesting {

		case NestingSingle:
			switch {
			case ty.HasAttribute(typeName):
				var err error
				val := in.GetAttr(typeName)
				attrs[typeName], err = blockS.coerceValue(val, append(path, cty.GetAttrStep{Name: typeName}))
				if err != nil {
					return cty.UnknownVal(b.ImpliedType()), err
				}
			case blockS.MinItems != 1 && blockS.MaxItems != 1:
				attrs[typeName] = cty.NullVal(blockS.ImpliedType())
			default:
				// We use the word "attribute" here because we're talking about
				// the cty sense of that word rather than the HCL sense.
				return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q is required", typeName)
			}

		case NestingList:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.List(b.ImpliedType()))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.List(b.ImpliedType()))
					continue
				}

				if !coll.CanIterateElements() {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q must be a list", typeName)
				}
				l := coll.LengthInt()
				if l < blockS.MinItems {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("insufficient items for attribute %q; must have at least %d", typeName, blockS.MinItems)
				}
				if l > blockS.MaxItems && blockS.MaxItems > 0 {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("too many items for attribute %q; must have at least %d", typeName, blockS.MinItems)
				}
				if l == 0 {
					attrs[typeName] = cty.ListValEmpty(b.ImpliedType())
					continue
				}
				elems := make([]cty.Value, 0, l)
				for it := in.ElementIterator(); it.Next(); {
					var err error
					_, val := it.Element()
					val, err = blockS.coerceValue(val, append(path, cty.GetAttrStep{Name: typeName}))
					if err != nil {
						return cty.UnknownVal(b.ImpliedType()), err
					}
					elems = append(elems, val)
				}
				attrs[typeName] = cty.ListVal(elems)
			case blockS.MinItems == 0:
				attrs[typeName] = cty.ListValEmpty(blockS.ImpliedType())
			default:
				return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q is required", typeName)
			}

		case NestingSet:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.Set(b.ImpliedType()))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.Set(b.ImpliedType()))
					continue
				}

				if !coll.CanIterateElements() {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q must be a set", typeName)
				}
				l := coll.LengthInt()
				if l < blockS.MinItems {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("insufficient items for attribute %q; must have at least %d", typeName, blockS.MinItems)
				}
				if l > blockS.MaxItems && blockS.MaxItems > 0 {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("too many items for attribute %q; must have at least %d", typeName, blockS.MinItems)
				}
				if l == 0 {
					attrs[typeName] = cty.SetValEmpty(b.ImpliedType())
					continue
				}
				elems := make([]cty.Value, 0, l)
				for it := in.ElementIterator(); it.Next(); {
					var err error
					_, val := it.Element()
					val, err = blockS.coerceValue(val, append(path, cty.GetAttrStep{Name: typeName}))
					if err != nil {
						return cty.UnknownVal(b.ImpliedType()), err
					}
					elems = append(elems, val)
				}
				attrs[typeName] = cty.SetVal(elems)
			case blockS.MinItems == 0:
				attrs[typeName] = cty.SetValEmpty(blockS.ImpliedType())
			default:
				return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q is required", typeName)
			}

		case NestingMap:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.Map(b.ImpliedType()))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.Map(b.ImpliedType()))
					continue
				}

				if !coll.CanIterateElements() {
					return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q must be a map", typeName)
				}
				l := coll.LengthInt()
				if l == 0 {
					attrs[typeName] = cty.MapValEmpty(b.ImpliedType())
					continue
				}
				elems := make(map[string]cty.Value)
				for it := in.ElementIterator(); it.Next(); {
					var err error
					key, val := it.Element()
					if key.Type() != cty.String || key.IsNull() || !key.IsKnown() {
						return cty.UnknownVal(b.ImpliedType()), path.NewErrorf("attribute %q must be a map", typeName)
					}
					val, err = blockS.coerceValue(val, append(path, cty.GetAttrStep{Name: typeName}))
					if err != nil {
						return cty.UnknownVal(b.ImpliedType()), err
					}
					elems[key.AsString()] = val
				}
				attrs[typeName] = cty.MapVal(elems)
			default:
				attrs[typeName] = cty.MapValEmpty(blockS.ImpliedType())
			}

		default:
			// should never happen because above is exhaustive
			panic(fmt.Errorf("unsupported nesting mode %#v", blockS.Nesting))
		}
	}

	return cty.ObjectVal(attrs), nil
}

func (a *Attribute) coerceValue(in cty.Value, path cty.Path) (cty.Value, error) {
	val, err := convert.Convert(in, a.Type)
	if err != nil {
		return cty.UnknownVal(a.Type), path.NewError(err)
	}
	return val, nil
}
