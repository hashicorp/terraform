package funcs

import (
	"encoding/base64"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// Base64DecodeFunc constructs a function that decodes a string containing a base64 sequence.
var Base64DecodeFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		s := args[0].AsString()
		sDec, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("failed to decode base64 data '%s'", s)
		}

		return cty.StringVal(string(sDec)), nil
	},
})

// Base64Decode decodes a string containing a base64 sequence.
//
// Terraform uses the "standard" Base64 alphabet as defined in
// [RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).
//
// Strings in the Terraform language are sequences of unicode characters rather
// than bytes, so this function will also interpret the resulting bytes as
// UTF-8. If the bytes after Base64 decoding are _not_ valid UTF-8, this function
// produces an error.
func Base64Decode(str cty.Value) (cty.Value, error) {
	return Base64DecodeFunc.Call([]cty.Value{str})
}
