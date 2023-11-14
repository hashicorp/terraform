// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

// FillAttribute makes the input value match the specified attribute by adding
// attributes and/or performing conversions to make the input value correct.
//
// It is similar to FillType, except it accepts attributes instead of types.
func FillAttribute(in cty.Value, attribute *configschema.Attribute) (cty.Value, error) {
	return fillAttribute(in, attribute, cty.Path{})
}

func fillAttribute(in cty.Value, attribute *configschema.Attribute, path cty.Path) (cty.Value, error) {
	if attribute.NestedType != nil {

		// Then the in value must be an object.
		if !in.Type().IsObjectType() {
			return cty.NilVal, path.NewErrorf("incompatible types; expected object type, found %s", in.Type().FriendlyName())
		}

		switch attribute.NestedType.Nesting {
		case configschema.NestingSingle, configschema.NestingGroup:
			children := make(map[string]cty.Value)
			for name, attribute := range attribute.NestedType.Attributes {
				if in.Type().HasAttribute(name) {
					child, err := fillAttribute(in.GetAttr(name), attribute, path.GetAttr(name))
					if err != nil {
						return cty.NilVal, err
					}
					children[name] = child
					continue
				}

				children[name] = GenerateValueForAttribute(attribute)
			}
			if len(children) == 0 {
				return cty.EmptyObjectVal, nil
			}
			return cty.ObjectVal(children), nil
		case configschema.NestingSet:
			return cty.SetValEmpty(attribute.ImpliedType().ElementType()), nil
		case configschema.NestingList:
			return cty.ListValEmpty(attribute.ImpliedType().ElementType()), nil
		case configschema.NestingMap:
			return cty.MapValEmpty(attribute.ImpliedType().ElementType()), nil
		default:
			panic(fmt.Errorf("unknown nesting mode: %d", attribute.NestedType.Nesting))
		}
	}

	return fillType(in, attribute.Type, path)
}

// FillType makes the input value match the target type by adding attributes
// directly to it or to any nested objects. Essentially, this is a "safe"
// conversion between two objects.
//
// This function can error if one of the embedded types within value doesn't
// match the type expected by target.
//
// If the supplied value isn't an object (or a map that can be treated as an
// object) then a normal conversion is attempted from value into target.
//
// Superfluous attributes within the supplied value (ie. attributes not
// mentioned by the target type) are dropped without error.
func FillType(in cty.Value, target cty.Type) (cty.Value, error) {
	return fillType(in, target, cty.Path{})
}

