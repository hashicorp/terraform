package differ

import "github.com/hashicorp/terraform/internal/command/jsonformat/change"

func (v Value) ComputeChangeForOutput() change.Change {
	panic("not implemented")
}
