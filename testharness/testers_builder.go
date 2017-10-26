package testharness

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
	lua "github.com/yuin/gopher-lua"
)

// testersBuilder provides functions for constructing a sequence of testers
// from within a Lua test spec.
type testersBuilder struct {
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
