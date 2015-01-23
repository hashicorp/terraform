package remote

import (
	"bytes"
	"crypto/md5"
	"os"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/terraform"
)

func TestConsulRemote_Interface(t *testing.T) {
	var client interface{} = &ConsulRemoteClient{}
	if _, ok := client.(RemoteClient); !ok {
		t.Fatalf("does not implement interface")
	}
}

func checkConsul(t *testing.T) {
	if os.Getenv("CONSUL_ADDR") == "" {
		t.SkipNow()
	}
}

func TestConsulRemote_Validate(t *testing.T) {
	conf := map[string]string{}
	if _, err := NewConsulRemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	conf["path"] = "test"
	if _, err := NewConsulRemoteClient(conf); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestConsulRemote_GetState(t *testing.T) {
	checkConsul(t)
	type tcase struct {
		Path      string
		Body      []byte
		ExpectMD5 []byte
		ExpectErr string
	}
	inp := []byte("testing")
	inpMD5 := md5.Sum(inp)
	hash := inpMD5[:16]
	cases := []*tcase{
		&tcase{
			Path:      "foo",
			Body:      inp,
			ExpectMD5: hash,
		},
		&tcase{
			Path: "none",
		},
	}

	for _, tc := range cases {
		if tc.Body != nil {
			conf := consulapi.DefaultConfig()
			conf.Address = os.Getenv("CONSUL_ADDR")
			client, _ := consulapi.NewClient(conf)
			pair := &consulapi.KVPair{Key: tc.Path, Value: tc.Body}
			client.KV().Put(pair, nil)
		}

		remote := &terraform.RemoteState{
			Type: "consul",
			Config: map[string]string{
				"address": os.Getenv("CONSUL_ADDR"),
				"path":    tc.Path,
			},
		}
		r, err := NewClientByState(remote)
		if err != nil {
			t.Fatalf("Err: %v", err)
		}

		payload, err := r.GetState()
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.ExpectErr {
			t.Fatalf("bad err: %v %v", errStr, tc.ExpectErr)
		}

		if tc.ExpectMD5 != nil {
			if payload == nil || !bytes.Equal(payload.MD5, tc.ExpectMD5) {
				t.Fatalf("bad: %#v", payload)
			}
		}

		if tc.Body != nil {
			if !bytes.Equal(payload.State, tc.Body) {
				t.Fatalf("bad: %#v", payload)
			}
		}
	}
}

func TestConsulRemote_PutState(t *testing.T) {
	checkConsul(t)
	path := "foobar"
	inp := []byte("testing")

	remote := &terraform.RemoteState{
		Type: "consul",
		Config: map[string]string{
			"address": os.Getenv("CONSUL_ADDR"),
			"path":    path,
		},
	}
	r, err := NewClientByState(remote)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	err = r.PutState(inp, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	conf := consulapi.DefaultConfig()
	conf.Address = os.Getenv("CONSUL_ADDR")
	client, _ := consulapi.NewClient(conf)
	pair, _, err := client.KV().Get(path, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !bytes.Equal(pair.Value, inp) {
		t.Fatalf("bad value")
	}
}

func TestConsulRemote_DeleteState(t *testing.T) {
	checkConsul(t)
	path := "testdelete"

	// Create the state
	conf := consulapi.DefaultConfig()
	conf.Address = os.Getenv("CONSUL_ADDR")
	client, _ := consulapi.NewClient(conf)
	pair := &consulapi.KVPair{Key: path, Value: []byte("test")}
	client.KV().Put(pair, nil)

	remote := &terraform.RemoteState{
		Type: "consul",
		Config: map[string]string{
			"address": os.Getenv("CONSUL_ADDR"),
			"path":    path,
		},
	}
	r, err := NewClientByState(remote)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	err = r.DeleteState()
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	pair, _, err = client.KV().Get(path, nil)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	if pair != nil {
		t.Fatalf("state not deleted")
	}
}
