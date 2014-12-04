package remote

import "testing"

func TestHTTPRemote_Interface(t *testing.T) {
	var client interface{} = &HTTPRemoteClient{}
	if _, ok := client.(RemoteClient); !ok {
		t.Fatalf("does not implement interface")
	}
}

func TestHTTPRemote(t *testing.T) {
	// TODO
}
