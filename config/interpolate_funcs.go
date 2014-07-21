package config

// Funcs is the mapping of built-in functions for configuration.
var Funcs map[string]InterpolationFunc

func init() {
	Funcs = map[string]InterpolationFunc{
		"lookup": interpolationFuncLookup,
	}
}

// interpolationFuncLookup implements the "lookup" function that allows
// dynamic lookups of map types within a Terraform configuration.
func interpolationFuncLookup(
	vs map[string]string, args ...string) (string, error) {
	return "", nil
}
