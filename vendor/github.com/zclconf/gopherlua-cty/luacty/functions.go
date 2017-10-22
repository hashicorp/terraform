package luacty

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// WrapCtyFunction wraps a cty function so that it can be called from Lua
// as a Lua function.
//
// Arguments to the produced Lua function are converted to the cty types
// required by the function. The return value is always a cty Value wrapped
// in a Lua userdata, as would be returned from WrapCtyValue.
func (c *Converter) WrapCtyFunction(f function.Function) *lua.LFunction {
	params := f.Params()
	varParam := f.VarParam()
	return c.lstate.NewFunction(func(L *lua.LState) int {
		nArg := L.GetTop()
		args := make([]cty.Value, nArg)

		for i := range args {
			var param function.Parameter
			if i < len(params) {
				param = params[i]
			} else {
				if varParam == nil {
					L.ArgError(i+1, "too many arguments")
					return 0
				}
				param = *varParam
			}

			vL := L.CheckAny(i + 1)
			v, err := c.ToCtyValue(vL, param.Type)
			if err != nil {
				L.ArgError(i+1, err.Error())
				return 0
			}
			args[i] = v
		}

		result, err := f.Call(args)
		if err != nil {
			L.Error(lua.LString(err.Error()), 1)
			return 0
		}

		L.Push(c.WrapCtyValue(result))
		return 1
	})
}

// ToCtyFunction wraps a Lua function so that it can be used as a cty
// function.
//
// Since Lua functions do not have statically-defined argument types,
// all of the parameters in the returned function are typed as
// cty.DynamicPseudoType, and the return type is also cty.DynamicPseudoType.
func (c *Converter) ToCtyFunction(f *lua.LFunction) function.Function {
	proto := f.Proto

	spec := &function.Spec{
		Type: function.StaticReturnType(cty.DynamicPseudoType),
	}

	if proto != nil {
		// If we have a prototype then we'll translate it into a well-specified
		// parameter list for the cty function.
		if proto.NumParameters > 0 {
			spec.Params = make([]function.Parameter, proto.NumParameters)
			for i := range spec.Params {
				spec.Params[i] = function.Parameter{
					Name: fmt.Sprintf("arg%d", i+1),
					Type: cty.DynamicPseudoType,
				}
			}
			if proto.IsVarArg != 0 {
				spec.VarParam = &function.Parameter{
					Name: "...",
					Type: cty.DynamicPseudoType,
				}
			}
		}
	} else {
		// If there's no prototype (e.g. because this is a wrapped Go function)
		// then we'll generate a variadic-only signature.
		spec.VarParam = &function.Parameter{
			Name: "...",
			Type: cty.DynamicPseudoType,
		}
	}

	spec.Impl = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		for _, arg := range args {
			c.lstate.Push(c.WrapCtyValue(arg))
		}
		err := c.lstate.PCall(len(args), 1, f)
		if err != nil {
			return cty.DynamicVal, err
		}
		resultL := c.lstate.CheckAny(1)
		result, err := c.ToCtyValue(resultL, retType)
		if err != nil {
			return cty.DynamicVal, err
		}
		return result, nil
	}

	return function.New(spec)
}
