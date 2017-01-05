package net

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
)

const (
	JobFinished            = "finished"
	JobFailed              = "failed"
	DefaultPollingThrottle = 5 * time.Second
	DefaultDialTimeout     = 5 * time.Second
)

type JobResource struct {
	Entity struct {
		Status       string
		ErrorDetails struct {
			Description string
		} `json:"error_details"`
	}
}

type AsyncResource struct {
	Metadata struct {
		URL string
	}
}

type apiErrorHandler func(statusCode int, body []byte) error

type tokenRefresher interface {
	RefreshAuthToken() (string, error)
}

type Request struct {
	HTTPReq      *http.Request
	SeekableBody io.ReadSeeker
}

type Gateway struct {
	authenticator   tokenRefresher
	errHandler      apiErrorHandler
	PollingEnabled  bool
	PollingThrottle time.Duration
	trustedCerts    []tls.Certificate
	config          coreconfig.Reader
	warnings        *[]string
	Clock           func() time.Time
	transport       *http.Transport
	ui              terminal.UI
	logger          trace.Printer
	DialTimeout     time.Duration
}

func (gateway *Gateway) AsyncTimeout() time.Duration {
	if gateway.config.AsyncTimeout() > 0 {
		return time.Duration(gateway.config.AsyncTimeout()) * time.Minute
	}

	return 0
}

func (gateway *Gateway) SetTokenRefresher(auth tokenRefresher) {
	gateway.authenticator = auth
}

func (gateway Gateway) GetResource(url string, resource interface{}) (err error) {
	request, err := gateway.NewRequest("GET", url, gateway.config.AccessToken(), nil)
	if err != nil {
		return
	}

	_, err = gateway.PerformRequestForJSONResponse(request, resource)
	return
}

func (gateway Gateway) CreateResourceFromStruct(endpoint, url string, resource interface{}) error {
	data, err := json.Marshal(resource)
	if err != nil {
		return err
	}

	return gateway.CreateResource(endpoint, url, bytes.NewReader(data))
}

func (gateway Gateway) UpdateResourceFromStruct(endpoint, apiURL string, resource interface{}) error {
	data, err := json.Marshal(resource)
	if err != nil {
		return err
	}

	return gateway.UpdateResource(endpoint, apiURL, bytes.NewReader(data))
}

func (gateway Gateway) CreateResource(endpoint, apiURL string, body io.ReadSeeker, resource ...interface{}) error {
	return gateway.createUpdateOrDeleteResource("POST", endpoint, apiURL, body, false, resource...)
}

func (gateway Gateway) UpdateResource(endpoint, apiURL string, body io.ReadSeeker, resource ...interface{}) error {
	return gateway.createUpdateOrDeleteResource("PUT", endpoint, apiURL, body, false, resource...)
}

func (gateway Gateway) UpdateResourceSync(endpoint, apiURL string, body io.ReadSeeker, resource ...interface{}) error {
	return gateway.createUpdateOrDeleteResource("PUT", endpoint, apiURL, body, true, resource...)
}

func (gateway Gateway) DeleteResourceSynchronously(endpoint, apiURL string) error {
	return gateway.createUpdateOrDeleteResource("DELETE", endpoint, apiURL, nil, true, &AsyncResource{})
}

func (gateway Gateway) DeleteResource(endpoint, apiURL string) error {
	return gateway.createUpdateOrDeleteResource("DELETE", endpoint, apiURL, nil, false, &AsyncResource{})
}

func (gateway Gateway) ListPaginatedResources(
	target string,
	path string,
	resource interface{},
	cb func(interface{}) bool,
) error {
	for path != "" {
		pagination := NewPaginatedResources(resource)

		apiErr := gateway.GetResource(fmt.Sprintf("%s%s", target, path), &pagination)
		if apiErr != nil {
			return apiErr
		}

		resources, err := pagination.Resources()
		if err != nil {
			return fmt.Errorf("%s: %s", T("Error parsing JSON"), err.Error())
		}

		for _, resource := range resources {
			if !cb(resource) {
				return nil
			}
		}

		path = pagination.NextURL
	}

	return nil
}

func (gateway Gateway) createUpdateOrDeleteResource(verb, endpoint, apiURL string, body io.ReadSeeker, sync bool, optionalResource ...interface{}) error {
	var resource interface{}
	if len(optionalResource) > 0 {
		resource = optionalResource[0]
	}

	request, err := gateway.NewRequest(verb, endpoint+apiURL, gateway.config.AccessToken(), body)
	if err != nil {
		return err
	}

	if resource == nil {
		_, err = gateway.PerformRequest(request)
		return err
	}

	if gateway.PollingEnabled && !sync {
		_, err = gateway.PerformPollingRequestForJSONResponse(endpoint, request, resource, gateway.AsyncTimeout())
		return err
	}

	_, err = gateway.PerformRequestForJSONResponse(request, resource)
	if err != nil {
		return err
	}

	return nil
}

