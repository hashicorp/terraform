package heroku

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/pborman/uuid"
)

var DefaultTransport = &Transport{}

var DefaultClient = &http.Client{
	Transport: DefaultTransport,
}

type Transport struct {
	// Username is the HTTP basic auth username for API calls made by this Client.
	Username string

	// Password is the HTTP basic auth password for API calls made by this Client.
	Password string

	// UserAgent to be provided in API requests. Set to DefaultUserAgent if not
	// specified.
	UserAgent string

	// Debug mode can be used to dump the full request and response to stdout.
	Debug bool

	// AdditionalHeaders are extra headers to add to each HTTP request sent by
	// this Client.
	AdditionalHeaders http.Header

	// Transport is the HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// Forward CancelRequest to underlying Transport
func (t *Transport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	tr, ok := t.Transport.(canceler)
	if !ok {
		log.Printf("heroku: Client Transport of type %T doesn't support CancelRequest; Timeout not supported\n", t.Transport)
		return
	}
	tr.CancelRequest(req)
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Transport == nil {
		t.Transport = http.DefaultTransport
	}

	// Making a copy of the Request so that
	// we don't modify the Request we were given.
	req = cloneRequest(req)

	if t.UserAgent != "" {
		req.Header.Set("User-Agent", t.UserAgent)
	}

	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Request-Id", uuid.New())
	req.SetBasicAuth(t.Username, t.Password)
	for k, v := range t.AdditionalHeaders {
		req.Header[k] = v
	}

	if t.Debug {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Println(err)
		} else {
			os.Stderr.Write(dump)
			os.Stderr.Write([]byte{'\n', '\n'})
		}
	}

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	if t.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Println(err)
		} else {
			os.Stderr.Write(dump)
			os.Stderr.Write([]byte{'\n'})
		}
	}

	if err = checkResponse(resp); err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	return resp, nil
}

type Error struct {
	error
	ID  string
	URL string
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode/100 != 2 { // 200, 201, 202, etc
		var e struct {
			Message string
			ID      string
			URL     string `json:"url"`
		}
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return fmt.Errorf("encountered an error : %s", resp.Status)
		}
		return Error{error: errors.New(e.Message), ID: e.ID, URL: e.URL}
	}
	if msg := resp.Header.Get("X-Heroku-Warning"); msg != "" {
		log.Println(strings.TrimSpace(msg))
	}
	return nil
}

// cloneRequest returns a clone of the provided *http.Request.
func cloneRequest(req *http.Request) *http.Request {
	// shallow copy of the struct
	clone := new(http.Request)
	*clone = *req
	// deep copy of the Header
	clone.Header = make(http.Header)
	for k, s := range req.Header {
		clone.Header[k] = s
	}
	return clone
}
