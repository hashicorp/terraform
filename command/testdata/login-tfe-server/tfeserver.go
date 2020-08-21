// Package tfeserver is a test stub implementing a subset of the TFE API used
// only for the testing of the "terraform login" command.
package tfeserver

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	goodToken      = "good-token"
	accountDetails = `{"data":{"id":"user-abc123","type":"users","attributes":{"username":"testuser","email":"testuser@example.com"}}}`
	entitlements   = `{"data":{"id":"hashicorp","type":"entitlement-sets","attributes":{"operations":true}}}`
	workspace      = `{"data":{"id":"ws-abc123","type":"workspaces","attributes":{"name":"foo"}}}`
)

// Handler is an implementation of net/http.Handler that provides a stub
// TFE API server implementation with the following endpoints for login:
//
//     /ping            - API existence endpoint
//     /account/details - current user endpoint
//
// It also includes endpoints for initializing a remote backend:
//
//     /organizations/hashicorp/entitlement-set
//     /organizations/hashicorp/workspaces/foo
var Handler http.Handler

type handler struct{}

func (h handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/vnd.api+json")
	switch req.URL.Path {
	case "/api/v2/ping":
		h.servePing(resp, req)
	case "/api/v2/account/details":
		h.serveIfAuthenticated(resp, req, accountDetails)
	case "/api/v2/organizations/hashicorp/entitlement-set":
		h.serveIfAuthenticated(resp, req, entitlements)
	case "/api/v2/organizations/hashicorp/workspaces/foo":
		h.serveIfAuthenticated(resp, req, workspace)
	default:
		fmt.Printf("404 when fetching %s\n", req.URL.String())
		http.Error(resp, `{"errors":[{"status":"404","title":"not found"}]}`, http.StatusNotFound)
	}
}

func (h handler) servePing(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNoContent)
}

func (h handler) serveIfAuthenticated(resp http.ResponseWriter, req *http.Request, body string) {
	if !strings.Contains(req.Header.Get("Authorization"), goodToken) {
		http.Error(resp, `{"errors":[{"status":"401","title":"unauthorized"}]}`, http.StatusUnauthorized)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(body))
}

func init() {
	Handler = handler{}
}
