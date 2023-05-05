// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsondiff

import "fmt"

type Type string

const (
	Number Type = "number"
	Object Type = "object"
	Array  Type = "array"
	Bool   Type = "bool"
	String Type = "string"
	Null   Type = "null"
)

func GetType(json interface{}) Type {
	switch json.(type) {
	case []interface{}:
		return Array
	case float64:
		return Number
	case string:
		return String
	case bool:
		return Bool
	case nil:
		return Null
	case map[string]interface{}:
		return Object
	default:
		panic(fmt.Sprintf("unrecognized json type %T", json))
	}
}
