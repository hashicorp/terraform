package configschema

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// ForceObjectConformance takes a cty.Value of an object type and attempts to force
// it to conform to the implied type of the given schema.
//
// This function serves as a best effort to upgrade pre-existing state values
// for simple schema changes made in later versions. Providers can additionally
// provide resource-specific upgrade functions that can implement more
// elaborate upgrade procedures, which should be handled prior to a call to
// this function so that the provider may, for example, move a value from one
// attribute name to another before the old attribute is removed entirely by
// this function. Custom upgrade functions are the concern of the Terraform
// Core code that produces diffs and are thus not dealt with in this package.
//
// Any attributes that exist both in the value and the schema will be
// converted to the attribute type given in schema, returning an error if
// conversion is not possible. Conversion is implemented with the standard
// transforms provided by the cty convert package.
//
// Attributes that exist in schema but not in the given value will be populated
// with a null value of the required type.
//
// Attributes that exist in the given value but not in the schema will be
// silently discarded under the assumption that they are no longer required.
//
// Any nested blocks types will cause recursive calls to ForceConformance if
// already present in the value, or will be populated with an empty collection
// of the appropriate nesting type if not present. An error is returned if
// an attribute within the given value does not have the type required by the
// nesting mode for its corresponding block type.
//
// The returned error may be a cty.PathError identifying a sub-value within
// the given object that cannot be conformed.
func ForceObjectConformance(val cty.Value, schema *Block) (cty.Value, error) {
	return forceObjectConformance(val, schema, nil)
}

func forceObjectConformance(val cty.Value, schema *Block, path cty.Path) (cty.Value, error) {
	if val.IsNull() {
		return cty.NullVal(schema.ImpliedType()), nil
	}
	if !val.IsKnown() {
		return cty.UnknownVal(schema.ImpliedType()), nil
	}

	newVals := make(map[string]cty.Value)

	oldTy := val.Type()

	// Create path capacity for at least one more element so we can append
	// our attribute names cheaply below.
	path = append(path, nil)
	path = path[:len(path)-1]

	for name, attrS := range schema.Attributes {
		if !oldTy.HasAttribute(name) {
			newVals[name] = cty.NullVal(attrS.Type)
			continue
		}

		path := append(path, cty.GetAttrStep{
			Name: name,
		})

		oldVal := val.GetAttr(name)
		newVal, err := convert.Convert(oldVal, attrS.Type)
		if err != nil {
			return cty.NilVal, path.NewError(err)
		}

		newVals[name] = newVal
	}

	for name, blockS := range schema.BlockTypes {
		path = append(path, cty.GetAttrStep{
			Name: name,
		})

		switch blockS.Nesting {

		case NestingSingle:
			if !oldTy.HasAttribute(name) {
				newVals[name] = cty.NullVal(blockS.ImpliedType())
				continue
			}

			oldVal := val.GetAttr(name)
			if !oldVal.Type().IsObjectType() {
				return cty.NilVal, path.NewErrorf("must have an object type")
			}
			newVal, err := forceObjectConformance(oldVal, &blockS.Block, path)
			if err != nil {
				// Don't use path.NewError here because forceObjectConformance already did that
				return cty.NilVal, err
			}
			newVals[name] = newVal

		case NestingList:
			if !oldTy.HasAttribute(name) {
				newVals[name] = cty.ListValEmpty(blockS.ImpliedType())
				continue
			}

			oldVal := val.GetAttr(name)
			if !oldVal.Type().IsListType() {
				return cty.NilVal, path.NewErrorf("must have a list type")
			}
			if !oldVal.Type().ElementType().IsObjectType() {
				return cty.NilVal, path.NewErrorf("must be a list of an object type")
			}
			if oldVal.IsNull() {
				newVals[name] = cty.ListValEmpty(blockS.ImpliedType())
				continue
			}
			if !oldVal.IsKnown() {
				newVals[name] = cty.UnknownVal(cty.List(blockS.ImpliedType()))
				continue
			}
			length := oldVal.LengthInt()
			if length == 0 {
				newVals[name] = cty.ListValEmpty(blockS.ImpliedType())
				continue
			}

			newElems := make([]cty.Value, 0, oldVal.LengthInt())
			path = append(path, nil)
			path = path[:len(path)-1]

			for it := oldVal.ElementIterator(); it.Next(); {
				key, oldElem := it.Element()
				path := append(path, cty.IndexStep{
					Key: key,
				})
				newElem, err := forceObjectConformance(oldElem, &blockS.Block, path)
				if err != nil {
					// Don't use path.NewError here because forceObjectConformance already did that
					return cty.NilVal, err
				}
				newElems = append(newElems, newElem)
			}

			newVals[name] = cty.ListVal(newElems)

		case NestingSet:
			if !oldTy.HasAttribute(name) {
				newVals[name] = cty.SetValEmpty(blockS.ImpliedType())
				continue
			}

			oldVal := val.GetAttr(name)
			if !oldVal.Type().IsSetType() {
				return cty.NilVal, path.NewErrorf("must have a set type")
			}
			if !oldVal.Type().ElementType().IsObjectType() {
				return cty.NilVal, path.NewErrorf("must be a set of an object type")
			}
			if oldVal.IsNull() {
				newVals[name] = cty.SetValEmpty(blockS.ImpliedType())
				continue
			}
			if !oldVal.IsKnown() {
				newVals[name] = cty.UnknownVal(cty.Set(blockS.ImpliedType()))
				continue
			}
			length := oldVal.LengthInt()
			if length == 0 {
				newVals[name] = cty.SetValEmpty(blockS.ImpliedType())
				continue
			}

			newElems := make([]cty.Value, 0, oldVal.LengthInt())
			path = append(path, nil)
			path = path[:len(path)-1]

			for it := oldVal.ElementIterator(); it.Next(); {
				key, oldElem := it.Element()
				path := append(path, cty.IndexStep{
					Key: key,
				})
				newElem, err := forceObjectConformance(oldElem, &blockS.Block, path)
				if err != nil {
					// Don't use path.NewError here because forceObjectConformance already did that
					return cty.NilVal, err
				}
				newElems = append(newElems, newElem)
			}

			newVals[name] = cty.SetVal(newElems)

		case NestingMap:
			if !oldTy.HasAttribute(name) {
				newVals[name] = cty.MapValEmpty(blockS.ImpliedType())
				continue
			}

			oldVal := val.GetAttr(name)
			if oldVal.IsNull() {
				newVals[name] = cty.MapValEmpty(blockS.ImpliedType())
				continue
			}
			if !oldVal.IsKnown() {
				newVals[name] = cty.UnknownVal(cty.Map(blockS.ImpliedType()))
				continue
			}

			length := oldVal.LengthInt()
			if length == 0 {
				newVals[name] = cty.MapValEmpty(blockS.ImpliedType())
				continue
			}

			newElems := make(map[string]cty.Value)
			path = append(path, nil)
			path = path[:len(path)-1]

			for it := oldVal.ElementIterator(); it.Next(); {
				key, oldElem := it.Element()
				path := append(path, cty.IndexStep{
					Key: key,
				})
				newElem, err := forceObjectConformance(oldElem, &blockS.Block, path)
				if err != nil {
					// Don't use path.NewError here because forceObjectConformance already did that
					return cty.NilVal, err
				}
				newElems[key.AsString()] = newElem
			}

			newVals[name] = cty.MapVal(newElems)

		default:
			// Should never happen because the above is exhaustive
			return cty.NilVal, path.NewErrorf("block type %q has unsupported nesting mode %s", name, blockS.Nesting)
		}
	}

	return cty.ObjectVal(newVals), nil

}