func (gateway Gateway) newRequest(request *http.Request, accessToken string, body io.ReadSeeker) *Request {
	if accessToken != "" {
		request.Header.Set("Authorization", accessToken)
	}

	request.Header.Set("accept", "application/json")
	request.Header.Set("Connection", "close")
	request.Header.Set("content-type", "application/json")
	request.Header.Set("User-Agent", "go-cli "+cf.Version+" / "+runtime.GOOS)

	return &Request{HTTPReq: request, SeekableBody: body}
}

func (gateway Gateway) NewRequestForFile(method, fullURL, accessToken string, body *os.File) (*Request, error) {
	progressReader := NewProgressReader(body, gateway.ui, 5*time.Second)
	_, _ = progressReader.Seek(0, 0)

	fileStats, err := body.Stat()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", T("Error getting file info"), err.Error())
	}

	request, err := http.NewRequest(method, fullURL, progressReader)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", T("Error building request"), err.Error())
	}

	fileSize := fileStats.Size()
	progressReader.SetTotalSize(fileSize)
	request.ContentLength = fileSize

	if err != nil {
		return nil, fmt.Errorf("%s: %s", T("Error building request"), err.Error())
	}

	return gateway.newRequest(request, accessToken, progressReader), nil
}

func (gateway Gateway) NewRequest(method, path, accessToken string, body io.ReadSeeker) (*Request, error) {
	request, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", T("Error building request"), err.Error())
	}
	return gateway.newRequest(request, accessToken, body), nil
}

func (gateway Gateway) PerformRequest(request *Request) (*http.Response, error) {
	return gateway.doRequestHandlingAuth(request)
}

func (gateway Gateway) performRequestForResponseBytes(request *Request) ([]byte, http.Header, *http.Response, error) {
	rawResponse, err := gateway.doRequestHandlingAuth(request)
	if err != nil {
		return nil, nil, rawResponse, err
	}
	defer rawResponse.Body.Close()

	bytes, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		return bytes, nil, rawResponse, fmt.Errorf("%s: %s", T("Error reading response"), err.Error())
	}

	return bytes, rawResponse.Header, rawResponse, nil
}

func (gateway Gateway) PerformRequestForTextResponse(request *Request) (string, http.Header, error) {
	bytes, headers, _, err := gateway.performRequestForResponseBytes(request)
	return string(bytes), headers, err
}

func (gateway Gateway) PerformRequestForJSONResponse(request *Request, response interface{}) (http.Header, error) {
	bytes, headers, rawResponse, err := gateway.performRequestForResponseBytes(request)
	if err != nil {
		if rawResponse != nil && rawResponse.Body != nil {
			b, _ := ioutil.ReadAll(rawResponse.Body)
			_ = json.Unmarshal(b, &response)
		}
		return headers, err
	}

	if rawResponse.StatusCode > 203 || strings.TrimSpace(string(bytes)) == "" {
		return headers, nil
	}

	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return headers, fmt.Errorf("%s: %s", T("Invalid JSON response from server"), err.Error())
	}

	return headers, nil
}

func (gateway Gateway) PerformPollingRequestForJSONResponse(endpoint string, request *Request, response interface{}, timeout time.Duration) (http.Header, error) {
	query := request.HTTPReq.URL.Query()
	query.Add("async", "true")
	request.HTTPReq.URL.RawQuery = query.Encode()

	bytes, headers, rawResponse, err := gateway.performRequestForResponseBytes(request)
	if err != nil {
		return headers, err
	}
	defer rawResponse.Body.Close()

	if rawResponse.StatusCode > 203 || strings.TrimSpace(string(bytes)) == "" {
		return headers, nil
	}

	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return headers, fmt.Errorf("%s: %s", T("Invalid JSON response from server"), err.Error())
	}

	asyncResource := &AsyncResource{}
	err = json.Unmarshal(bytes, &asyncResource)
	if err != nil {
		return headers, fmt.Errorf("%s: %s", T("Invalid async response from server"), err.Error())
	}

	jobURL := asyncResource.Metadata.URL
	if jobURL == "" {
		return headers, nil
	}

	if !strings.Contains(jobURL, "/jobs/") {
		return headers, nil
	}

	err = gateway.waitForJob(endpoint+jobURL, request.HTTPReq.Header.Get("Authorization"), timeout)

	return headers, err
}

