package chef

import (
	"bytes"
	"encoding/json"
	"io"
)

// JSONReader handles arbitrary types and synthesizes a streaming encoder for them.
func JSONReader(v interface{}) (r io.Reader, err error) {
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(v)
	r = bytes.NewReader(buf.Bytes())
	return
}
