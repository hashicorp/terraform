package cty

import (
	"encoding/gob"
	"fmt"
	"math/big"
	"strings"

	"github.com/zclconf/go-cty/cty/set"
)

// InternalTypesToRegister is a slice of values that covers all of the
// internal types used in the representation of cty.Type and cty.Value
// across all cty Types.
//
// This is intended to be used to register these types with encoding
// packages that require registration of types used in interfaces, such as
// encoding/gob, thus allowing cty types and values to be included in streams
// created from those packages. However, registering with gob is not necessary
// since that is done automatically as a side-effect of importing this package.
//
// Callers should not do anything with the values here except pass them on
// verbatim to a registration function.
//
// If the calling application uses Capsule types that wrap local structs either
// directly or indirectly, these structs may also need to be registered in
// order to support encoding and decoding of values of these types. That is the
// responsibility of the calling application.
var InternalTypesToRegister []interface{}

func init() {
	InternalTypesToRegister = []interface{}{
		primitiveType{},
		typeList{},
		typeMap{},
		typeObject{},
		typeSet{},
		setRules{},
		set.Set{},
		typeTuple{},
		big.Float{},
		capsuleType{},
		[]interface{}(nil),
		map[string]interface{}(nil),
	}

	// Register these with gob here, rather than in gob.go, to ensure
	// that this will always happen after we build the above.
	for _, tv := range InternalTypesToRegister {
		typeName := fmt.Sprintf("%T", tv)
		if strings.HasPrefix(typeName, "cty.") {
			gob.RegisterName(fmt.Sprintf("github.com/zclconf/go-cty/%s", typeName), tv)
		} else {
			gob.Register(tv)
		}
	}
}
