package funcs

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
	"golang.org/x/crypto/bcrypt"
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

// Base64Sha256Func constructs a function that computes the SHA256 hash of a given string and encodes it with
// Base64.
var Base64Sha256Func = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		s := args[0].AsString()
		h := sha256.New()
		h.Write([]byte(s))
		shaSum := h.Sum(nil)
		return cty.StringVal(base64.StdEncoding.EncodeToString(shaSum[:])), nil
	},
})

// Base64Sha512Func constructs a function that computes the SHA256 hash of a given string and encodes it with
// Base64.
var Base64Sha512Func = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		s := args[0].AsString()
		h := sha512.New()
		h.Write([]byte(s))
		shaSum := h.Sum(nil)
		return cty.StringVal(base64.StdEncoding.EncodeToString(shaSum[:])), nil
	},
})

// BcryptFunc constructs a function that computes a hash of the given string using the Blowfish cipher.
var BcryptFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "str",
			Type: cty.String,
		},
	},
	VarParam: &function.Parameter{
		Name: "cost",
		Type: cty.Number,
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		defaultCost := 10

		if len(args) > 1 {
			var val int
			if err := gocty.FromCtyValue(args[1], &val); err != nil {
				return cty.UnknownVal(cty.String), err
			}
			defaultCost = val
		}

		if len(args) > 2 {
			return cty.UnknownVal(cty.String), fmt.Errorf("bcrypt() takes no more than two arguments")
		}

		input := args[0].AsString()
		out, err := bcrypt.GenerateFromPassword([]byte(input), defaultCost)
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("error occured generating password %s", err.Error())
		}

		return cty.StringVal(string(out)), nil
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

// Base64sha256 computes the SHA256 hash of a given string and encodes it with
// Base64.
//
// The given string is first encoded as UTF-8 and then the SHA256 algorithm is applied
// as defined in [RFC 4634](https://tools.ietf.org/html/rfc4634). The raw hash is
// then encoded with Base64 before returning. Terraform uses the "standard" Base64
// alphabet as defined in [RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).
func Base64Sha256(str cty.Value) (cty.Value, error) {
	return Base64Sha256Func.Call([]cty.Value{str})
}

// Base64sha512 computes the SHA512 hash of a given string and encodes it with
// Base64.
//
// The given string is first encoded as UTF-8 and then the SHA256 algorithm is applied
// as defined in [RFC 4634](https://tools.ietf.org/html/rfc4634). The raw hash is
// then encoded with Base64 before returning. Terraform uses the "standard" Base64
// alphabet as defined in [RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).
func Base64Sha512(str cty.Value) (cty.Value, error) {
	return Base64Sha512Func.Call([]cty.Value{str})
}

// Bcrypt computes a hash of the given string using the Blowfish cipher,
// returning a string in the Modular Crypt Format(https://passlib.readthedocs.io/en/stable/modular_crypt_format.html)
// usually expected in the shadow password file on many Unix systems.
func Bcrypt(str cty.Value, cost ...cty.Value) (cty.Value, error) {
	args := make([]cty.Value, len(cost)+1)
	args[0] = str
	copy(args[1:], cost)
	return BcryptFunc.Call(args)
}
