// Package oauthserver is a very simplistic OAuth server used only for
// the testing of the "terraform login" and "terraform logout" commands.
package oauthserver

import (
	"log"
	"net/http"
)

// Handler is an implementation of net/http.Handler that provides a stub
// OAuth server implementation with the following endpoints:
//
//     /authz  - authorization endpoint
//     /token  - token endpoint
//     /revoke - token revocation (logout) endpoint
//
// The authorization endpoint returns HTML per normal OAuth conventions, but
// it also includes an HTTP header X-Redirect-To giving the same URL that the
// link in the HTML indicates, allowing a non-browser user-agent to traverse
// this robotically in automated tests.
var Handler http.Handler

type handler struct{}

func (h handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/authz":
		h.serveAuthz(resp, req)
	case "/token":
		h.serveToken(resp, req)
	case "/revoke":
		h.serveRevoke(resp, req)
	default:
		resp.WriteHeader(404)
	}
}

func (h handler) serveAuthz(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(404)
}

func (h handler) serveToken(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		resp.WriteHeader(405)
		log.Printf("/token: unsupported request method %q", req.Method)
		return
	}

	if err := req.ParseForm(); err != nil {
		resp.WriteHeader(500)
		log.Printf("/token: error parsing body: %s", err)
		return
	}

	grantType := req.Form.Get("grant_type")
	log.Printf("/token: grant_type is %q", grantType)
	switch grantType {
	case "password":
		username := req.Form.Get("username")
		password := req.Form.Get("password")

		if username == "wrong" || password == "wrong" {
			// These special "credentials" allow testing for the error case.
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(400)
			resp.Write([]byte(`{"error":"invalid_grant"}`))
			log.Println("/token: 'wrong' credentials")
			return
		}

		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(200)
		resp.Write([]byte(`{"access_token":"good-token","token_type":"bearer"}`))
		log.Println("/token: successful request")

	default:
		resp.WriteHeader(400)
		log.Printf("/token: unsupported grant type %q", grantType)
	}
}

func (h handler) serveRevoke(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(404)
}

func init() {
	Handler = handler{}
}
