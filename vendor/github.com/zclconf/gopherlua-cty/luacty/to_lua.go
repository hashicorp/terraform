package luacty

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// WrapCtyValue takes a cty Value and returns a Lua value (of type UserData)
// that represents the same value, with its metatable configured such that
// many Lua operations will delegate to the cty API.
//
// WrapCtyValue produces a result that stays as close as possible to cty
// semantics when used with other such wrapped values, but the result may
// not integrate well with native Lua values. For example, a wrapped cty.String
// value will not compare equal to any native Lua string.
func (c *Converter) WrapCtyValue(val cty.Value) lua.LValue {
	ret := c.lstate.NewUserData()
	ret.Value = val
	ret.Metatable = c.metatable
	return ret
}

func (c *Converter) ctyMetatable() *lua.LTable {
	L := c.lstate
	table := L.NewTable()

	table.RawSet(lua.LString("__eq"), c.lstate.NewFunction(c.ctyEq))
	table.RawSet(lua.LString("__add"), c.lstate.NewFunction(c.ctyArithmetic(stdlib.Add)))
	table.RawSet(lua.LString("__sub"), c.lstate.NewFunction(c.ctyArithmetic(stdlib.Subtract)))
	table.RawSet(lua.LString("__mul"), c.lstate.NewFunction(c.ctyArithmetic(stdlib.Multiply)))
	table.RawSet(lua.LString("__div"), c.lstate.NewFunction(c.ctyArithmetic(stdlib.Divide)))
	table.RawSet(lua.LString("__mod"), c.lstate.NewFunction(c.ctyArithmetic(stdlib.Modulo)))
	table.RawSet(lua.LString("__unm"), c.lstate.NewFunction(c.ctyNegate))
	table.RawSet(lua.LString("__concat"), c.lstate.NewFunction(c.ctyConcat))
	table.RawSet(lua.LString("__len"), c.lstate.NewFunction(c.ctyLength))
	table.RawSet(lua.LString("__lt"), c.lstate.NewFunction(c.ctyLessThan))
	table.RawSet(lua.LString("__index"), c.lstate.NewFunction(c.ctyIndex))
	table.RawSet(lua.LString("__newindex"), c.lstate.NewFunction(c.ctyInvalidOp("collection is immutable")))
	table.RawSet(lua.LString("__call"), c.lstate.NewFunction(c.ctyInvalidOp("value cannot be called")))

	return table
}

func (c *Converter) ctyEq(L *lua.LState) int {
	// On the stack we should have two LUserData values, because Lua
	// only calls __eq if both operands have the same type. However, we
	// don't know if both userdatas will be our own (other packages can
	// create UserData values too) and the user may call __eq directly,
	// so we will be defensive.
	a := L.CheckUserData(1)
	b := L.CheckUserData(2)
	L.Pop(2)

	if a == nil || b == nil {
		L.Push(lua.LBool(false))
		return 1
	}
	if _, isOurs := a.Value.(cty.Value); !isOurs {
		L.Push(lua.LBool(false))
		return 1
	}
	if _, isOurs := b.Value.(cty.Value); !isOurs {
		L.Push(lua.LBool(false))
		return 1
	}

	result := a.Value.(cty.Value).Equals(b.Value.(cty.Value))
	if result.IsKnown() {
		L.Push(lua.LBool(result.True()))
	} else {
		// Lua doesn't have the concept of an unknown bool, so we just
		// treat unknown result as false. (The result of eq is forced to
		// be a native lua bool, so we can't do better here.)
		L.Push(lua.LBool(false))
	}
	return 1
}

func (c *Converter) ctyArithmetic(op func(a, b cty.Value) (cty.Value, error)) lua.LGFunction {
	return func(L *lua.LState) int {
		aL := L.CheckAny(1)
		bL := L.CheckAny(2)

		a, err := c.ToCtyValue(aL, cty.Number)
		if err != nil {
			L.Error(lua.LString(err.Error()), 1)
		}
		b, err := c.ToCtyValue(bL, cty.Number)
		if err != nil {
			L.Error(lua.LString(err.Error()), 1)
		}

		result, err := op(a, b)
		if err != nil {
			L.Error(lua.LString(err.Error()), 1)
		}

		L.Push(c.WrapCtyValue(result))
		return 1
	}
}

