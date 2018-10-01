package remote

import (
	"bytes"
	"testing"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/states/statefile"
)

// testClient is a generic function to test any client.
func testClient(t *testing.T, c Client) {
	var buf bytes.Buffer
	s := state.TestStateInitial()
	sf := &statefile.File{State: s}
	if err := statefile.Write(sf, &buf); err != nil {
		t.Fatalf("err: %s", err)
	}
	data := buf.Bytes()

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

func TestRemoteClient_noPayload(t *testing.T) {
	s := &State{
		Client: nilClient{},
	}
	if err := s.RefreshState(); err != nil {
		t.Fatal("error refreshing empty remote state")
	}
}

// nilClient returns nil for everything
type nilClient struct{}

func (nilClient) Get() (*Payload, error) { return nil, nil }

func (c nilClient) Put([]byte) error { return nil }

func (c nilClient) Delete() error { return nil }
