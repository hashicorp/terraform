package mailgun

import (
	"os"
	"testing"

	. "github.com/motain/gocheck"
	"github.com/pearkes/mailgun/testutil"
)

type S struct {
	client *Client
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	var err error
	s.client, err = NewClient("foobar")
	s.client.URL = "http://localhost:4444"
	if err != nil {
		panic(err)
	}
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
}

func makeClient(t *testing.T) *Client {
	client, err := NewClient("foobarkey")

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.ApiKey != "foobarkey" {
		t.Fatalf("key not set on client: %s", client.ApiKey)
	}

	return client
}

func Test_NewClient_env(t *testing.T) {
	os.Setenv("MAILGUN_API_KEY", "bar")
	client, err := NewClient("")

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.ApiKey != "bar" {
		t.Fatalf("key not set on client: %s", client.ApiKey)
	}
}

func TestClient_NewRequest(t *testing.T) {
	c := makeClient(t)

	params := map[string]string{
		"foo": "bar",
		"baz": "bar",
	}
	req, err := c.NewRequest(params, "POST", "/bar")
	if err != nil {
		t.Fatalf("bad: %v", err)
	}

	encoded := req.URL.Query()
	if encoded.Get("foo") != "bar" {
		t.Fatalf("bad: %v", encoded)
	}

	if encoded.Get("baz") != "bar" {
		t.Fatalf("bad: %v", encoded)
	}

	if req.URL.String() != "https://api.mailgun.net/v2/bar?baz=bar&foo=bar" {
		t.Fatalf("bad base url: %v", req.URL.String())
	}

	if req.Header.Get("Authorization") != "Basic YXBpOmZvb2JhcmtleQ==" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Method != "POST" {
		t.Fatalf("bad method: %v", req.Method)
	}
}