func (c *Converter) ctyNegate(L *lua.LState) int {
	vL := L.CheckAny(1)

	v, err := c.ToCtyValue(vL, cty.Number)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}

	result, err := stdlib.Negate(v)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}

	L.Push(c.WrapCtyValue(result))
	return 1
}

func (c *Converter) ctyConcat(L *lua.LState) int {
	aL := L.CheckAny(1)
	bL := L.CheckAny(2)

	a, err := c.ToCtyValue(aL, cty.String)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}
	b, err := c.ToCtyValue(bL, cty.String)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}

	if !(a.IsKnown() && b.IsKnown()) {
		L.Push(c.WrapCtyValue(cty.UnknownVal(cty.String)))
		return 1
	}

	result := cty.StringVal(a.AsString() + b.AsString())
	L.Push(c.WrapCtyValue(result))
	return 1
}

func (c *Converter) ctyLength(L *lua.LState) int {
	vL := L.CheckAny(1)

	v, err := c.ToCtyValue(vL, cty.DynamicPseudoType)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
	}

	if v.Type() == cty.String {
		result, err := stdlib.Strlen(v)
		if err != nil {
			L.Error(lua.LString(err.Error()), 1)
			return 0
		}

		L.Push(c.WrapCtyValue(result))
		return 1
	}

	result, err := stdlib.Length(v)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
		return 0
	}

	L.Push(c.WrapCtyValue(result))
	return 1
}

func (c *Converter) ctyLessThan(L *lua.LState) int {
	aL := L.CheckAny(1)
	bL := L.CheckAny(2)

	a, err := c.ToCtyValue(aL, cty.Number)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
		return 0
	}
	b, err := c.ToCtyValue(bL, cty.Number)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
		return 0
	}

	result, err := stdlib.LessThan(a, b)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
		return 0
	}

	if !result.IsKnown() {
		L.Push(lua.LBool(false)) // can't represent unknown as Lua bool
		return 1
	}

	L.Push(lua.LBool(result.True()))
	return 1
}

func (c *Converter) ctyIndex(L *lua.LState) int {
	collL := L.CheckAny(1)
	keyL := L.CheckAny(2)

	coll, err := c.ToCtyValue(collL, cty.DynamicPseudoType)
	if err != nil {
		L.Error(lua.LString(err.Error()), 1)
		return 0
	}

	collTy := coll.Type()
	var keyType cty.Type
	switch {
	case collTy.IsMapType() || collTy.IsObjectType():
		keyType = cty.String
	case collTy.IsListType() || collTy.IsTupleType():
		keyType = cty.Number
	default:
		L.Error(lua.LString(fmt.Sprintf("can't index value of type %s", collTy.FriendlyName())), 1)
	}

	key, err := c.ToCtyValue(keyL, keyType)
	if err != nil {
		L.Error(lua.LString(fmt.Sprintf("invalid key for %s: %s", collTy.FriendlyName(), err)), 1)
		return 0
	}

	switch {
	case collTy.IsListType() || collTy.IsMapType() || collTy.IsTupleType():
		hasIndex := coll.HasIndex(key)
		if !hasIndex.IsKnown() {
			L.Push(c.WrapCtyValue(cty.DynamicVal))
			return 1
		}

		if hasIndex.False() {
			// Lua semantics require this to return nil rather than throwing
			// an error.
			L.Push(lua.LNil)
			return 1
		}

		result := coll.Index(key)
		L.Push(c.WrapCtyValue(result))
		return 1
	case collTy.IsObjectType():
		if !key.IsKnown() {
			L.Push(c.WrapCtyValue(cty.DynamicVal))
			return 1
		}

		attrName := key.AsString()

		if !collTy.HasAttribute(attrName) {
			// Lua semantics require this to return nil rather than throwing
			// an error.
			L.Push(lua.LNil)
			return 1
		}

		result := coll.GetAttr(attrName)
		L.Push(c.WrapCtyValue(result))
		return 1
	default:
		// should never happen
		panic(fmt.Errorf("don't know how to index %#v with %#v", coll, key))
	}
}

func (c *Converter) ctyInvalidOp(msg string) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Error(lua.LString(msg), 1)
		return 0
	}
}
