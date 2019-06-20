package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
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

// mockClient is a client that tracks persisted state snapshots only in
// memory and also logs what it has been asked to do for use in test
// assertions.
type mockClient struct {
	current []byte
	log     []mockClientRequest
}

type mockClientRequest struct {
	Method  string
	Content map[string]interface{}
}

func (c *mockClient) Get() (*Payload, error) {
	c.appendLog("Get", c.current)
	if c.current == nil {
		return nil, nil
	}
	checksum := md5.Sum(c.current)
	return &Payload{
		Data: c.current,
		MD5:  checksum[:],
	}, nil
}

func (c *mockClient) Put(data []byte) error {
	c.appendLog("Put", data)
	c.current = data
	return nil
}

func (c *mockClient) Delete() error {
	c.appendLog("Delete", c.current)
	c.current = nil
	return nil
}

func (c *mockClient) appendLog(method string, content []byte) {
	// For easier test assertions, we actually log the result of decoding
	// the content JSON rather than the raw bytes. Callers are in principle
	// allowed to provide any arbitrary bytes here, but we know we're only
	// using this to test our own State implementation here and that always
	// uses the JSON state format, so this is fine.

	var contentVal map[string]interface{}
	if content != nil {
		err := json.Unmarshal(content, &contentVal)
		if err != nil {
			panic(err) // should never happen because our tests control this input
		}
	}
	c.log = append(c.log, mockClientRequest{method, contentVal})
}
