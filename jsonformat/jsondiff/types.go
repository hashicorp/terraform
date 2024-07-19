// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsondiff

import (
	"encoding/json"
	"fmt"
)

type Type string

const (
	Number Type = "number"
	Object Type = "object"
	Array  Type = "array"
	Bool   Type = "bool"
	String Type = "string"
	Null   Type = "null"
)

func GetType(value interface{}) Type {
	switch value.(type) {
	case []interface{}:
		return Array
	case json.Number:
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
		panic(fmt.Sprintf("unrecognized json type %T", value))
	}
}
