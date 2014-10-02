package remote

import (
	"bytes"
	"crypto/md5"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestGetState(t *testing.T) {
	type tcase struct {
		Code      int
		Header    http.Header
		Body      []byte
		ExpectMD5 []byte
		ExpectErr string
	}
	inp := []byte("testing")
	inpMD5 := md5.Sum(inp)
	hash := inpMD5[:16]
	cases := []*tcase{
		&tcase{
			Code:      http.StatusOK,
			Body:      inp,
			ExpectMD5: hash,
		},
		&tcase{
			Code: http.StatusNoContent,
		},
		&tcase{
			Code: http.StatusNotFound,
		},
		&tcase{
			Code:      http.StatusUnauthorized,
			ExpectErr: "Remote server requires authentication",
		},
		&tcase{
			Code:      http.StatusForbidden,
			ExpectErr: "Invalid authentication",
		},
		&tcase{
			Code:      http.StatusInternalServerError,
			ExpectErr: "Remote server reporting internal error",
		},
		&tcase{
			Code:      418,
			ExpectErr: "Unexpected HTTP response code 418",
		},
	}

	for _, tc := range cases {
		cb := func(resp http.ResponseWriter, req *http.Request) {
			for k, v := range tc.Header {
				resp.Header()[k] = v
			}
			resp.WriteHeader(tc.Code)
			if tc.Body != nil {
				resp.Write(tc.Body)
			}
		}
		s := httptest.NewServer(http.HandlerFunc(cb))
		defer s.Close()

		remote := &terraform.RemoteState{
			Name:   "foobar",
			Server: s.URL,
		}

		payload, err := GetState(remote)
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
			buf := bytes.NewBuffer(nil)
			io.Copy(buf, payload.R)
			if !bytes.Equal(buf.Bytes(), tc.Body) {
				t.Fatalf("bad: %#v", payload)
			}
		}
	}
}
