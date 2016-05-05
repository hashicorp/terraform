//
// gocommon - Go library to interact with the JoyentCloud
// An HTTP Client which sends json and binary requests, handling data marshalling and response processing.
//
// Copyright (c) 2013 Joyent Inc.
//
// Written by Daniele Stroppa <daniele.stroppa@joyent.com>
//

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joyent/gocommon"
	"github.com/joyent/gocommon/errors"
	"github.com/joyent/gocommon/jpc"
	"github.com/joyent/gosign/auth"
)

const (
	contentTypeJSON        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
)

type Client struct {
	http.Client
	maxSendAttempts int
	credentials     *auth.Credentials
	apiVersion      string
	logger          *log.Logger
	trace           bool
}

type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("Failed: %d: %s", e.Code, e.Message)
}

type ErrorWrapper struct {
	Error ErrorResponse `json:"error"`
}

type RequestData struct {
	ReqHeaders http.Header
	Params     *url.Values
	ReqValue   interface{}
	ReqReader  io.Reader
	ReqLength  int
}

type ResponseData struct {
	ExpectedStatus []int
	RespHeaders    *http.Header
	RespValue      interface{}
	RespReader     io.ReadCloser
}

const (
	// The maximum number of times to try sending a request before we give up
	// (assuming any unsuccessful attempts can be sensibly tried again).
	MaxSendAttempts = 3
)

// New returns a new http *Client using the default net/http client.
func New(credentials *auth.Credentials, apiVersion string, logger *log.Logger) *Client {
	return &Client{*http.DefaultClient, MaxSendAttempts, credentials, apiVersion, logger, false}
}

// SetTrace allows control over whether requests will write their
// contents to the logger supplied during construction. Note that this
// is not safe to call from multiple go-routines.
func (client *Client) SetTrace(traceEnabled bool) {
	client.trace = traceEnabled
}

func gojoyentAgent() string {
	return fmt.Sprintf("gocommon (%s)", gocommon.Version)
}

func createHeaders(extraHeaders http.Header, credentials *auth.Credentials, contentType, rfc1123Date,
	apiVersion string, isMantaRequest bool) (http.Header, error) {

	headers := make(http.Header)
	if extraHeaders != nil {
		for header, values := range extraHeaders {
			for _, value := range values {
				headers.Add(header, value)
			}
		}
	}
	if extraHeaders.Get("Content-Type") == "" {
		headers.Add("Content-Type", contentType)
	}
	if extraHeaders.Get("Accept") == "" {
		headers.Add("Accept", contentType)
	}
	if rfc1123Date != "" {
		headers.Set("Date", rfc1123Date)
	} else {
		headers.Set("Date", getDateForRegion(credentials, isMantaRequest))
	}
	authHeaders, err := auth.CreateAuthorizationHeader(headers, credentials, isMantaRequest)
	if err != nil {
		return http.Header{}, err
	}
	headers.Set("Authorization", authHeaders)
	if apiVersion != "" {
		headers.Set("X-Api-Version", apiVersion)
	}
	headers.Add("User-Agent", gojoyentAgent())
	return headers, nil
}

func getDateForRegion(credentials *auth.Credentials, isManta bool) string {
	if isManta {
		location, _ := time.LoadLocation(jpc.Locations["us-east-1"])
		return time.Now().In(location).Format(time.RFC1123)
	} else {
		location, _ := time.LoadLocation(jpc.Locations[credentials.Region()])
		return time.Now().In(location).Format(time.RFC1123)
	}
}

// JsonRequest JSON encodes and sends the object in reqData.ReqValue (if any) to the specified URL.
// Optional method arguments are passed using the RequestData object.
// Relevant RequestData fields:
// ReqHeaders: additional HTTP header values to add to the request.
// ExpectedStatus: the allowed HTTP response status values, else an error is returned.
// ReqValue: the data object to send.
// RespValue: the data object to decode the result into.
func (c *Client) JsonRequest(method, url, rfc1123Date string, request *RequestData, response *ResponseData) (err error) {
	err = nil
	var body []byte
	if request.Params != nil {
		url += "?" + request.Params.Encode()
	}
	if request.ReqValue != nil {
		body, err = json.Marshal(request.ReqValue)
		if err != nil {
			err = errors.Newf(err, "failed marshalling the request body")
			return
		}
	}
	headers, err := createHeaders(request.ReqHeaders, c.credentials, contentTypeJSON, rfc1123Date, c.apiVersion,
		isMantaRequest(url, c.credentials.UserAuthentication.User))
	if err != nil {
		return err
	}
	respBody, respHeader, err := c.sendRequest(
		method, url, bytes.NewReader(body), len(body), headers, response.ExpectedStatus, c.logger)
	if err != nil {
		return
	}
	defer respBody.Close()
	respData, err := ioutil.ReadAll(respBody)
	if err != nil {
		err = errors.Newf(err, "failed reading the response body")
		return
	}

	if len(respData) > 0 {
		if response.RespValue != nil {
			if dest, ok := response.RespValue.(*[]byte); ok {
				*dest = respData
				//err = decodeJSON(bytes.NewReader(respData), false, response.RespValue)
				//if err != nil {
				//	err = errors.Newf(err, "failed unmarshaling/decoding the response body: %s", respData)
				//}
			} else {
				err = json.Unmarshal(respData, response.RespValue)
				if err != nil {
					err = decodeJSON(bytes.NewReader(respData), true, response.RespValue)
					if err != nil {
						err = errors.Newf(err, "failed unmarshaling/decoding the response body: %s", respData)
					}
				}
			}
		}
	}

	if respHeader != nil {
		response.RespHeaders = respHeader
	}

	return
}

