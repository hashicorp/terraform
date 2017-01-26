package spotinst

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type responseWrapper struct {
	Request struct {
		ID string `json:"id"`
	} `json:"request"`
	Response struct {
		Errors []responseError   `json:"errors"`
		Items  []json.RawMessage `json:"items"`
	} `json:"response"`
}

type responseError struct {
	// Error code
	Code string `json:"code"`

	// Error message
	Message string `json:"message"`

	// Error field
	Field string `json:"field"`
}

// An Error reports the error caused by an API request
type Error struct {
	// HTTP response that caused this error
	Response *http.Response `json:"-"`

	// Error code
	Code string `json:"code"`

	// Error message
	Message string `json:"message"`

	// RequestID returned from the API, useful to contact support.
	RequestID string `json:"requestId"`
}

func (e *Error) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%v %v: %d (request %q) %v",
			e.Response.Request.Method, e.Response.Request.URL,
			e.Response.StatusCode, e.RequestID, e.Message)
	}
	return fmt.Sprintf("%v %v: %d %v",
		e.Response.Request.Method, e.Response.Request.URL,
		e.Response.StatusCode, e.Message)
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

// requireOK is used to verify response status code is a successful one (200 OK)
func requireOK(d time.Duration, resp *http.Response, err error) (time.Duration, *http.Response, error) {
	if err != nil {
		return d, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := extractError(resp)
		return d, nil, err
	}
	return d, resp, nil
}

// extractError is used to extract the inner/logical errors from the response
func extractError(resp *http.Response) error {
	b := bytes.NewBuffer(make([]byte, 0))

	// TeeReader returns a Reader that writes to b
	// what it reads from r.Body.
	reader := io.TeeReader(resp.Body, b)
	defer resp.Body.Close()
	resp.Body = ioutil.NopCloser(b)

	output := &responseWrapper{}
	if err := json.NewDecoder(reader).Decode(output); err != nil {
		return err
	}

	// TODO(liran): Should be extracted from the X-Request-ID header
	err := &Error{
		Response:  resp,
		RequestID: output.Request.ID,
	}

	if errs := output.Response.Errors; len(errs) > 0 {
		err.Code = errs[0].Code
		err.Message = errs[0].Message
	} else {
		err.Code = strconv.Itoa(resp.StatusCode)
		err.Message = http.StatusText(resp.StatusCode)
	}

	return err
}
