package states

import (
	"github.com/hashicorp/terraform/addrs"
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