func decodeJSON(r io.Reader, multiple bool, into interface{}) error {
	d := json.NewDecoder(r)
	if multiple {
		return decodeStream(d, into)
	}
	return d.Decode(into)
}

func decodeStream(d *json.Decoder, into interface{}) error {
	t := reflect.TypeOf(into)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("unexpected type %s", t)
	}
	elemType := t.Elem().Elem()
	slice := reflect.ValueOf(into).Elem()
	for {
		val := reflect.New(elemType)
		if err := d.Decode(val.Interface()); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		slice.Set(reflect.Append(slice, val.Elem()))
	}
	return nil
}

// Sends the byte array in reqData.ReqValue (if any) to the specified URL.
// Optional method arguments are passed using the RequestData object.
// Relevant RequestData fields:
// ReqHeaders: additional HTTP header values to add to the request.
// ExpectedStatus: the allowed HTTP response status values, else an error is returned.
// ReqReader: an io.Reader providing the bytes to send.
// RespReader: assigned an io.ReadCloser instance used to read the returned data..
func (c *Client) BinaryRequest(method, url, rfc1123Date string, request *RequestData, response *ResponseData) (err error) {
	err = nil

	if request.Params != nil {
		url += "?" + request.Params.Encode()
	}
	headers, err := createHeaders(request.ReqHeaders, c.credentials, contentTypeOctetStream, rfc1123Date,
		c.apiVersion, isMantaRequest(url, c.credentials.UserAuthentication.User))
	if err != nil {
		return err
	}
	respBody, respHeader, err := c.sendRequest(
		method, url, request.ReqReader, request.ReqLength, headers, response.ExpectedStatus, c.logger)
	if err != nil {
		return
	}
	if response.RespReader != nil {
		response.RespReader = respBody
	}
	if respHeader != nil {
		response.RespHeaders = respHeader
	}
	return
}

// Sends the specified request to URL and checks that the HTTP response status is as expected.
// reqReader: a reader returning the data to send.
// length: the number of bytes to send.
// headers: HTTP headers to include with the request.
// expectedStatus: a slice of allowed response status codes.
func (c *Client) sendRequest(method, URL string, reqReader io.Reader, length int, headers http.Header,
	expectedStatus []int, logger *log.Logger) (rc io.ReadCloser, respHeader *http.Header, err error) {
	reqData := make([]byte, length)
	if reqReader != nil {
		nrRead, err := io.ReadFull(reqReader, reqData)
		if err != nil {
			err = errors.Newf(err, "failed reading the request data, read %v of %v bytes", nrRead, length)
			return rc, respHeader, err
		}
	}
	rawResp, err := c.sendRateLimitedRequest(method, URL, headers, reqData, logger)
	if err != nil {
		return
	}

	if logger != nil && c.trace {
		logger.Printf("Request: %s %s\n", method, URL)
		logger.Printf("Request header: %s\n", headers)
		logger.Printf("Request body: %s\n", reqData)
		logger.Printf("Response: %s\n", rawResp.Status)
		logger.Printf("Response header: %s\n", rawResp.Header)
		logger.Printf("Response body: %s\n", rawResp.Body)
		logger.Printf("Response error: %s\n", err)
	}

	foundStatus := false
	if len(expectedStatus) == 0 {
		expectedStatus = []int{http.StatusOK}
	}
	for _, status := range expectedStatus {
		if rawResp.StatusCode == status {
			foundStatus = true
			break
		}
	}
	if !foundStatus && len(expectedStatus) > 0 {
		err = handleError(URL, rawResp)
		rawResp.Body.Close()
		return
	}
	return rawResp.Body, &rawResp.Header, err
}

