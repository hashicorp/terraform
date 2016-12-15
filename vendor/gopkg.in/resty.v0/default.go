// Copyright (c) 2015-2016 Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package resty

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

// DefaultClient of resty
var DefaultClient *Client

// New method creates a new go-resty client
func New() *Client {
	cookieJar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	c := &Client{
		HostURL:    "",
		QueryParam: url.Values{},
		FormData:   url.Values{},
		Header:     http.Header{},
		UserInfo:   nil,
		Token:      "",
		Cookies:    make([]*http.Cookie, 0),
		Debug:      false,
		Log:        getLogger(os.Stderr),
		httpClient: &http.Client{Jar: cookieJar},
		transport:  &http.Transport{},
		mutex:      &sync.Mutex{},
		RetryCount: 0,
	}

	// Default redirect policy
	c.SetRedirectPolicy(NoRedirectPolicy())

	// default before request middlewares
	c.beforeRequest = []func(*Client, *Request) error{
		parseRequestURL,
		parseRequestHeader,
		parseRequestBody,
		createHTTPRequest,
		addCredentials,
		requestLogger,
	}

	// default after response middlewares
	c.afterResponse = []func(*Client, *Response) error{
		responseLogger,
		parseResponseBody,
		saveResponseIntoFile,
	}

	return c
}

// R creates a new resty request object, it is used form a HTTP/RESTful request
// such as GET, POST, PUT, DELETE, HEAD, PATCH and OPTIONS.
func R() *Request {
	return DefaultClient.R()
}

// SetHostURL sets Host URL. See `Client.SetHostURL for more information.
func SetHostURL(url string) *Client {
	return DefaultClient.SetHostURL(url)
}

// SetHeader sets single header. See `Client.SetHeader` for more information.
func SetHeader(header, value string) *Client {
	return DefaultClient.SetHeader(header, value)
}

// SetHeaders sets multiple headers. See `Client.SetHeaders` for more information.
func SetHeaders(headers map[string]string) *Client {
	return DefaultClient.SetHeaders(headers)
}

// SetCookie sets single cookie object. See `Client.SetCookie` for more information.
func SetCookie(hc *http.Cookie) *Client {
	return DefaultClient.SetCookie(hc)
}

// SetCookies sets multiple cookie object. See `Client.SetCookies` for more information.
func SetCookies(cs []*http.Cookie) *Client {
	return DefaultClient.SetCookies(cs)
}

// SetQueryParam method sets single paramater and its value. See `Client.SetQueryParam` for more information.
func SetQueryParam(param, value string) *Client {
	return DefaultClient.SetQueryParam(param, value)
}

// SetQueryParams method sets multiple paramaters and its value. See `Client.SetQueryParams` for more information.
func SetQueryParams(params map[string]string) *Client {
	return DefaultClient.SetQueryParams(params)
}

// SetFormData method sets Form parameters and its values. See `Client.SetFormData` for more information.
func SetFormData(data map[string]string) *Client {
	return DefaultClient.SetFormData(data)
}

// SetBasicAuth method sets the basic authentication header. See `Client.SetBasicAuth` for more information.
func SetBasicAuth(username, password string) *Client {
	return DefaultClient.SetBasicAuth(username, password)
}

// SetAuthToken method sets bearer auth token header. See `Client.SetAuthToken` for more information.
func SetAuthToken(token string) *Client {
	return DefaultClient.SetAuthToken(token)
}

// OnBeforeRequest method sets request middleware. See `Client.OnBeforeRequest` for more information.
func OnBeforeRequest(m func(*Client, *Request) error) *Client {
	return DefaultClient.OnBeforeRequest(m)
}

// OnAfterResponse method sets response middleware. See `Client.OnAfterResponse` for more information.
func OnAfterResponse(m func(*Client, *Response) error) *Client {
	return DefaultClient.OnAfterResponse(m)
}

// SetDebug method enables the debug mode. See `Client.SetDebug` for more information.
func SetDebug(d bool) *Client {
	return DefaultClient.SetDebug(d)
}

