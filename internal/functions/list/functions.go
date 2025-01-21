package list

import (
	"github.com/zclconf/go-cty/cty/function"
)

func Functions() map[string]function.Function {
	return map[string]function.Function{
		"transpose": TransposeFunc(),
	}
}
