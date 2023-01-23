package funcs

import (
	"os"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// EnvVarFunc constructs a function that reads the specfied environment variable
// and returns its value or a specified default value.
var EnvVarFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "envvar",
			Type: cty.String,
		},
		{
			Name: "default",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		envvar := args[0].AsString()
		defaultVal := args[1].AsString()

		v, ok := os.LookupEnv(envvar)
		if !ok {
			return cty.StringVal(defaultVal), nil
		}

		return cty.StringVal(v), nil
	},
})

// EnvVar reads the value of an environment variable or returns a default value.
func EnvVar(env cty.Value, defaultValue cty.Value) (cty.Value, error) {
	return EnvVarFunc.Call([]cty.Value{env, defaultValue})
}
