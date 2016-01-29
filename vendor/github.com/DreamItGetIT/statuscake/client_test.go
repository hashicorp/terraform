package statuscake

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_validate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	auth := &Auth{}
	err := auth.validate()

	require.NotNil(err)
	assert.Contains(err.Error(), "Username is required")
	assert.Contains(err.Error(), "Apikey is required")

	auth.Username = "foo"
	err = auth.validate()

	require.NotNil(err)
	assert.Equal("Apikey is required", err.Error())

	auth.Apikey = "bar"
	err = auth.validate()
	assert.Nil(err)
}

func TestClient(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	assert.Equal("random-user", c.username)
	assert.Equal("my-pass", c.apiKey)
}

func TestClient_newRequest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	r, err := c.newRequest("GET", "/hello", nil, nil)

	require.Nil(err)
	assert.Equal("GET", r.Method)
	assert.Equal("https://www.statuscake.com/API/hello", r.URL.String())
	assert.Equal("random-user", r.Header.Get("Username"))
	assert.Equal("my-pass", r.Header.Get("API"))
}

func TestClient_doRequest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	hc := &fakeHTTPClient{StatusCode: 200}
	c.c = hc

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	require.Nil(err)

	_, err = c.doRequest(req)
	require.Nil(err)

	assert.Len(hc.requests, 1)
	assert.Equal("http://example.com/test", hc.requests[0].URL.String())
}

func TestClient_doRequest_WithHTTPErrors(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	hc := &fakeHTTPClient{
		StatusCode: 500,
	}
	c.c = hc

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	require.Nil(err)

	_, err = c.doRequest(req)
	require.NotNil(err)
	assert.IsType(&httpError{}, err)
}

func TestClient_doRequest_HttpAuthenticationErrors(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	hc := &fakeHTTPClient{
		StatusCode: 200,
		Fixture:    "auth_error.json",
	}
	c.c = hc

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	require.Nil(err)

	_, err = c.doRequest(req)
	require.NotNil(err)
	assert.IsType(&AuthenticationError{}, err)
}

func TestClient_get(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	hc := &fakeHTTPClient{}
	c.c = hc

	c.get("/hello", nil)
	assert.Len(hc.requests, 1)
	assert.Equal("GET", hc.requests[0].Method)
	assert.Equal("https://www.statuscake.com/API/hello", hc.requests[0].URL.String())
}

func TestClient_put(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	hc := &fakeHTTPClient{}
	c.c = hc

	v := url.Values{"foo": {"bar"}}
	c.put("/hello", v)
	assert.Len(hc.requests, 1)
	assert.Equal("PUT", hc.requests[0].Method)
	assert.Equal("https://www.statuscake.com/API/hello", hc.requests[0].URL.String())

	b, err := ioutil.ReadAll(hc.requests[0].Body)
	require.Nil(err)
	assert.Equal("foo=bar", string(b))
}

func TestClient_delete(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	hc := &fakeHTTPClient{}
	c.c = hc

	v := url.Values{"foo": {"bar"}}
	c.delete("/hello", v)
	assert.Len(hc.requests, 1)
	assert.Equal("DELETE", hc.requests[0].Method)
	assert.Equal("https://www.statuscake.com/API/hello?foo=bar", hc.requests[0].URL.String())
}

func TestClient_Tests(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	c, err := New(Auth{Username: "random-user", Apikey: "my-pass"})
	require.Nil(err)

	expected := &tests{
		client: c,
	}

	assert.Equal(expected, c.Tests())
}

type fakeBody struct {
	io.Reader
}

func (f *fakeBody) Close() error {
	return nil
}

type fakeHTTPClient struct {
	StatusCode int
	Fixture    string
	requests   []*http.Request
}

func (c *fakeHTTPClient) Do(r *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, r)
	var body []byte

	if c.Fixture != "" {
		p := filepath.Join("fixtures", c.Fixture)
		f, err := os.Open(p)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		b, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		body = b
	}

	resp := &http.Response{
		StatusCode: c.StatusCode,
		Body:       &fakeBody{Reader: bytes.NewReader(body)},
	}

	return resp, nil
}
