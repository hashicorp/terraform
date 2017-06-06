package librato

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion = "0.1"
	defaultBaseURL = "https://metrics-api.librato.com/v1/"
	userAgent      = "go-librato/" + libraryVersion

	defaultMediaType = "application/json"
)

// A Client manages communication with the Librato API.
type Client struct {
	// HTTP client used to communicate with the API
	client *http.Client

	// Headers to attach to every request made with the client. Headers will be
	// used to provide Librato API authentication details and other necessary
	// headers.
	Headers map[string]string

	// Email and Token contains the authentication details needed to authenticate
	// against the Librato API.
	Email, Token string

	// Base URL for API requests. Defaults to the public Librato API, but can be
	// set to an alternate endpoint if necessary. BaseURL should always be
	// terminated by a slash.
	BaseURL *url.URL

	// User agent used when communicating with the Librato API.
	UserAgent string

	// Services used to manipulate API entities.
	Spaces   *SpacesService
	Metrics  *MetricsService
	Alerts   *AlertsService
	Services *ServicesService
}

// NewClient returns a new Librato API client bound to the public Librato API.
func NewClient(email, token string) *Client {
	bu, err := url.Parse(defaultBaseURL)
	if err != nil {
		panic("Default Librato API base URL couldn't be parsed")
	}

	return NewClientWithBaseURL(bu, email, token)
}

// NewClientWithBaseURL returned a new Librato API client with a custom base URL.
func NewClientWithBaseURL(baseURL *url.URL, email, token string) *Client {
	headers := map[string]string{
		"Content-Type": defaultMediaType,
		"Accept":       defaultMediaType,
	}

	c := &Client{
		client:    http.DefaultClient,
		Headers:   headers,
		Email:     email,
		Token:     token,
		BaseURL:   baseURL,
		UserAgent: userAgent,
	}

	c.Spaces = &SpacesService{client: c}
	c.Metrics = &MetricsService{client: c}
	c.Alerts = &AlertsService{client: c}
	c.Services = &ServicesService{client: c}

	return c
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body. If specified, the map provided by headers will be used to
// update request headers.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	u := c.BaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		encodeErr := json.NewEncoder(buf).Encode(body)
		if encodeErr != nil {
			return nil, encodeErr
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Email, c.Token)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

// Do sends an API request and returns the API response.  The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred.  If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = CheckResponse(resp)
	if err != nil {
		return resp, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}

	return resp, err
}

// ErrorResponse reports an error caused by an API request.
// ErrorResponse implements the Error interface.
type ErrorResponse struct {
	// HTTP response that caused this error
	Response *http.Response

	// Error messages produces by Librato API.
	Errors ErrorResponseMessages `json:"errors"`
}

func (er *ErrorResponse) Error() string {
	buf := new(bytes.Buffer)

	if er.Errors.Params != nil && len(er.Errors.Params) > 0 {
		buf.WriteString(" Parameter errors:")
		for param, errs := range er.Errors.Params {
			fmt.Fprintf(buf, " %s:", param)
			for _, err := range errs {
				fmt.Fprintf(buf, " %s,", err)
			}
		}
		buf.WriteString(".")
	}

	if er.Errors.Request != nil && len(er.Errors.Request) > 0 {
		buf.WriteString(" Request errors:")
		for _, err := range er.Errors.Request {
			fmt.Fprintf(buf, " %s,", err)
		}
		buf.WriteString(".")
	}

	if er.Errors.System != nil && len(er.Errors.System) > 0 {
		buf.WriteString(" System errors:")
		for _, err := range er.Errors.System {
			fmt.Fprintf(buf, " %s,", err)
		}
		buf.WriteString(".")
	}

	return fmt.Sprintf(
		"%v %v: %d %v",
		er.Response.Request.Method,
		er.Response.Request.URL,
		er.Response.StatusCode,
		buf.String(),
	)
}

// ErrorResponseMessages contains error messages returned from the Librato API.
type ErrorResponseMessages struct {
	Params  map[string][]string `json:"params,omitempty"`
	Request []string            `json:"request,omitempty"`
	System  []string            `json:"system,omitempty"`
}

// CheckResponse checks the API response for errors; and returns them if
// present. A Response is considered an error if it has a status code outside
// the 2XX range.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}

	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}

	return errorResponse
}

func urlWithOptions(s string, opt interface{}) (string, error) {
	rv := reflect.ValueOf(opt)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return "", err
	}
	u.RawQuery = qs.Encode()

	return u.String(), nil
}

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}

// Int is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it, but unlike Int32
// its argument value is an int.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// Uint is a helper routine that allocates a new uint value
// to store v and returns a pointer to it.
func Uint(v uint) *uint {
	p := new(uint)
	*p = v
	return p
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}

// Float is a helper routine that allocates a new float64 value
// to store v and returns a pointer to it.
func Float(v float64) *float64 {
	p := new(float64)
	*p = v
	return p
}