func (gateway Gateway) Warnings() []string {
	return *gateway.warnings
}

func (gateway Gateway) waitForJob(jobURL, accessToken string, timeout time.Duration) error {
	startTime := gateway.Clock()
	for true {
		if gateway.Clock().Sub(startTime) > timeout && timeout != 0 {
			return errors.NewAsyncTimeoutError(jobURL)
		}
		var request *Request
		request, err := gateway.NewRequest("GET", jobURL, accessToken, nil)
		response := &JobResource{}
		_, err = gateway.PerformRequestForJSONResponse(request, response)
		if err != nil {
			return err
		}

		switch response.Entity.Status {
		case JobFinished:
			return nil
		case JobFailed:
			return errors.New(response.Entity.ErrorDetails.Description)
		}

		accessToken = request.HTTPReq.Header.Get("Authorization")

		time.Sleep(gateway.PollingThrottle)
	}
	return nil
}

func (gateway Gateway) doRequestHandlingAuth(request *Request) (*http.Response, error) {
	httpReq := request.HTTPReq

	if request.SeekableBody != nil {
		httpReq.Body = ioutil.NopCloser(request.SeekableBody)
	}

	// perform request
	rawResponse, err := gateway.doRequestAndHandlerError(request)
	if err == nil || gateway.authenticator == nil {
		return rawResponse, err
	}

	switch err.(type) {
	case *errors.InvalidTokenError:
		// refresh the auth token
		var newToken string
		newToken, err = gateway.authenticator.RefreshAuthToken()
		if err != nil {
			return rawResponse, err
		}

		// reset the auth token and request body
		httpReq.Header.Set("Authorization", newToken)
		if request.SeekableBody != nil {
			_, _ = request.SeekableBody.Seek(0, 0)
			httpReq.Body = ioutil.NopCloser(request.SeekableBody)
		}

		// make the request again
		rawResponse, err = gateway.doRequestAndHandlerError(request)
	}

	return rawResponse, err
}

func (gateway Gateway) doRequestAndHandlerError(request *Request) (*http.Response, error) {
	rawResponse, err := gateway.doRequest(request.HTTPReq)
	if err != nil {
		return rawResponse, WrapNetworkErrors(request.HTTPReq.URL.Host, err)
	}

	if rawResponse.StatusCode > 299 {
		defer rawResponse.Body.Close()
		jsonBytes, _ := ioutil.ReadAll(rawResponse.Body)
		rawResponse.Body = ioutil.NopCloser(bytes.NewBuffer(jsonBytes))
		err = gateway.errHandler(rawResponse.StatusCode, jsonBytes)
	}

	return rawResponse, err
}

func (gateway Gateway) doRequest(request *http.Request) (*http.Response, error) {
	var response *http.Response
	var err error

	if gateway.transport == nil {
		makeHTTPTransport(&gateway)
	}

	httpClient := NewHTTPClient(gateway.transport, NewRequestDumper(gateway.logger))

	httpClient.DumpRequest(request)

	for i := 0; i < 3; i++ {
		response, err = httpClient.Do(request)
		if response == nil && err != nil {
			continue
		} else {
			break
		}
	}

	if err != nil {
		return response, err
	}

	httpClient.DumpResponse(response)

	header := http.CanonicalHeaderKey("X-Cf-Warnings")
	rawWarnings := response.Header[header]
	for _, rawWarning := range rawWarnings {
		warning, _ := url.QueryUnescape(rawWarning)
		*gateway.warnings = append(*gateway.warnings, warning)
	}

	return response, err
}

func makeHTTPTransport(gateway *Gateway) {
	gateway.transport = &http.Transport{
		Dial:            (&net.Dialer{Timeout: gateway.DialTimeout}).Dial,
		TLSClientConfig: NewTLSConfig(gateway.trustedCerts, gateway.config.IsSSLDisabled()),
		Proxy:           http.ProxyFromEnvironment,
	}
}

func dialTimeout(envDialTimeout string) time.Duration {
	dialTimeout := DefaultDialTimeout
	if timeout, err := strconv.Atoi(envDialTimeout); err == nil {
		dialTimeout = time.Duration(timeout) * time.Second
	}
	return dialTimeout
}

func (gateway *Gateway) SetTrustedCerts(certificates []tls.Certificate) {
	gateway.trustedCerts = certificates
	makeHTTPTransport(gateway)
}
