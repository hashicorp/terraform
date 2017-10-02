package configschema

import (
	"github.com/hashicorp/hcl2/hcldec"
)

// DecoderSpec returns a zcldec.Spec that can be used to decode a zcl Body
// using the facilities in the zcldec package.
//
// The returned specification is guaranteed to return a value of the same type
// returned by method ImpliedType, but it may contain null or unknown values if
// any of the block attributes are defined as optional and/or computed
// respectively.
func (b *Block) DecoderSpec() hcldec.Spec {
	return nil
}
