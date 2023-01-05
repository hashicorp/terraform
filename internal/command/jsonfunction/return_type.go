package jsonfunction

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func getReturnType(f function.Function) string {
	args := make([]cty.Type, 0)
	for _, param := range f.Params() {
		args = append(args, param.Type)
	}
	if f.VarParam() != nil {
		args = append(args, f.VarParam().Type)
	}

	returnType, err := f.ReturnType(args)
	if err != nil {
		return "" // TODO? handle error
	}
	return returnType.FriendlyName()
}
