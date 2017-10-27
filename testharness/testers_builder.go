package testharness

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	lua "github.com/yuin/gopher-lua"
	"github.com/zclconf/gopherlua-cty/luacty"
)

// testersBuilder provides functions for constructing a sequence of testers
// from within a Lua test spec.
type testersBuilder struct {
	Context *Context
	Testers Testers
	Diags   *Diagnostics
}

func (b *testersBuilder) luaDescribeFunc(L *lua.LState) int {
	if L.GetTop() != 2 {
		b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid \"describe\" call",
			Detail:   "A \"describe\" call must have two arguments: the object or name it is describing, and a definition function.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}

	describedL := L.CheckAny(1)
	bodyFn := L.OptFunction(2, nil)

	if bodyFn == nil {
		b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid \"describe\" call",
			Detail:   "A \"describe\" call must have a definition function as its second argument.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}

	var described contextSetter
	if lua.LVCanConvToString(describedL) {
		name := lua.LVAsString(describedL)
		described = simpleContextSetter(name)
	} else {
		if ud, isUd := describedL.(*lua.LUserData); isUd {
			if setter, isSetter := ud.Value.(contextSetter); isSetter {
				described = setter
			}
		}
	}

	if described == nil {
		b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid \"describe\" call",
			Detail:   "A \"describe\" call must have the object or name it is describing as its first argument.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}

	desc := &describe{
		Described: described,
		BodyFn:    bodyFn,
	}
	if rng := callingRange(L, 1); rng != nil {
		desc.DefRange = tfdiags.SourceRangeFromHCL(*rng)
	}

	b.Testers = append(b.Testers, desc)

	return 0
}

func (b *testersBuilder) luaResourceObj(L *lua.LState) lua.LValue {
	ret := L.NewUserData()
	meta := L.NewTable()

	conv := luacty.NewConverter(L)

	if b.Context.HasResource() {
		meta.RawSet(lua.LString("__index"), conv.WrapCtyValue(b.Context.Resource()))
		meta.RawSet(lua.LString("__call"), L.NewFunction(func(L *lua.LState) int {
			L.Error(lua.LString("a resource is already being described by a parent block"), 2)
			return 0
		}))
	} else {
		meta.RawSet(lua.LString("__index"), L.NewFunction(func(L *lua.LState) int {
			L.Error(lua.LString("no resource is being described by the current block"), 2)
			return 0
		}))
		meta.RawSet(lua.LString("__call"), L.NewFunction(func(L *lua.LState) int {
			addrS := L.CheckString(2)
			addr, err := terraform.ParseResourceAddress(addrS)
			if err != nil {
				L.Error(lua.LString(fmt.Sprintf("invalid resource address: %s", err)), 2)
				return 0
			}
			if addr.Name == "" {
				L.Error(lua.LString("invalid resource address: must refer to specific resource"), 2)
				return 0
			}

			ctxSet := &resourceContextSetter{
				Addr: addr,
			}
			if rng := callingRange(L, 1); rng != nil {
				ctxSet.DefRange = tfdiags.SourceRangeFromHCL(*rng)
			}

			ret := L.NewUserData()
			ret.Value = ctxSet
			L.Push(ret)
			return 1
		}))
	}

	ret.Metatable = meta

	return ret
}
