package remote

import (
	"bytes"
	"crypto/md5"
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestS3Remote_NewClient(t *testing.T) {
	conf := map[string]string{}
	if _, err := NewS3RemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	conf["access_token"] = "test"
	conf["secret_token"] = "test"
	conf["address"] = "s3://plan3-test/hashicorp/test-state"
	conf["region"] = "eu-west-1"
	if _, err := NewS3RemoteClient(conf); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestS3Remote_Validate_envVar(t *testing.T) {
	conf := map[string]string{}
	if _, err := NewS3RemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	defer os.Setenv("AWS_ACCESS_KEY", os.Getenv("AWS_ACCESS_KEY"))
	os.Setenv("AWS_ACCESS_KEY", "foo")

	defer os.Setenv("AWS_SECRET_KEY", os.Getenv("AWS_SECRET_KEY"))
	os.Setenv("AWS_SECRET_KEY", "foo")

	defer os.Setenv("AWS_DEFAULT_REGION", os.Getenv("AWS_DEFAULT_REGION"))
	os.Setenv("AWS_DEFAULT_REGION", "eu-west-1")

	conf["address"] = "s3://terraform-state/hashicorp/test-state"
	if _, err := NewS3RemoteClient(conf); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func checkS3(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY") == "" || os.Getenv("AWS_SECRET_KEY") == "" || os.Getenv("AWS_DEFAULT_REGION") == "" || os.Getenv("TERRAFORM_STATE_BUCKET") == "" {
		t.SkipNow()
	}
}

func TestS3Remote(t *testing.T) {
	checkS3(t)
	remote := &terraform.RemoteState{
		Type: "atlas",
		Config: map[string]string{
			"access_token": "some-access-token",
			"name":         "hashicorp/test-remote-state",
		},
	}
	r, err := NewClientByType("s3", map[string]string{
		"bucket": os.Getenv("TERRAFORM_STATE_BUCKET"),
		"path":   "test-remote-state",
	})
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