// SetRetryCount method set the retry count. See `Client.SetRetryCount` for more information.
func SetRetryCount(count int) *Client {
	return DefaultClient.SetRetryCount(count)
}

// AddRetryCondition method appends check function for retry. See `Client.AddRetryCondition` for more information.
func AddRetryCondition(condition RetryConditionFunc) *Client {
	return DefaultClient.AddRetryCondition(condition)
}

// SetDisableWarn method disables warning comes from `go-resty` client. See `Client.SetDisableWarn` for more information.
func SetDisableWarn(d bool) *Client {
	return DefaultClient.SetDisableWarn(d)
}

// SetLogger method sets given writer for logging. See `Client.SetLogger` for more information.
func SetLogger(w io.Writer) *Client {
	return DefaultClient.SetLogger(w)
}

// SetContentLength method enables `Content-Length` value. See `Client.SetContentLength` for more information.
func SetContentLength(l bool) *Client {
	return DefaultClient.SetContentLength(l)
}

// SetError method is to register the global or client common `Error` object. See `Client.SetError` for more information.
func SetError(err interface{}) *Client {
	return DefaultClient.SetError(err)
}

// SetRedirectPolicy method sets the client redirect poilicy. See `Client.SetRedirectPolicy` for more information.
func SetRedirectPolicy(policies ...interface{}) *Client {
	return DefaultClient.SetRedirectPolicy(policies...)
}

// SetHTTPMode method sets go-resty mode into HTTP. See `Client.SetMode` for more information.
func SetHTTPMode() *Client {
	return DefaultClient.SetHTTPMode()
}

// SetRESTMode method sets go-resty mode into RESTful. See `Client.SetMode` for more information.
func SetRESTMode() *Client {
	return DefaultClient.SetRESTMode()
}

// Mode method returns the current client mode. See `Client.Mode` for more information.
func Mode() string {
	return DefaultClient.Mode()
}

// SetTLSClientConfig method sets TLSClientConfig for underling client Transport. See `Client.SetTLSClientConfig` for more information.
func SetTLSClientConfig(config *tls.Config) *Client {
	return DefaultClient.SetTLSClientConfig(config)
}

// SetTimeout method sets timeout for request. See `Client.SetTimeout` for more information.
func SetTimeout(timeout time.Duration) *Client {
	return DefaultClient.SetTimeout(timeout)
}

// SetProxy method sets Proxy for request. See `Client.SetProxy` for more information.
func SetProxy(proxyURL string) *Client {
	return DefaultClient.SetProxy(proxyURL)
}

// RemoveProxy method removes the proxy configuration. See `Client.RemoveProxy` for more information.
func RemoveProxy() *Client {
	return DefaultClient.RemoveProxy()
}

// SetCertificates method helps to set client certificates into resty conveniently.
// See `Client.SetCertificates` for more information and example.
func SetCertificates(certs ...tls.Certificate) *Client {
	return DefaultClient.SetCertificates(certs...)
}

// SetRootCertificate method helps to add one or more root certificates into resty client.
// See `Client.SetRootCertificate` for more information.
func SetRootCertificate(pemFilePath string) *Client {
	return DefaultClient.SetRootCertificate(pemFilePath)
}

// SetOutputDirectory method sets output directory. See `Client.SetOutputDirectory` for more information.
func SetOutputDirectory(dirPath string) *Client {
	return DefaultClient.SetOutputDirectory(dirPath)
}

// SetTransport method sets custom *http.Transport in the resty client.
// See `Client.SetTransport` for more information.
func SetTransport(transport *http.Transport) *Client {
	return DefaultClient.SetTransport(transport)
}

// SetScheme method sets custom scheme in the resty client.
// See `Client.SetScheme` for more information.
func SetScheme(scheme string) *Client {
	return DefaultClient.SetScheme(scheme)
}

// SetCloseConnection method sets close connection value in the resty client.
// See `Client.SetCloseConnection` for more information.
func SetCloseConnection(close bool) *Client {
	return DefaultClient.SetCloseConnection(close)
}

func init() {
	DefaultClient = New()
}
