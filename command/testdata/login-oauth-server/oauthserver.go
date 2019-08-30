// Package oauthserver is a very simplistic OAuth server used only for
// the testing of the "terraform login" and "terraform logout" commands.
package oauthserver

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strings"
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
	args := req.URL.Query()
	if rt := args.Get("response_type"); rt != "code" {
		resp.WriteHeader(400)
		resp.Write([]byte("wrong response_type"))
		log.Printf("/authz: incorrect response type %q", rt)
		return
	}
	redirectURL, err := url.Parse(args.Get("redirect_uri"))
	if err != nil {
		resp.WriteHeader(400)
		resp.Write([]byte(fmt.Sprintf("invalid redirect_uri %s: %s", args.Get("redirect_uri"), err)))
		return
	}

	state := args.Get("state")
	challenge := args.Get("code_challenge")
	challengeMethod := args.Get("code_challenge_method")
	if challengeMethod == "" {
		challengeMethod = "plain"
	}

	// NOTE: This is not a suitable implementation for a real OAuth server
	// because the code challenge is providing no security whatsoever. This
	// is just a simple implementation for this stub server.
	code := fmt.Sprintf("%s:%s", challengeMethod, challenge)

	redirectQuery := redirectURL.Query()
	redirectQuery.Set("code", code)
	if state != "" {
		redirectQuery.Set("state", state)
	}
	redirectURL.RawQuery = redirectQuery.Encode()

	respBody := fmt.Sprintf(`<a href="%s">Log In and Consent</a>`, html.EscapeString(redirectURL.String()))
	resp.Header().Set("Content-Type", "text/html")
	resp.Header().Set("Content-Length", fmt.Sprintf("%d", len(respBody)))
	resp.Header().Set("X-Redirect-To", redirectURL.String()) // For robotic clients, using webbrowser.MockLauncher
	resp.WriteHeader(200)
	resp.Write([]byte(respBody))
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

	case "authorization_code":
		code := req.Form.Get("code")
		codeParts := strings.SplitN(code, ":", 2)
		if len(codeParts) != 2 {
			log.Printf("/token: invalid code %q", code)
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(400)
			resp.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}

		codeVerifier := req.Form.Get("code_verifier")

		switch codeParts[0] {
		case "plain":
			if codeParts[1] != codeVerifier {
				log.Printf("/token: incorrect code verifier %q; want %q", codeParts[1], codeVerifier)
				resp.Header().Set("Content-Type", "application/json")
				resp.WriteHeader(400)
				resp.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
		case "S256":
			h := sha256.New()
			h.Write([]byte(codeVerifier))
			encVerifier := base64.URLEncoding.EncodeToString(h.Sum(nil))
			if codeParts[1] != encVerifier {
				log.Printf("/token: incorrect code verifier %q; want %q", codeParts[1], encVerifier)
				resp.Header().Set("Content-Type", "application/json")
				resp.WriteHeader(400)
				resp.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
		default:
			log.Printf("/token: unsupported challenge method %q", codeParts[0])
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(400)
			resp.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}

		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(200)
		resp.Write([]byte(`{"access_token":"good-token","token_type":"bearer"}`))
		log.Println("/token: successful request")

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