func fillType(in cty.Value, target cty.Type, path cty.Path) (cty.Value, error) {
	// If we're targeting an object directly, then the in value must be an
	// object or a map. We'll check for those two cases specifically.
	if in.Type().IsObjectType() && target.IsObjectType() {
		attributes := make(map[string]cty.Value)
		for name, attributeType := range target.AttributeTypes() {
			if in.Type().HasAttribute(name) {
				child, err := fillType(in.GetAttr(name), attributeType, path.IndexString(name))
				if err != nil {
					return cty.NilVal, err
				}
				attributes[name] = child
				continue
			}

			attributes[name] = GenerateValueForType(attributeType)
		}
		if len(attributes) == 0 {
			return cty.EmptyObjectVal, nil
		}
		return cty.ObjectVal(attributes), nil
	}

	// And for map.
	if in.Type().IsMapType() && target.IsObjectType() {
		attributes := make(map[string]cty.Value)
		for name, attributeType := range target.AttributeTypes() {
			index := cty.StringVal(name)
			if in.HasIndex(index).True() {
				child, err := fillType(in.Index(index), attributeType, path.Index(index))
				if err != nil {
					return cty.NilVal, err
				}
				attributes[name] = child
				continue
			}

			attributes[name] = GenerateValueForType(attributeType)
		}
		if len(attributes) == 0 {
			return cty.EmptyObjectVal, nil
		}
		return cty.ObjectVal(attributes), nil
	}

	if target.IsObjectType() {
		// If the target is an object type, and the input wasn't an object or
		// a map, then we have incompatible types.
		return cty.NilVal, path.NewErrorf("incompatible types; expected %s, found %s", target.FriendlyName(), in.Type().FriendlyName())
	}

	// We also do a special check for any types that might contain an object as
	// we'll need to recursively call fill over the nested objects.
	if target.IsCollectionType() && target.ElementType().IsObjectType() {
		switch {
		case target.IsListType():
			var values []cty.Value
			switch {
			case in.Type().IsSetType(), in.Type().IsListType(), in.Type().IsTupleType():
				for iterator := in.ElementIterator(); iterator.Next(); {
					index, value := iterator.Element()
					child, err := fillType(value, target.ElementType(), path.Index(index))
					if err != nil {
						return cty.NilVal, err
					}
					values = append(values, child)
				}
			default:
				return cty.NilVal, path.NewErrorf("incompatible types; expected %s, found %s", target.FriendlyName(), in.Type().FriendlyName())
			}
			if len(values) == 0 {
				return cty.ListValEmpty(target.ElementType()), nil
			}
			return cty.ListVal(values), nil
		case target.IsSetType():
			var values []cty.Value
			switch {
			case in.Type().IsSetType(), in.Type().IsListType(), in.Type().IsTupleType():
				for iterator := in.ElementIterator(); iterator.Next(); {
					index, value := iterator.Element()
					child, err := fillType(value, target.ElementType(), path.Index(index))
					if err != nil {
						return cty.NilVal, err
					}
					values = append(values, child)
				}
			default:
				return cty.NilVal, path.NewErrorf("incompatible types; expected %s, found %s", target.FriendlyName(), in.Type().FriendlyName())
			}
			if len(values) == 0 {
				return cty.SetValEmpty(target.ElementType()), nil
			}
			return cty.SetVal(values), nil
		case target.IsMapType():
			values := make(map[string]cty.Value)
			switch {
			case in.Type().IsMapType():
				for name, value := range in.AsValueMap() {
					child, err := fillType(value, target.ElementType(), path.IndexString(name))
					if err != nil {
						return cty.NilVal, err
					}
					values[name] = child
				}
			case in.Type().IsObjectType():
				for name := range in.Type().AttributeTypes() {
					value := in.GetAttr(name)
					child, err := fillType(value, target.ElementType(), path.IndexString(name))
					if err != nil {
						return cty.NilVal, err
					}
					values[name] = child
				}
			default:
				return cty.NilVal, path.NewErrorf("incompatible types; expected %s, found %s", target.FriendlyName(), in.Type().FriendlyName())
			}
			if len(values) == 0 {
				return cty.MapValEmpty(target.ElementType()), nil
			}
			return cty.MapVal(values), nil
		default:
			panic(fmt.Errorf("unrecognized collection type: %s", target.FriendlyName()))
		}
	}

	if target.IsTupleType() && in.Type().IsTupleType() {
		if target.Length() != in.Type().Length() {
			return cty.NilVal, path.NewErrorf("incompatible types; expected %s with length %d, found %s with length %d", target.FriendlyName(), target.Length(), in.Type().FriendlyName(), in.Type().Length())
		}

		var values []cty.Value
		for ix, value := range in.AsValueSlice() {
			child, err := fillType(value, target.TupleElementType(ix), path.IndexInt(ix))
			if err != nil {
				return cty.NilVal, err
			}
			values = append(values, child)
		}
		if len(values) == 0 {
			return cty.EmptyTupleVal, nil
		}
		return cty.TupleVal(values), nil
	}

	// Otherwise, we don't have any nested object types we need to fill and this
	// isn't an actual object either. So we can just do a simple conversion into
	// the target type.
	value, err := convert.Convert(in, target)
	if err != nil {
		return value, path.NewError(err)
	}
	return value, nil
}
