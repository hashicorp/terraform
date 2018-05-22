package funcs

import (
	uuid "github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var UUIDFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		result, err := uuid.GenerateUUID()
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}
		return cty.StringVal(result), nil
	},
})

// UUID generates and returns a Type-4 UUID in the standard hexadecimal string
// format.
//
// This is not a pure function: it will generate a different result for each
// call. It must therefore be registered as an impure function in the function
// table in the "lang" package.
func UUID() (cty.Value, error) {
	return UUIDFunc.Call(nil)
}
