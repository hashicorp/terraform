package luacty

import (
	lua "github.com/yuin/gopher-lua"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// ToCtyValue attempts to convert the given Lua value to a cty Value of the
// given type.
//
// If the given type is cty.DynamicPseudoType then this method will select
// a cty type automatically based on the Lua value type, which is an obvious
// mapping for most types but note that Lua tables are always converted to
// object types unless specifically typed other wise.
//
// If the requested conversion is not possible -- because the given Lua value
// is not of a suitable type for the target type -- the result is cty.DynamicVal
// and an error is returned.
//
// Not all Lua types have corresponding cty types; those that don't will
// produce an error regardless of the target type.
//
// Error messages are written with a Lua developer as the audience, and so
// will not include Go-specific implementation details. Where possible, the
// result is a cty.PathError describing the location of the error within
// the given data structure.
func (c *Converter) ToCtyValue(val lua.LValue, ty cty.Type) (cty.Value, error) {
	// 'path' starts off as empty but will grow for each level of recursive
	// call we make, so by the time toCtyValue returns it is likely to have
	// unused capacity on the end of it, depending on how deeply-recursive
	// the given Type is.
	path := make(cty.Path, 0)
	return c.toCtyValue(val, ty, path)
}

func (c *Converter) toCtyValue(val lua.LValue, ty cty.Type, path cty.Path) (cty.Value, error) {
	if val.Type() == lua.LTNil {
		return cty.NullVal(ty), nil
	}

	if ty == cty.DynamicPseudoType {
		// Choose a type automatically
		var err error
		ty, err = c.impliedCtyType(val, path)
		if err != nil {
			return cty.DynamicVal, err
		}
	}

	// If the value is a userdata produced by this package then we will
	// unwrap it and attempt conversion using the standard cty conversion
	// logic.
	if val.Type() == lua.LTUserData {
		ud := val.(*lua.LUserData)
		if ctyV, isCty := ud.Value.(cty.Value); isCty {
			ret, err := convert.Convert(ctyV, ty)
			if err != nil {
				return cty.DynamicVal, path.NewError(err)
			}
			return ret, nil
		}
	}

	// If we have a native Lua value, our conversion strategy depends on our
	// target type, now that we've picked one.
	switch {
	case ty == cty.Bool:
		nv := lua.LVAsBool(val)
		return cty.BoolVal(nv), nil
	case ty == cty.Number:
		switch val.Type() {
		case lua.LTNumber:
			nv := float64(val.(lua.LNumber))
			return cty.NumberFloatVal(nv), nil
		default:
			dyVal, err := c.toCtyValue(val, cty.DynamicPseudoType, path)
			if err != nil {
				return cty.DynamicVal, err
			}
			numV, err := convert.Convert(dyVal, cty.Number)
			if err != nil {
				return cty.DynamicVal, path.NewError(err)
			}
			return numV, nil
		}
	case ty == cty.String:
		switch val.Type() {
		case lua.LTString:
			nv := string(val.(lua.LString))
			return cty.StringVal(nv), nil
		default:
			if !lua.LVCanConvToString(val) {
				return cty.DynamicVal, path.NewErrorf("a string is required")
			}
			return cty.StringVal(lua.LVAsString(val)), nil
		}
	case ty.IsObjectType():
		return c.toCtyObject(val, ty, path)
	case ty.IsTupleType():
		return c.toCtyTuple(val, ty, path)
	case ty.IsMapType():
		return c.toCtyMap(val, ty, path)
	case ty.IsListType() || ty.IsSetType():
		return c.toCtyListOrSet(val, ty, path)
	default:
		return cty.DynamicVal, path.NewErrorf("%s values are not allowed", val.Type().String())
	}
}

func (c *Converter) toCtyObject(val lua.LValue, ty cty.Type, path cty.Path) (cty.Value, error) {
	if val.Type() != lua.LTTable {
		return cty.DynamicVal, path.NewErrorf("a table is required")
	}

	attrs := map[string]cty.Value{}
	table := val.(*lua.LTable)

	// Make sure we have capacity in our path array for our key step
	path = append(path, cty.PathStep(nil))

	atys := ty.AttributeTypes()
	for name, aty := range atys {
		path[len(path)-1] = cty.GetAttrStep{
			Name: name,
		}
		avL := table.RawGet(lua.LString(name))
		av, err := c.toCtyValue(avL, aty, path)
		if err != nil {
			return cty.DynamicVal, err
		}
		attrs[name] = av
	}

	var err error
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if err != nil {
			return
		}
		if key.Type() != lua.LTString {
			err = path.NewErrorf("unexpected key %q", key.String())
			return
		}
		if _, expected := atys[string(key.(lua.LString))]; !expected {
			err = path.NewErrorf("unexpected key %q", key.String())
			return
		}
	})
	if err != nil {
		return cty.DynamicVal, err
	}

	return cty.ObjectVal(attrs), nil
}

