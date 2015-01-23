package remote

import (
	"bytes"
	"crypto/md5"
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestAtlasRemote_Interface(t *testing.T) {
	var client interface{} = &AtlasRemoteClient{}
	if _, ok := client.(RemoteClient); !ok {
		t.Fatalf("does not implement interface")
	}
}

func checkAtlas(t *testing.T) {
	if os.Getenv("ATLAS_TOKEN") == "" {
		t.SkipNow()
	}
}

func TestAtlasRemote_Validate(t *testing.T) {
	conf := map[string]string{}
	if _, err := NewAtlasRemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	conf["access_token"] = "test"
	conf["name"] = "hashicorp/test-state"
	if _, err := NewAtlasRemoteClient(conf); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestAtlasRemote_Validate_envVar(t *testing.T) {
	conf := map[string]string{}
	if _, err := NewAtlasRemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	defer os.Setenv("ATLAS_TOKEN", os.Getenv("ATLAS_TOKEN"))
	os.Setenv("ATLAS_TOKEN", "foo")

	conf["name"] = "hashicorp/test-state"
	if _, err := NewAtlasRemoteClient(conf); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestAtlasRemote(t *testing.T) {
	checkAtlas(t)
	remote := &terraform.RemoteState{
		Type: "atlas",
		Config: map[string]string{
			"access_token": os.Getenv("ATLAS_TOKEN"),
			"name":         "hashicorp/test-remote-state",
		},
	}
	r, err := NewClientByState(remote)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	// Get a valid input
	inp, err := blankState(remote)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	inpMD5 := md5.Sum(inp)
	hash := inpMD5[:16]

	// Delete the state, should be none
	err = r.DeleteState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure no state
	payload, err := r.GetState()
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	if payload != nil {
		t.Fatalf("unexpected payload")
	}

	// Put the state
	err = r.PutState(inp, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Get it back
	payload, err = r.GetState()
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	if payload == nil {
		t.Fatalf("unexpected payload")
	}

	// Check the payload
	if !bytes.Equal(payload.MD5, hash) {
		t.Fatalf("bad hash: %x %x", payload.MD5, hash)
	}
	if !bytes.Equal(payload.State, inp) {
		t.Errorf("inp: %s", inp)
		t.Fatalf("bad response: %s", payload.State)
	}

	// Delete the state
	err = r.DeleteState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Should be gone
	payload, err = r.GetState()
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	if payload != nil {
		t.Fatalf("unexpected payload")
	}
}
