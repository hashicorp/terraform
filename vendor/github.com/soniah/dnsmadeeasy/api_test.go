package dnsmadeeasy

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func makeClient(t *testing.T) *Client {
	client, err := NewClient("aaaaaa1a-11a1-1aa1-a101-11a1a11aa1aa",
		"11a0a11a-a1a1-111a-a11a-a11110a11111")

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.AKey != "aaaaaa1a-11a1-1aa1-a101-11a1a11aa1aa" {
		t.Fatalf("api key not set on client: %s", client.AKey)
	}

	if client.SKey != "11a0a11a-a1a1-111a-a11a-a11110a11111" {
		t.Fatalf("secret key not set on client: %s", client.SKey)
	}

	return client
}

func TestClient_NewRequest(t *testing.T) {
	c := makeClient(t)

	body := bytes.NewBuffer(nil)
	req, err := c.NewRequest("POST", "/bar", body, "Thu, 04 Dec 2014 11:02:57 GMT")
	if err != nil {
		t.Fatalf("bad: %v", err)
	}

	if req.URL.String() != "https://api.dnsmadeeasy.com/V2.0/bar" {
		t.Fatalf("bad base url: %v", req.URL.String())
	}

	if req.Header.Get("X-Dnsme-Apikey") != "aaaaaa1a-11a1-1aa1-a101-11a1a11aa1aa" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Header.Get("X-Dnsme-Requestdate") != "Thu, 04 Dec 2014 11:02:57 GMT" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Header.Get("X-Dnsme-Hmac") != "7a8c517d5eab84e524a537ce3a73e565cabf8f6a" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Method != "POST" {
		t.Fatalf("bad method: %v", req.Method)
	}
}

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() error {
	return nil
}

var errExample = &http.Response{
	Body: &ClosingBuffer{bytes.NewBufferString(`{"error":["Record with this type (A), name (test), and value (1.1.1.9) already exists."]}`)},
}

func Test_ParseError(t *testing.T) {
	should := errors.New("API Error (0): Record with this type (A), name (test), and value (1.1.1.9) already exists.")
	actual := parseError(errExample)

	if fmt.Sprintf("%v", should) != fmt.Sprintf("%v", actual) {
		t.Fatalf("parseError\nshould: |%v|\nactual: |%v|\n", should, actual)
	}
}
