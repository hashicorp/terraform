package edgegrid

import "bytes"

type reader struct {
	*bytes.Buffer
}

func (m reader) Close() error { return nil }
