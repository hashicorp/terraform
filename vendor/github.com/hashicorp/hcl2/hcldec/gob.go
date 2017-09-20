package hcldec

import (
	"encoding/gob"
)

func init() {
	// Every Spec implementation should be registered with gob, so that
	// specs can be sent over gob channels, such as using
	// github.com/hashicorp/go-plugin with plugins that need to describe
	// what shape of configuration they are expecting.
	gob.Register(ObjectSpec(nil))
	gob.Register(TupleSpec(nil))
	gob.Register((*AttrSpec)(nil))
	gob.Register((*LiteralSpec)(nil))
	gob.Register((*ExprSpec)(nil))
	gob.Register((*BlockSpec)(nil))
	gob.Register((*BlockListSpec)(nil))
	gob.Register((*BlockSetSpec)(nil))
	gob.Register((*BlockMapSpec)(nil))
	gob.Register((*BlockLabelSpec)(nil))
}
