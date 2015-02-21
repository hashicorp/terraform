package remote

import (
	"bytes"
	"testing"
)

// testClient is a generic function to test any client.
func testClient(t *testing.T, c Client) {
	data := []byte("foo")

	if err := c.Put(data); err != nil {
		t.Fatalf("put: %s", err)
	}

	p, err := c.Get()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if !bytes.Equal(p.Data, data) {
		t.Fatalf("bad: %#v", p)
	}

	if err := c.Delete(); err != nil {
		t.Fatalf("delete: %s", err)
	}

	p, err = c.Get()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if p != nil {
		t.Fatalf("bad: %#v", p)
	}
}
