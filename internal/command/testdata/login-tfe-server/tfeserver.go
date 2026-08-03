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
	MOTD           = `{"msg":"Welcome to HCP Terraform!"}`

	orgsWithMaxTTL    = `{"data":[{"id":"org-abc123","type":"organizations","attributes":{"name":"ttl-org","max-ttl-enabled":true}}],"meta":{"pagination":{"current-page":1,"total-pages":1,"total-count":1}}}`
	orgsWithoutMaxTTL = `{"data":[{"id":"org-abc123","type":"organizations","attributes":{"name":"no-ttl-org","max-ttl-enabled":false}}],"meta":{"pagination":{"current-page":1,"total-pages":1,"total-count":1}}}`
)

// Handler is an implementation of net/http.Handler that provides a stub
// TFE API server implementation with the following endpoints:
//
//	/ping            - API existence endpoint
//	/account/details - current user endpoint
//	/organizations   - organizations list endpoint
var Handler http.Handler

// HandlerWithMaxTTL is like Handler but returns organizations with MaxTTLEnabled set to true.
var HandlerWithMaxTTL http.Handler

type handler struct {
	maxTTLEnabled bool
}

func (h handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/vnd.api+json")
	switch req.URL.Path {
	case "/api/v2/ping":
		h.servePing(resp, req)
	case "/api/v2/account/details":
		h.serveAccountDetails(resp, req)
	case "/api/v2/organizations":
		h.serveOrganizations(resp, req)
	case "/api/terraform/motd":
		h.serveMOTD(resp, req)
	default:
		fmt.Printf("404 when fetching %s\n", req.URL.String())
		http.Error(resp, `{"errors":[{"status":"404","title":"not found"}]}`, http.StatusNotFound)
	}
}

func (h handler) servePing(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusNoContent)
}

func (h handler) serveAccountDetails(resp http.ResponseWriter, req *http.Request) {
	if !strings.Contains(req.Header.Get("Authorization"), goodToken) {
		http.Error(resp, `{"errors":[{"status":"401","title":"unauthorized"}]}`, http.StatusUnauthorized)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(accountDetails))
}

func (h handler) serveOrganizations(resp http.ResponseWriter, req *http.Request) {
	if !strings.Contains(req.Header.Get("Authorization"), goodToken) {
		http.Error(resp, `{"errors":[{"status":"401","title":"unauthorized"}]}`, http.StatusUnauthorized)
		return
	}
	if h.maxTTLEnabled {
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte(orgsWithMaxTTL))
	} else {
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte(orgsWithoutMaxTTL))
	}
}

func (h handler) serveMOTD(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(MOTD))
}

func init() {
	Handler = handler{maxTTLEnabled: false}
	HandlerWithMaxTTL = handler{maxTTLEnabled: true}
}
