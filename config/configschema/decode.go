package configschema

import (
	"github.com/zclconf/go-zcl/zcldec"
)

// DecoderSpec returns a zcldec.Spec that can be used to decode a zcl Body
// using the facilities in the zcldec package.
//
// The returned specification is guaranteed to return a value of the type
// returned by method ImpliedType, but it may contain null values if any
// of the block attributes are defined as optional and/or computed.
func (b *Block) DecoderSpec() zcldec.Spec {
	// TODO: Implement
	return nil
}
