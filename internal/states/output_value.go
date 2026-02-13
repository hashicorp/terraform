// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package states

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// OutputValue represents the state of a particular output value.
//
// It is not valid to mutate an OutputValue object once it has been created.
// Instead, create an entirely new OutputValue to replace the previous one.
type OutputValue struct {
	Addr      addrs.AbsOutputValue
	Value     cty.Value
	Sensitive bool
}

func (o *OutputValue) Equal(other *OutputValue) bool {
	if o == other {
		return true
	}
	if o == nil || other == nil {
		return false
	}
	if !o.Addr.Equal(other.Addr) {
		return false
	}
	if o.Sensitive != other.Sensitive {
		return false
	}
	return o.Value.RawEquals(other.Value)
}
