package configs

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/zclconf/go-cty/cty"
)

// Backend represents a "backend" block inside a "terraform" block in a module
// or file.
type Backend struct {
	Type   string
	Config hcl.Body

	TypeRange hcl.Range
	DeclRange hcl.Range
}

func decodeBackendBlock(block *hcl.Block) (*Backend, hcl.Diagnostics) {
	return &Backend{
		Type:      block.Labels[0],
		TypeRange: block.LabelRanges[0],
		Config:    block.Body,
		DeclRange: block.DefRange,
	}, nil
}

// Hash produces a hash value for the reciever that covers the type and the
// portions of the config that conform to the given schema.
//
// If the config does not conform to the schema then the result is not
// meaningful for comparison since it will be based on an incomplete result.
//
// As an exception, required attributes in the schema are treated as optional
// for the purpose of hashing, so that an incomplete configuration can still
// be hashed. Other errors, such as extraneous attributes, have no such special
// case.
func (b *Backend) Hash(schema *configschema.Block) int {
	// Don't fail if required attributes are not set. Instead, we'll just
	// hash them as nulls.
	schema = schema.NoneRequired()
	spec := schema.DecoderSpec()
	val, _ := hcldec.Decode(b.Config, spec, nil)
	if val == cty.NilVal {
		val = cty.UnknownVal(schema.ImpliedType())
	}

	toHash := cty.TupleVal([]cty.Value{
		cty.StringVal(b.Type),
		val,
	})

	return toHash.Hash()
}
