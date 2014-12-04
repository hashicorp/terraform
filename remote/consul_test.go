package remote

import "testing"

func TestConsulRemote_Interface(t *testing.T) {
	var client interface{} = &ConsulRemoteClient{}
	if _, ok := client.(RemoteClient); !ok {
		t.Fatalf("does not implement interface")
	}
}

func TestConsulRemote(t *testing.T) {
	// TODO
}
