package fastly

import (
	"bytes"
	"encoding"
)

type statusResp struct {
	Status string
	Msg    string
}

func (t *statusResp) Ok() bool {
	return t.Status == "ok"
}

// Ensure Compatibool implements the proper interfaces.
var (
	_ encoding.TextMarshaler   = new(Compatibool)
	_ encoding.TextUnmarshaler = new(Compatibool)
)

// Compatibool is a boolean value that marshalls to 0/1 instead of true/false
// for compatability with Fastly's API.
type Compatibool bool

// MarshalText implements the encoding.TextMarshaler interface.
func (b Compatibool) MarshalText() ([]byte, error) {
	if b {
		return []byte("1"), nil
	}
	return []byte("0"), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (b Compatibool) UnmarshalText(t []byte) error {
	if bytes.Equal(t, []byte("1")) {
		b = Compatibool(true)
	}
	return nil
}
