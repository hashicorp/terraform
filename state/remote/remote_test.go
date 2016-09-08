package remote

import (
	"bytes"
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
	defer os.Remove(localStateFile.Name())

	remoteStateFile, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	remoteStateFile.Close()
	os.Remove(remoteStateFile.Name())
	defer os.Remove(remoteStateFile.Name())

	// Now we need an empty state to initialize the state files.
	newState := terraform.NewState()
	newState.Remote = &terraform.RemoteState{
		Type:   "_local",
		Config: map[string]string{"path": remoteStateFile.Name()},
	}

	remoteClient := &FileClient{
		Path: remoteStateFile.Name(),
	}

	cache := &state.CacheState{
		Cache: &state.LocalState{
			Path: localStateFile.Name(),
		},
		Durable: &State{
			Client: remoteClient,
		},
	}

	// This will write the local state file, and set the state field in the CacheState
	err = cache.WriteState(newState)
	if err != nil {
		t.Fatal(err)
	}

	// This will persist the local state we just wrote to the remote state file
	err = cache.PersistState()
	if err != nil {
		t.Fatal(err)
	}

	// now compare the two state files just to be sure
	localData, err := ioutil.ReadFile(localStateFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	remoteData, err := ioutil.ReadFile(remoteStateFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(localData, remoteData) {
		t.Log("state files don't match")
		t.Log("Local:\n", string(localData))
		t.Log("Remote:\n", string(remoteData))
		t.Fatal("failed to initialize remote state")
	}
}