func (c *Converter) toCtyTuple(val lua.LValue, ty cty.Type, path cty.Path) (cty.Value, error) {
	if val.Type() != lua.LTTable {
		return cty.DynamicVal, path.NewErrorf("a table is required")
	}

	etys := ty.TupleElementTypes()
	elems := make([]cty.Value, len(etys))
	table := val.(*lua.LTable)

	// Make sure we have capacity in our path array for our index step
	path = append(path, cty.PathStep(nil))

	for i, ety := range etys {
		path[len(path)-1] = cty.IndexStep{
			Key: cty.NumberIntVal(int64(i)),
		}
		evL := table.RawGet(lua.LNumber(float64(i + 1))) // lua tables are 1-indexed
		ev, err := c.toCtyValue(evL, ety, path)
		if err != nil {
			return cty.DynamicVal, err
		}
		elems[i] = ev
	}

	var err error
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if err != nil {
			return
		}
		if key.Type() != lua.LTNumber {
			err = path.NewErrorf("unexpected key %q", key.String())
			return
		}
		i := float64(key.(lua.LNumber))
		if i != float64(int(i)) {
			err = path.NewErrorf("unexpected key %q", key.String())
			return
		}
		if int(i) < 1 || int(i) > len(etys) {
			err = path.NewErrorf("index out of range %d", int(i))
			return
		}
	})
	if err != nil {
		return cty.DynamicVal, err
	}

	return cty.TupleVal(elems), nil
}

func (c *Converter) toCtyMap(val lua.LValue, ty cty.Type, path cty.Path) (cty.Value, error) {
	if val.Type() != lua.LTTable {
		return cty.DynamicVal, path.NewErrorf("a table is required")
	}

	ety := ty.ElementType()
	elems := make(map[string]cty.Value)
	table := val.(*lua.LTable)

	// Make sure we have capacity in our path array for our index step
	path = append(path, cty.PathStep(nil))
	path = path[:len(path)-1]

	var err error
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if err != nil {
			return
		}
		keyV, keyErr := c.toCtyValue(key, cty.String, path)
		if keyErr != nil {
			err = path.NewErrorf("invalid key %s: %s", key.String(), keyErr)
			return
		}
		path = append(path, cty.IndexStep{
			Key: keyV,
		})

		valueV, valueErr := c.toCtyValue(value, ety, path)
		if valueErr != nil {
			err = path.NewError(valueErr)
			return
		}

		elems[keyV.AsString()] = valueV
	})
	if err != nil {
		return cty.DynamicVal, err
	}

	// If our element type is DynamicPseudoType then the caller wants us to
	// choose a single element type to unify all of the values.
	if ety == cty.DynamicPseudoType {
		names := make([]string, len(elems))
		etys := make([]cty.Type, len(elems))
		i := 0
		for k, v := range elems {
			names[i] = k
			etys[i] = v.Type()
			i++
		}
		uTy, convs := convert.Unify(etys)
		if uTy == cty.NilType {
			return cty.DynamicVal, path.NewErrorf("all values must be of the same type")
		}
		ety = uTy
		for i, conv := range convs {
			if conv == nil {
				continue
			}

			path := append(path, cty.IndexStep{
				Key: cty.StringVal(names[i]),
			})

			elems[names[i]], err = conv(elems[names[i]])
			if err != nil {
				return cty.DynamicVal, path.NewError(err)
			}
		}
	}

	if len(elems) == 0 {
		return cty.MapValEmpty(ety), nil
	}

	return cty.MapVal(elems), nil
}

