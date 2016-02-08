package dnsimple

import (
	"testing"
)

func makeClient(t *testing.T) *Client {
	client, err := NewClient("foobaremail", "foobartoken")

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.Token != "foobartoken" {
		t.Fatalf("token not set on client: %s", client.Token)
	}

	return client
}

func makeClientWithDomainToken(t *testing.T) *Client {
	client, err := NewClientWithDomainToken("foobardomaintoken")

	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if client.DomainToken != "foobardomaintoken" {
		t.Fatalf("domian token not set on client: %s", client.DomainToken)
	}

	return client
}

func TestClient_NewRequest(t *testing.T) {
	c := makeClient(t)

	body := map[string]interface{}{
		"foo": "bar",
		"baz": "bar",
	}
	req, err := c.NewRequest(body, "POST", "/bar")
	if err != nil {
		t.Fatalf("bad: %v", err)
	}

	if req.URL.String() != "https://api.dnsimple.com/v1/bar" {
		t.Fatalf("bad base url: %v", req.URL.String())
	}

	if req.Header.Get("X-DNSimple-Token") != "foobaremail:foobartoken" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Method != "POST" {
		t.Fatalf("bad method: %v", req.Method)
	}
}

func TestClientDomainToken_NewRequest(t *testing.T) {
	c := makeClientWithDomainToken(t)

	body := map[string]interface{}{
		"foo": "bar",
		"baz": "bar",
	}
	req, err := c.NewRequest(body, "POST", "/bar")
	if err != nil {
		t.Fatalf("bad: %v", err)
	}

	if req.URL.String() != "https://api.dnsimple.com/v1/bar" {
		t.Fatalf("bad base url: %v", req.URL.String())
	}

	if req.Header.Get("X-DNSimple-Domain-Token") != "foobardomaintoken" {
		t.Fatalf("bad auth header: %v", req.Header)
	}

	if req.Method != "POST" {
		t.Fatalf("bad method: %v", req.Method)
	}
}
