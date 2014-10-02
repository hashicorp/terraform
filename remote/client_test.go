package remote

import (
	"bytes"
	"crypto/md5"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/armon/consul-api"
	"github.com/hashicorp/terraform/terraform"
)

var haveInternet bool

func init() {
	// Use google to check if we are on the net
	_, err := http.Get("http://www.google.com")
	haveInternet = (err == nil)
}

func TestGetState_Consul(t *testing.T) {
	if !haveInternet {
		t.SkipNow()
	}

	// Use the Consul demo cluster
	conf := consulapi.DefaultConfig()
	conf.Address = "demo.consul.io:80"
	client, err := consulapi.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Write some test data
	pair := &consulapi.KVPair{
		Key:   "test/tf/remote/foobar",
		Value: []byte("testing"),
	}
	kv := client.KV()
	if _, err := kv.Put(pair, nil); err != nil {
		t.Fatalf("err: %v", err)
	}
	defer kv.Delete(pair.Key, nil)

	// Check we can get the state
	remote := &terraform.RemoteState{
		Name:   "foobar",
		Server: "http://demo.consul.io/v1/kv/test/tf/remote",
	}
REQ:
	payload, err := GetState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the MD5
	expect := md5.Sum(pair.Value)
	if !bytes.Equal(payload.MD5, expect[:md5.Size]) {
		t.Fatalf("Bad md5")
	}

	// Check the body
	var buf bytes.Buffer
	io.Copy(&buf, payload.R)
	if string(buf.Bytes()) != "testing" {
		t.Fatalf("Bad body")
	}

	// Try doing a ?raw lookup
	if !strings.Contains(remote.Server, "?raw") {
		remote.Server += "?raw"
		goto REQ
	}
}