func (c *Converter) toCtyListOrSet(val lua.LValue, ty cty.Type, path cty.Path) (cty.Value, error) {
	if val.Type() != lua.LTTable {
		return cty.DynamicVal, path.NewErrorf("a table is required")
	}

	table := val.(*lua.LTable)
	l := table.Len()

	ety := ty.ElementType()
	elems := make([]cty.Value, l)

	for i := 0; i < l; i++ {
		path := append(path, cty.IndexStep{
			Key: cty.NumberIntVal(int64(i)),
		})

		value := table.RawGetInt(i + 1)
		valueV, valueErr := c.toCtyValue(value, ety, path)
		if valueErr != nil {
			return cty.DynamicVal, path.NewError(valueErr)
		}

		elems[i] = valueV
	}

	var err error
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if err != nil {
			return
		}
		if key.Type() != lua.LTNumber {
			err = path.NewErrorf("unexpected key %q", key.String())
			return
		}
		i := float64(key.(lua.LNumber))
		if i != float64(int(i)) {
			err = path.NewErrorf("unexpected key %q", key.String())
			return
		}
		if int(i) < 1 || int(i) > l {
			err = path.NewErrorf("index out of range %d", int(i))
			return
		}
	})
	if err != nil {
		return cty.DynamicVal, err
	}

	// If our element type is DynamicPseudoType then the caller wants us to
	// choose a single element type to unify all of the values.
	if ety == cty.DynamicPseudoType {
		etys := make([]cty.Type, len(elems))
		for i, v := range elems {
			etys[i] = v.Type()
		}
		uTy, convs := convert.Unify(etys)
		if uTy == cty.NilType {
			return cty.DynamicVal, path.NewErrorf("all values must be of the same type")
		}
		for i, conv := range convs {
			if conv == nil {
				continue
			}

			path := append(path, cty.IndexStep{
				Key: cty.NumberIntVal(int64(i)),
			})

			elems[i], err = conv(elems[i])
			if err != nil {
				return cty.DynamicVal, path.NewError(err)
			}
		}
	}

	if len(elems) == 0 {
		if ty.IsSetType() {
			return cty.SetValEmpty(ety), nil
		} else {
			return cty.ListValEmpty(ety), nil
		}
	}

	if ty.IsSetType() {
		return cty.SetVal(elems), nil
	} else {
		return cty.ListVal(elems), nil
	}
}

// ImpliedCtyType attempts to produce a cty Type that is suitable to recieve
// the given Lua value, or returns an error if no mapping is possible.
//
// Error messages are written with a Lua developer as the audience, and so
// will not include Go-specific implementation details. Where possible, the
// result is a cty.PathError describing the location of the error within
// the given data structure.
func (c *Converter) ImpliedCtyType(val lua.LValue) (cty.Type, error) {
	path := make(cty.Path, 0)
	return c.impliedCtyType(val, path)
}

func (c *Converter) impliedCtyType(val lua.LValue, path cty.Path) (cty.Type, error) {
	switch val.Type() {

	case lua.LTNil:
		return cty.DynamicPseudoType, nil

	case lua.LTBool:
		return cty.Bool, nil

	case lua.LTNumber:
		return cty.Number, nil

	case lua.LTString:
		return cty.String, nil

	case lua.LTUserData:
		ud := val.(*lua.LUserData)
		if ctyV, isCty := ud.Value.(cty.Value); isCty {
			return ctyV.Type(), nil
		}

		// Other userdata types (presumably created by other packages) are not allowed
		return cty.DynamicPseudoType, path.NewErrorf("userdata values are not allowed")

	case lua.LTTable:
		table := val.(*lua.LTable)
		var err error

		// Make sure we have capacity in our path array for our key step
		path = append(path, cty.PathStep(nil))
		path = path[:len(path)-1]

		atys := make(map[string]cty.Type)

		table.ForEach(func(key lua.LValue, val lua.LValue) {
			if err != nil {
				return
			}
			keyCty, keyErr := c.ToCtyValue(key, cty.String)
			if keyErr != nil {
				err = path.NewErrorf("all table keys must be strings")
				return
			}
			attrName := keyCty.AsString()
			keyPath := append(path, cty.GetAttrStep{
				Name: attrName,
			})
			aty, valErr := c.impliedCtyType(val, keyPath)
			if valErr != nil {
				err = valErr
				return
			}
			atys[attrName] = aty
		})
		if err != nil {
			return cty.DynamicPseudoType, err
		}

		return cty.Object(atys), nil

	default:
		return cty.DynamicPseudoType, path.NewErrorf("%s values are not allowed", val.Type().String())

	}
}
