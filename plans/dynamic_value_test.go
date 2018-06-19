package plans

import (
	"github.com/zclconf/go-cty/cty"
)

func mustNewDynamicValue(val cty.Value, ty cty.Type) DynamicValue {
	ret, err := NewDynamicValue(val, ty)
	if err != nil {
		panic(err)
	}
	return ret
}
