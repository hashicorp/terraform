// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// CoerceValue attempts to force the given value to conform to the type
// implied by the receiever.
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
	convType := b.specType()
	impliedType := convType.WithoutOptionalAttributesDeep()

	switch {
	case in.IsNull():
		return cty.NullVal(impliedType), nil
	case !in.IsKnown():
		return cty.UnknownVal(impliedType), nil
	}

	ty := in.Type()
	if !ty.IsObjectType() {
		return cty.UnknownVal(impliedType), path.NewErrorf("an object is required")
	}

	for name := range ty.AttributeTypes() {
		if _, defined := b.Attributes[name]; defined {
			continue
		}
		if _, defined := b.BlockTypes[name]; defined {
			continue
		}
		return cty.UnknownVal(impliedType), path.NewErrorf("unexpected attribute %q", name)
	}

	attrs := make(map[string]cty.Value)

	for name, attrS := range b.Attributes {
		attrType := impliedType.AttributeType(name)
		attrConvType := convType.AttributeType(name)

		var val cty.Value
		switch {
		case ty.HasAttribute(name):
			val = in.GetAttr(name)
		case attrS.Computed || attrS.Optional:
			val = cty.NullVal(attrType)
		case attrS.Required:
			return cty.UnknownVal(impliedType), path.NewErrorf("attribute %q is required", name)
		default:
			return cty.UnknownVal(impliedType), path.NewErrorf("attribute %q has none of required, optional, or computed set", name)
		}

		val, err := convert.Convert(val, attrConvType)
		if err != nil {
			return cty.UnknownVal(impliedType), append(path, cty.GetAttrStep{Name: name}).NewError(err)
		}
		attrs[name] = val
	}

	for typeName, blockS := range b.BlockTypes {
		switch blockS.Nesting {

		case NestingSingle, NestingGroup:
			switch {
			case ty.HasAttribute(typeName):
				var err error
				val := in.GetAttr(typeName)
				attrs[typeName], err = blockS.coerceValue(val, append(path, cty.GetAttrStep{Name: typeName}))
				if err != nil {
					return cty.UnknownVal(impliedType), err
				}
			default:
				attrs[typeName] = blockS.EmptyValue()
			}

		case NestingList:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.List(blockS.ImpliedType()))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.List(blockS.ImpliedType()))
					continue
				}

				coll, marks := coll.Unmark()

				if !coll.CanIterateElements() {
					return cty.UnknownVal(impliedType), path.NewErrorf("must be a list")
				}
				l := coll.LengthInt()

				if l == 0 {
					attrs[typeName] = cty.ListValEmpty(blockS.ImpliedType()).WithMarks(marks)
					continue
				}
				elems := make([]cty.Value, 0, l)
				{
					path = append(path, cty.GetAttrStep{Name: typeName})
					for it := coll.ElementIterator(); it.Next(); {
						var err error
						idx, val := it.Element()
						val, err = blockS.coerceValue(val, append(path, cty.IndexStep{Key: idx}))
						if err != nil {
							return cty.UnknownVal(impliedType), err
						}
						elems = append(elems, val)
					}
				}
				attrs[typeName] = cty.ListVal(elems).WithMarks(marks)
			default:
				attrs[typeName] = cty.ListValEmpty(blockS.ImpliedType())
			}

		case NestingSet:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.Set(blockS.ImpliedType()))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.Set(blockS.ImpliedType()))
					continue
				}
				coll, marks := coll.Unmark()

				if !coll.CanIterateElements() {
					return cty.UnknownVal(impliedType), path.NewErrorf("cannot iterate over %#v", coll)
				}
				l := coll.LengthInt()

				if l == 0 {
					attrs[typeName] = cty.SetValEmpty(blockS.ImpliedType()).WithMarks(marks)
					continue
				}
				elems := make([]cty.Value, 0, l)
				{
					path = append(path, cty.GetAttrStep{Name: typeName})
					for it := coll.ElementIterator(); it.Next(); {
						var err error
						idx, val := it.Element()
						val, err = blockS.coerceValue(val, append(path, cty.IndexStep{Key: idx}))
						if err != nil {
							return cty.UnknownVal(impliedType), err
						}
						elems = append(elems, val)
					}
				}
				attrs[typeName] = cty.SetVal(elems).WithMarks(marks)
			default:
				attrs[typeName] = cty.SetValEmpty(blockS.ImpliedType())
			}

		case NestingMap:
			switch {
			case ty.HasAttribute(typeName):
				coll := in.GetAttr(typeName)

				switch {
				case coll.IsNull():
					attrs[typeName] = cty.NullVal(cty.Map(blockS.ImpliedType()))
					continue
				case !coll.IsKnown():
					attrs[typeName] = cty.UnknownVal(cty.Map(blockS.ImpliedType()))
					continue
				}
				coll, marks := coll.Unmark()

				if !coll.CanIterateElements() {
					return cty.UnknownVal(impliedType), path.NewErrorf("must be a map")
				}
				l := coll.LengthInt()
				if l == 0 {
					attrs[typeName] = cty.MapValEmpty(blockS.ImpliedType()).WithMarks(marks)
					continue
				}
				elems := make(map[string]cty.Value)
				{
					path = append(path, cty.GetAttrStep{Name: typeName})
					for it := coll.ElementIterator(); it.Next(); {
						var err error
						key, val := it.Element()
						if key.Type() != cty.String || key.IsNull() || !key.IsKnown() {
							return cty.UnknownVal(impliedType), path.NewErrorf("must be a map")
						}
						val, err = blockS.coerceValue(val, append(path, cty.IndexStep{Key: key}))
						if err != nil {
							return cty.UnknownVal(impliedType), err
						}
						elems[key.AsString()] = val
					}
				}

				// If the attribute values here contain any DynamicPseudoTypes,
				// the concrete type must be an object.
				useObject := false
				if coll.Type().IsObjectType() || blockS.ImpliedType().HasDynamicTypes() {
					useObject = true
				}

				if useObject {
					attrs[typeName] = cty.ObjectVal(elems).WithMarks(marks)
				} else {
					attrs[typeName] = cty.MapVal(elems).WithMarks(marks)
				}
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
