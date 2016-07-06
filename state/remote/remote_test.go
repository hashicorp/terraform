package remote

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// testClient is a generic function to test any client.
func testClient(t *testing.T, c Client) {
	var buf bytes.Buffer
	s := state.TestStateInitial()
	if err := terraform.WriteState(s, &buf); err != nil {
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

// ensure that remote state can be properly initialized
func TestRemoteClient_stateInit(t *testing.T) {
	localStateFile, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatal(err)
	}

	// we need to remove the temp files so we recognize there's no local or
	// remote state.
	localStateFile.Close()
	os.Remove(localStateFile.Name())
	//defer os.Remove(localStateFile.Name())
	fmt.Println("LOCAL:", localStateFile.Name())

	local := &state.LocalState{
		Path: localStateFile.Name(),
	}
	if err := local.RefreshState(); err != nil {
		t.Fatal(err)
	}
	localState := local.State()

	fmt.Println("localState.Empty():", localState.Empty())

	remoteStateFile, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	remoteStateFile.Close()
	os.Remove(remoteStateFile.Name())
	//defer os.Remove(remoteStateFile.Name()
	fmt.Println("LOCAL:", localStateFile.Name())
	fmt.Println("REMOTE:", remoteStateFile.Name())

	remoteClient := &FileClient{
		Path: remoteStateFile.Name(),
	}

	durable := &State{
		Client: remoteClient,
	}

	cache := &state.CacheState{
		Cache:   local,
		Durable: durable,
	}

	if err := cache.RefreshState(); err != nil {
		t.Fatal(err)
	}

	switch cache.RefreshResult() {

	// we should be "refreshing" the remote state to initialize it
	case state.CacheRefreshLocalNewer:
		// Write our local state out to the durable storage to start.
		if err := cache.WriteState(localState); err != nil {
			t.Fatal("Error preparing remote state:", err)
		}
		if err := cache.PersistState(); err != nil {
			t.Fatal("Error preparing remote state:", err)
		}
	default:

		t.Fatal("unexpected refresh result:", cache.RefreshResult())
	}

}
