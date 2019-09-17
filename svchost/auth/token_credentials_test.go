package auth

import (
	"net/http"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestHostCredentialsToken(t *testing.T) {
	creds := HostCredentialsToken("foo-bar")

	{
		req := &http.Request{}
		creds.PrepareRequest(req)
		authStr := req.Header.Get("authorization")
		if got, want := authStr, "Bearer foo-bar"; got != want {
			t.Errorf("wrong Authorization header value %q; want %q", got, want)
		}
	}

	{
		got := creds.ToStore()
		want := cty.ObjectVal(map[string]cty.Value{
			"token": cty.StringVal("foo-bar"),
		})
		if !want.RawEquals(got) {
			t.Errorf("wrong storable object value\ngot:  %#v\nwant: %#v", got, want)
		}
	}
}
