package jsonfunction

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func getReturnType(f function.Function) ([]byte, error) {
	args := make([]cty.Type, 0)
	for _, param := range f.Params() {
		args = append(args, param.Type)
	}
	if f.VarParam() != nil {
		args = append(args, f.VarParam().Type)
	}

	returnType, err := f.ReturnType(args)
	if err != nil {
		return nil, err
	}
	return ctyjson.MarshalType(returnType)
}
