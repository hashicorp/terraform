package remote

import "testing"

func TestAtlasRemote_Interface(t *testing.T) {
	var client interface{} = &AtlasRemoteClient{}
	if _, ok := client.(RemoteClient); !ok {
		t.Fatalf("does not implement interface")
	}
}

func TestAtlasRemote(t *testing.T) {
	// TODO
}
