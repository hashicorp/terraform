package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestHTTPRemote_Interface(t *testing.T) {
	var client interface{} = &HTTPRemoteClient{}
	if _, ok := client.(RemoteClient); !ok {
		t.Fatalf("does not implement interface")
	}
}

func TestHTTPRemote_Validate(t *testing.T) {
	conf := map[string]string{}
	if _, err := NewHTTPRemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	conf["address"] = ""
	if _, err := NewHTTPRemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	conf["address"] = "*"
	if _, err := NewHTTPRemoteClient(conf); err == nil {
		t.Fatalf("expect error")
	}

	conf["address"] = "http://cool.com"
	if _, err := NewHTTPRemoteClient(conf); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestHTTPRemote_GetState(t *testing.T) {
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
			Type: "http",
			Config: map[string]string{
				"address": s.URL,
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

func TestHTTPRemote_PutState(t *testing.T) {
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
			Type: "http",
			Config: map[string]string{
				"address": s.URL + "/foobar",
			},
		}
		r, err := NewClientByState(remote)
		if err != nil {
			t.Fatalf("Err: %v", err)
		}

		err = r.PutState(tc.Body, tc.Force)
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.ExpectErr {
			t.Fatalf("bad err: %v %v", errStr, tc.ExpectErr)
		}
	}
}

func TestHTTPRemote_DeleteState(t *testing.T) {
	type tcase struct {
		Code      int
		Path      string
		Header    http.Header
		ExpectErr string
	}
	cases := []*tcase{
		&tcase{
			Code: http.StatusOK,
			Path: "/foobar",
		},
		&tcase{
			Code: http.StatusNoContent,
			Path: "/foobar",
		},
		&tcase{
			Code: http.StatusNotFound,
			Path: "/foobar",
		},
		&tcase{
			Code:      http.StatusUnauthorized,
			Path:      "/foobar",
			ExpectErr: ErrRequireAuth.Error(),
		},
		&tcase{
			Code:      http.StatusForbidden,
			Path:      "/foobar",
			ExpectErr: ErrInvalidAuth.Error(),
		},
		&tcase{
			Code:      http.StatusInternalServerError,
			Path:      "/foobar",
			ExpectErr: ErrRemoteInternal.Error(),
		},
		&tcase{
			Code:      418,
			Path:      "/foobar",
			ExpectErr: "Unexpected HTTP response code 418",
		},
	}

	for _, tc := range cases {
		cb := func(resp http.ResponseWriter, req *http.Request) {
			for k, v := range tc.Header {
				resp.Header()[k] = v
			}
			resp.WriteHeader(tc.Code)

			// Verify the path
			req.URL.Host = ""
			if req.URL.String() != tc.Path {
				t.Fatalf("Bad path: %v %v", req.URL.String(), tc.Path)
			}
		}
		s := httptest.NewServer(http.HandlerFunc(cb))
		defer s.Close()

		remote := &terraform.RemoteState{
			Type: "http",
			Config: map[string]string{
				"address": s.URL + "/foobar",
			},
		}
		r, err := NewClientByState(remote)
		if err != nil {
			t.Fatalf("Err: %v", err)
		}

		err = r.DeleteState()
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if errStr != tc.ExpectErr {
			t.Fatalf("bad err: %v %v", errStr, tc.ExpectErr)
		}
	}
}
