package funcs

import (
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"hash"

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

// Base64Sha256Func constructs a function that computes the SHA256 hash of a given string
// and encodes it with Base64.
var Base64Sha256Func = makeStringHashFunction(sha256.New, base64.StdEncoding.EncodeToString)

// Base64Sha512Func constructs a function that computes the SHA256 hash of a given string
// and encodes it with Base64.
var Base64Sha512Func = makeStringHashFunction(sha512.New, base64.StdEncoding.EncodeToString)

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

// Md5Func constructs a function that computes the MD5 hash of a given string and encodes it with hexadecimal digits.
var Md5Func = makeStringHashFunction(md5.New, hex.EncodeToString)

// RsaDecryptFunc constructs a function that decrypts an RSA-encrypted ciphertext.
var RsaDecryptFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "ciphertext",
			Type: cty.String,
		},
		{
			Name: "privatekey",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		s := args[0].AsString()
		key := args[1].AsString()

		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("failed to decode input %q: cipher text must be base64-encoded", s)
		}

		block, _ := pem.Decode([]byte(key))
		if block == nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("failed to parse key: no key found")
		}
		if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
			return cty.UnknownVal(cty.String), fmt.Errorf(
				"failed to parse key: password protected keys are not supported. Please decrypt the key prior to use",
			)
		}

		x509Key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		out, err := rsa.DecryptPKCS1v15(nil, x509Key, b)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(string(out)), nil
	},
})

// Sha1Func contructs a function that computes the SHA1 hash of a given string
// and encodes it with hexadecimal digits.
var Sha1Func = makeStringHashFunction(sha1.New, hex.EncodeToString)

// Sha256Func contructs a function that computes the SHA256 hash of a given string
// and encodes it with hexadecimal digits.
var Sha256Func = makeStringHashFunction(sha256.New, hex.EncodeToString)

// Sha512Func contructs a function that computes the SHA512 hash of a given string
// and encodes it with hexadecimal digits.
var Sha512Func = makeStringHashFunction(sha512.New, hex.EncodeToString)

func makeStringHashFunction(hf func() hash.Hash, enc func([]byte) string) function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "str",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
			s := args[0].AsString()
			h := hf()
			h.Write([]byte(s))
			rv := enc(h.Sum(nil))
			return cty.StringVal(rv), nil
		},
	})
}

// UUID generates and returns a Type-4 UUID in the standard hexadecimal string
// format.
//
// This is not a pure function: it will generate a different result for each
// call. It must therefore be registered as an impure function in the function
// table in the "lang" package.
func UUID() (cty.Value, error) {
	return UUIDFunc.Call(nil)
}

// Base64Sha256 computes the SHA256 hash of a given string and encodes it with
// Base64.
//
// The given string is first encoded as UTF-8 and then the SHA256 algorithm is applied
// as defined in RFC 4634. The raw hash is then encoded with Base64 before returning.
// Terraform uses the "standard" Base64 alphabet as defined in RFC 4648 section 4.
func Base64Sha256(str cty.Value) (cty.Value, error) {
	return Base64Sha256Func.Call([]cty.Value{str})
}

// Base64Sha512 computes the SHA512 hash of a given string and encodes it with
// Base64.
//
// The given string is first encoded as UTF-8 and then the SHA256 algorithm is applied
// as defined in RFC 4634. The raw hash is then encoded with Base64 before returning.
// Terraform uses the "standard" Base64  alphabet as defined in RFC 4648 section 4
func Base64Sha512(str cty.Value) (cty.Value, error) {
	return Base64Sha512Func.Call([]cty.Value{str})
}

// Bcrypt computes a hash of the given string using the Blowfish cipher,
// returning a string in the Modular Crypt Format
// usually expected in the shadow password file on many Unix systems.
func Bcrypt(str cty.Value, cost ...cty.Value) (cty.Value, error) {
	args := make([]cty.Value, len(cost)+1)
	args[0] = str
	copy(args[1:], cost)
	return BcryptFunc.Call(args)
}

// Md5 computes the MD5 hash of a given string and encodes it with hexadecimal digits.
func Md5(str cty.Value) (cty.Value, error) {
	return Md5Func.Call([]cty.Value{str})
}

// RsaDecrypt decrypts an RSA-encrypted ciphertext, returning the corresponding
// cleartext.
func RsaDecrypt(ciphertext, privatekey cty.Value) (cty.Value, error) {
	return RsaDecryptFunc.Call([]cty.Value{ciphertext, privatekey})
}

// Sha1 computes the SHA1 hash of a given string and encodes it with hexadecimal digits.
func Sha1(str cty.Value) (cty.Value, error) {
	return Sha1Func.Call([]cty.Value{str})
}

// Sha256 computes the SHA256 hash of a given string and encodes it with hexadecimal digits.
func Sha256(str cty.Value) (cty.Value, error) {
	return Sha256Func.Call([]cty.Value{str})
}

// Sha512 computes the SHA512 hash of a given string and encodes it with hexadecimal digits.
func Sha512(str cty.Value) (cty.Value, error) {
	return Sha512Func.Call([]cty.Value{str})
}
