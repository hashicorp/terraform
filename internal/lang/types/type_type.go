// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package types

import (
	"reflect"

	"github.com/zclconf/go-cty/cty"
)

// TypeType is a capsule type used to represent a cty.Type as a cty.Value. This
// is used by the `type()` console function to smuggle cty.Type values to the
// REPL session, where it can be displayed to the user directly.
var TypeType = cty.Capsule("type", reflect.TypeOf(cty.Type{}))