func (c *Client) sendRateLimitedRequest(method, URL string, headers http.Header, reqData []byte,
	logger *log.Logger) (resp *http.Response, err error) {
	for i := 0; i < c.maxSendAttempts; i++ {
		var reqReader io.Reader
		if reqData != nil {
			reqReader = bytes.NewReader(reqData)
		}
		req, err := http.NewRequest(method, URL, reqReader)
		if err != nil {
			err = errors.Newf(err, "failed creating the request %s", URL)
			return nil, err
		}
		// Setting req.Close to true to avoid malformed HTTP version "nullHTTP/1.1" error
		// See http://stackoverflow.com/questions/17714494/golang-http-request-results-in-eof-errors-when-making-multiple-requests-successi
		req.Close = true
		for header, values := range headers {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
		req.ContentLength = int64(len(reqData))
		resp, err = c.Do(req)
		if err != nil {
			return nil, errors.Newf(err, "failed executing the request %s", URL)
		}
		if resp.StatusCode != http.StatusRequestEntityTooLarge || resp.Header.Get("Retry-After") == "" {
			return resp, nil
		}
		resp.Body.Close()
		retryAfter, err := strconv.ParseFloat(resp.Header.Get("Retry-After"), 64)
		if err != nil {
			return nil, errors.Newf(err, "Invalid Retry-After header %s", URL)
		}
		if retryAfter == 0 {
			return nil, errors.Newf(err, "Resource limit exeeded at URL %s", URL)
		}
		if logger != nil {
			logger.Println("Too many requests, retrying in %dms.", int(retryAfter*1000))
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
	}
	return nil, errors.Newf(err, "Maximum number of attempts (%d) reached sending request to %s", c.maxSendAttempts, URL)
}

type HttpError struct {
	StatusCode      int
	Data            map[string][]string
	Url             string
	ResponseMessage string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("request %q returned unexpected status %d with body %q",
		e.Url,
		e.StatusCode,
		e.ResponseMessage,
	)
}

// The HTTP response status code was not one of those expected, so we construct an error.
// NotFound (404) codes have their own NotFound error type.
// We also make a guess at duplicate value errors.
func handleError(URL string, resp *http.Response) error {
	errBytes, _ := ioutil.ReadAll(resp.Body)
	errInfo := string(errBytes)
	// Check if we have a JSON representation of the failure, if so decode it.
	if resp.Header.Get("Content-Type") == contentTypeJSON {
		var errResponse ErrorResponse
		if err := json.Unmarshal(errBytes, &errResponse); err == nil {
			errInfo = errResponse.Message
		}
	}
	httpError := &HttpError{
		resp.StatusCode, map[string][]string(resp.Header), URL, errInfo,
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return errors.NewBadRequestf(httpError, "", "Bad request %s", URL)
	case http.StatusUnauthorized:
		return errors.NewNotAuthorizedf(httpError, "", "Unauthorised URL %s", URL)
		//return errors.NewInvalidCredentialsf(httpError, "", "Unauthorised URL %s", URL)
	case http.StatusForbidden:
		//return errors.
	case http.StatusNotFound:
		return errors.NewResourceNotFoundf(httpError, "", "Resource not found %s", URL)
	case http.StatusMethodNotAllowed:
		//return errors.
	case http.StatusNotAcceptable:
		return errors.NewInvalidHeaderf(httpError, "", "Invalid Header %s", URL)
	case http.StatusConflict:
		return errors.NewMissingParameterf(httpError, "", "Missing parameters %s", URL)
		//return errors.NewInvalidArgumentf(httpError, "", "Invalid parameter %s", URL)
	case http.StatusRequestEntityTooLarge:
		return errors.NewRequestTooLargef(httpError, "", "Request too large %s", URL)
	case http.StatusUnsupportedMediaType:
		//return errors.
	case http.StatusServiceUnavailable:
		return errors.NewInternalErrorf(httpError, "", "Internal error %s", URL)
	case 420:
		// SlowDown
		return errors.NewRequestThrottledf(httpError, "", "Request throttled %s", URL)
	case 422:
		// Unprocessable Entity
		return errors.NewInvalidArgumentf(httpError, "", "Invalid parameters %s", URL)
	case 449:
		// RetryWith
		return errors.NewInvalidVersionf(httpError, "", "Invalid version %s", URL)
		//RequestMovedError -> ?
	}

	return errors.NewUnknownErrorf(httpError, "", "Unknown error %s", URL)
}

func isMantaRequest(url, user string) bool {
	return strings.Contains(url, "/"+user+"/stor") || strings.Contains(url, "/"+user+"/jobs") || strings.Contains(url, "/"+user+"/public")
}
