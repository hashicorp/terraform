//
// gocommon - Go library to interact with the JoyentCloud
//
//
// Copyright (c) 2013 Joyent Inc.
//
// Written by Daniele Stroppa <daniele.stroppa@joyent.com>
//

package client

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	joyenthttp "github.com/joyent/gocommon/http"
	"github.com/joyent/gosign/auth"
)

const (
	// The HTTP request methods.
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
	HEAD   = "HEAD"
	COPY   = "COPY"
)

// Client implementations sends service requests to the JoyentCloud.
type Client interface {
	SendRequest(method, apiCall, rfc1123Date string, request *joyenthttp.RequestData, response *joyenthttp.ResponseData) (err error)
	// MakeServiceURL prepares a full URL to a service endpoint, with optional
	// URL parts. It uses the first endpoint it can find for the given service type.
	MakeServiceURL(parts []string) string
	SignURL(path string, expires time.Time) (string, error)
}

// This client sends requests without authenticating.
type client struct {
	mu         sync.Mutex
	logger     *log.Logger
	baseURL    string
	creds      *auth.Credentials
	httpClient *joyenthttp.Client
}

var _ Client = (*client)(nil)

func newClient(baseURL string, credentials *auth.Credentials, httpClient *joyenthttp.Client, logger *log.Logger) Client {
	client := client{baseURL: baseURL, logger: logger, creds: credentials, httpClient: httpClient}
	return &client
}

func NewClient(baseURL, apiVersion string, credentials *auth.Credentials, logger *log.Logger) Client {
	sharedHttpClient := joyenthttp.New(credentials, apiVersion, logger)
	return newClient(baseURL, credentials, sharedHttpClient, logger)
}

func (c *client) sendRequest(method, url, rfc1123Date string, request *joyenthttp.RequestData, response *joyenthttp.ResponseData) (err error) {
	if request.ReqValue != nil || response.RespValue != nil {
		err = c.httpClient.JsonRequest(method, url, rfc1123Date, request, response)
	} else {
		err = c.httpClient.BinaryRequest(method, url, rfc1123Date, request, response)
	}
	return
}

func (c *client) SendRequest(method, apiCall, rfc1123Date string, request *joyenthttp.RequestData, response *joyenthttp.ResponseData) (err error) {
	url := c.MakeServiceURL([]string{c.creds.UserAuthentication.User, apiCall})
	err = c.sendRequest(method, url, rfc1123Date, request, response)
	return
}

func makeURL(base string, parts []string) string {
	if !strings.HasSuffix(base, "/") && len(parts) > 0 {
		base += "/"
	}
	if parts[1] == "" {
		return base + parts[0]
	}
	return base + strings.Join(parts, "/")
}

func (c *client) MakeServiceURL(parts []string) string {
	return makeURL(c.baseURL, parts)
}

func (c *client) SignURL(path string, expires time.Time) (string, error) {
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("bad Manta endpoint URL %q: %v", c.baseURL, err)
	}
	userAuthentication := c.creds.UserAuthentication
	userAuthentication.Algorithm = "RSA-SHA1"
	keyId := url.QueryEscape(fmt.Sprintf("/%s/keys/%s", userAuthentication.User, c.creds.MantaKeyId))
	params := fmt.Sprintf("algorithm=%s&expires=%d&keyId=%s", userAuthentication.Algorithm, expires.Unix(), keyId)
	signingLine := fmt.Sprintf("GET\n%s\n%s\n%s", parsedURL.Host, path, params)

	signature, err := auth.GetSignature(userAuthentication, signingLine)
	if err != nil {
		return "", fmt.Errorf("cannot generate URL signature: %v", err)
	}
	signedURL := fmt.Sprintf("%s%s?%s&signature=%s", c.baseURL, path, params, url.QueryEscape(signature))
	return signedURL, nil
}
