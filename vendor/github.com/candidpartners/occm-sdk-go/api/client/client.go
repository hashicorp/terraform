// Package implements helper client functionality
package client

import (
  "fmt"
	"io"
	"io/ioutil"
	"net/http"
  "net/http/cookiejar"
  "crypto/tls"

  "github.com/candidpartners/occm-sdk-go/util"
	"github.com/pkg/errors"
)

// API client context
type Context struct {
	Host       string
  cookieJar  *cookiejar.Jar
}

// API client facilitates making calls and holds API configuration
type Client struct {
  *Context
  apiUrl           string
	headers          http.Header
	httpClient       *http.Client
}

// New creates a new OCCM API client
func New(context *Context) (*Client, error) {
  err := validateContext(context)
  if err != nil {
		return nil, errors.Wrap(err, ErrInvalidContext)
	}

	client := &Client{
    apiUrl:       fmt.Sprintf("https://%s/occm/api", context.Host),
		headers:      make(http.Header),
  }

  client.headers.Set("Content-Type", "application/json")

  // allow self-signed and expired certs
  tr := &http.Transport{
  	TLSClientConfig: &tls.Config{InsecureSkipVerify : true},
  }

  // create default client
	if client.httpClient == nil {
		client.httpClient = &http.Client{Transport: tr}
	}

  // set cookie jar for the entire context
  if context.cookieJar == nil {
    cookieJar, _ := cookiejar.New(nil)
    context.cookieJar = cookieJar
  }

  client.httpClient.Jar = context.cookieJar

	return client, nil
}

// invoke makes an API request to the provided URI
func (client *Client) Invoke(method, uri string, qsParams map[string]string, bodyParams interface{}) ([]byte, map[string][]string, error) {
  // convert params
	var data io.Reader
	if bodyParams != nil {
		parsed, err := util.ToJSONStream(bodyParams);

    // fmt.Println("SENDING BODY: ", util.ToString(bodyParams))

    parsed, err = util.ToJSONStream(bodyParams);
    if err != nil {
  		return nil, nil, errors.Wrapf(err, ErrJSONConversion)
  	}
    data = parsed
	}

  // create HTTP request
  req, err := http.NewRequest(method, client.apiUrl + uri, data)
	if err != nil {
		return nil, nil, errors.Wrapf(err, ErrCreatingHttpRequestForUri, uri)
	}

	// clone existing headers
	req.Header = cloneHeader(client.headers)

  // set query string params
  if qsParams != nil {
    q := req.URL.Query()
    for key, val := range qsParams {
      q.Add(key, val)
    }
    req.URL.RawQuery = q.Encode()
  }

  // invoke the API
	res, err := client.httpClient.Do(req)
	if err != nil {
    return nil, nil, errors.Wrapf(err, ErrInvokingHttpRequestForUri, uri)
	}

	defer res.Body.Close()

  // read response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, ErrReadingResponseBody)
	}

  // fmt.Println("GOT RESPONSE: ", util.ToString(body), ": STATUS: ", res.StatusCode, "; HEADER: ", res.Header)

  // and assert status
  status := res.StatusCode
	switch status {
	case http.StatusOK:
		// no-op
	case http.StatusUnauthorized:
		return nil, nil, errors.Errorf(ErrUnauthorized)
	case http.StatusForbidden:
		return nil, nil, errors.Errorf(ErrForbidden)
	default:
    if status >= 200 && status < 300 {
      // all 200 errors are allowed
      break
    } else if status >= 500 && status < 600 {
      return nil, nil, errors.Errorf(ErrServerError, status)
    }

		var s string
		if body != nil {
			s = string(body)
		}
		return nil, nil, errors.Errorf(ErrUnexpectedHttpResponse, res.StatusCode, s)
	}

	return body, res.Header, nil
}

// cloneHeader clones the header
// copied from https://godoc.org/github.com/golang/gddo/httputil/header#Copy
func cloneHeader(header http.Header) http.Header {
	h := make(http.Header)
	for k, vs := range header {
		h[k] = vs
	}
	return h
}

func validateContext(context *Context) error {
  if context == nil {
		return errors.New(ErrInvalidContext)
	}

  if context.Host == "" {
    return errors.New(ErrInvalidHost)
  }

  return nil
}
