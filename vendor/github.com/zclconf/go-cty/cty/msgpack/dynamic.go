package msgpack

import (
	"bytes"

	"github.com/vmihailenco/msgpack/v4"
	"github.com/zclconf/go-cty/cty"
)

type dynamicVal struct {
	Value cty.Value
	Path  cty.Path
}

func (dv *dynamicVal) MarshalMsgpack() ([]byte, error) {
	// Rather than defining a msgpack-specific serialization of types,
	// instead we use the existing JSON serialization.
	typeJSON, err := dv.Value.Type().MarshalJSON()
	if err != nil {
		return nil, dv.Path.NewErrorf("failed to serialize type: %s", err)
	}
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.EncodeArrayLen(2)
	enc.EncodeBytes(typeJSON)
	err = marshal(dv.Value, dv.Value.Type(), dv.Path, enc)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
