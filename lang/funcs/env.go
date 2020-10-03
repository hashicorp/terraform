package funcs

import (
	"fmt"
	"os"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// EnvFunc constructs a function that takes a key string and returns the value
// of the environment variable named by the key, or an error if it doesn't exist.
var EnvFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "key",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if val, exists := os.LookupEnv(args[0].AsString()); exists {
			return cty.StringVal(val), nil
		}
		return cty.NilVal, fmt.Errorf("environment variable doesn't exist")
	},
})

// EnvExistsFunc constructs a function that takes a key string and returns a
// boolean value reflecting whether an environment variable named by the key exists.
var EnvExistsFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "key",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.Bool),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		_, exists := os.LookupEnv(args[0].AsString())
		return cty.BoolVal(exists), nil
	},
})

// Env retrieves the value of the environment variable named by the key.
// It returns the value, or an error if the variable is not present.
func Env(key cty.Value) (cty.Value, error) {
	return EnvFunc.Call([]cty.Value{key})
}

// EnvExists takes a key string and returns a boolean value reflecting whether
// an environment variable named by the key exists.
func EnvExists(key cty.Value) (cty.Value, error) {
	return EnvExistsFunc.Call([]cty.Value{key})
}
