package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
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
	r := &remoteStateClient{conf: remote}
	payload, err := r.GetState()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the MD5
	expect := md5.Sum(pair.Value)
	if !bytes.Equal(payload.MD5, expect[:md5.Size]) {
		t.Fatalf("Bad md5")
	}

	// Check the body
	if string(payload.State) != "testing" {
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

		r := &remoteStateClient{remote}
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

func TestPutState(t *testing.T) {
	type tcase struct {
		Code      int
		Path      string
		Header    http.Header
		Body      []byte
		ExpectMD5 []byte
		Force     bool
		ExpectErr string
	}
	inp := []byte("testing")
	inpMD5 := md5.Sum(inp)
	hash := inpMD5[:16]
	cases := []*tcase{
		&tcase{
			Code:      http.StatusOK,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
		},
		&tcase{
			Code:      http.StatusOK,
			Path:      "/foobar?force=true",
			Body:      inp,
			Force:     true,
			ExpectMD5: hash,
		},
		&tcase{
			Code:      http.StatusConflict,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
			ExpectErr: ErrConflict.Error(),
		},
		&tcase{
			Code:      http.StatusPreconditionFailed,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
			ExpectErr: ErrServerNewer.Error(),
		},
		&tcase{
			Code:      http.StatusUnauthorized,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
			ExpectErr: ErrRequireAuth.Error(),
		},
		&tcase{
			Code:      http.StatusForbidden,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
			ExpectErr: ErrInvalidAuth.Error(),
		},
		&tcase{
			Code:      http.StatusInternalServerError,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
			ExpectErr: ErrRemoteInternal.Error(),
		},
		&tcase{
			Code:      418,
			Path:      "/foobar",
			Body:      inp,
			ExpectMD5: hash,
			ExpectErr: "Unexpected HTTP response code 418",
		},
	}

	for _, tc := range cases {
		cb := func(resp http.ResponseWriter, req *http.Request) {
			for k, v := range tc.Header {
				resp.Header()[k] = v
			}
			resp.WriteHeader(tc.Code)

			// Verify the body
			buf := bytes.NewBuffer(nil)
			io.Copy(buf, req.Body)
			if !bytes.Equal(buf.Bytes(), tc.Body) {
				t.Fatalf("bad body: %v", buf.Bytes())
			}

			// Verify the path
			req.URL.Host = ""
			if req.URL.String() != tc.Path {
				t.Fatalf("Bad path: %v %v", req.URL.String(), tc.Path)
			}

			// Verify the content length
			if req.ContentLength != int64(len(tc.Body)) {
				t.Fatalf("bad content length: %d", req.ContentLength)
			}

			// Verify the Content-MD5
			b64 := req.Header.Get("Content-MD5")
			raw, _ := base64.StdEncoding.DecodeString(b64)
			if !bytes.Equal(raw, tc.ExpectMD5) {
				t.Fatalf("bad md5: %v", raw)
			}
		}
		s := httptest.NewServer(http.HandlerFunc(cb))
		defer s.Close()

		remote := &terraform.RemoteState{
			Name:   "foobar",
			Server: s.URL,
		}

		r := &remoteStateClient{remote}
		err := r.PutState(tc.Body, tc.Force)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.ExpectErr {
			t.Fatalf("bad err: %v %v", errStr, tc.ExpectErr)
		}
	}
}
