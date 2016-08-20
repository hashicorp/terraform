package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/nomad/testutil"
)

type configCallback func(c *Config)

// seen is used to track which tests we have already marked as parallel
var seen map[*testing.T]struct{}

func init() {
	seen = make(map[*testing.T]struct{})
}

func makeClient(t *testing.T, cb1 configCallback,
	cb2 testutil.ServerConfigCallback) (*Client, *testutil.TestServer) {
	// Make client config
	conf := DefaultConfig()
	if cb1 != nil {
		cb1(conf)
	}

	// Create server
	server := testutil.NewTestServer(t, cb2)
	conf.Address = "http://" + server.HTTPAddr

	// Create client
	client, err := NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	return client, server
}

func TestRequestTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		d, err := json.Marshal(struct{ Done bool }{true})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(d)
	}))
	defer srv.Close()

	conf := DefaultConfig()
	conf.Address = srv.URL

	client, err := NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out interface{}

	qm, err := client.query("/", &out, nil)
	if err != nil {
		t.Fatalf("query err: %v", err)
	}
	if qm.RequestTime == 0 {
		t.Errorf("bad request time: %d", qm.RequestTime)
	}

	wm, err := client.write("/", struct{ S string }{"input"}, &out, nil)
	if err != nil {
		t.Fatalf("write err: %v", err)
	}
	if wm.RequestTime == 0 {
		t.Errorf("bad request time: %d", wm.RequestTime)
	}

	wm, err = client.delete("/", &out, nil)
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if wm.RequestTime == 0 {
		t.Errorf("bad request time: %d", wm.RequestTime)
	}
}

func TestDefaultConfig_env(t *testing.T) {
	url := "http://1.2.3.4:5678"
	auth := []string{"nomaduser", "12345"}

	os.Setenv("NOMAD_ADDR", url)
	defer os.Setenv("NOMAD_ADDR", "")

	os.Setenv("NOMAD_HTTP_AUTH", strings.Join(auth, ":"))
	defer os.Setenv("NOMAD_HTTP_AUTH", "")

	config := DefaultConfig()

	if config.Address != url {
		t.Errorf("expected %q to be %q", config.Address, url)
	}

	if config.HttpAuth.Username != auth[0] {
		t.Errorf("expected %q to be %q", config.HttpAuth.Username, auth[0])
	}

	if config.HttpAuth.Password != auth[1] {
		t.Errorf("expected %q to be %q", config.HttpAuth.Password, auth[1])
	}
}

func TestSetQueryOptions(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()

	r := c.newRequest("GET", "/v1/jobs")
	q := &QueryOptions{
		Region:     "foo",
		AllowStale: true,
		WaitIndex:  1000,
		WaitTime:   100 * time.Second,
	}
	r.setQueryOptions(q)

	if r.params.Get("region") != "foo" {
		t.Fatalf("bad: %v", r.params)
	}
	if _, ok := r.params["stale"]; !ok {
		t.Fatalf("bad: %v", r.params)
	}
	if r.params.Get("index") != "1000" {
		t.Fatalf("bad: %v", r.params)
	}
	if r.params.Get("wait") != "100000ms" {
		t.Fatalf("bad: %v", r.params)
	}
}

func TestSetWriteOptions(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()

	r := c.newRequest("GET", "/v1/jobs")
	q := &WriteOptions{
		Region: "foo",
	}
	r.setWriteOptions(q)

	if r.params.Get("region") != "foo" {
		t.Fatalf("bad: %v", r.params)
	}
}

func TestRequestToHTTP(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()

	r := c.newRequest("DELETE", "/v1/jobs/foo")
	q := &QueryOptions{
		Region: "foo",
	}
	r.setQueryOptions(q)
	req, err := r.toHTTP()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if req.Method != "DELETE" {
		t.Fatalf("bad: %v", req)
	}
	if req.URL.RequestURI() != "/v1/jobs/foo?region=foo" {
		t.Fatalf("bad: %v", req)
	}
}

func TestParseQueryMeta(t *testing.T) {
	resp := &http.Response{
		Header: make(map[string][]string),
	}
	resp.Header.Set("X-Nomad-Index", "12345")
	resp.Header.Set("X-Nomad-LastContact", "80")
	resp.Header.Set("X-Nomad-KnownLeader", "true")

	qm := &QueryMeta{}
	if err := parseQueryMeta(resp, qm); err != nil {
		t.Fatalf("err: %v", err)
	}

	if qm.LastIndex != 12345 {
		t.Fatalf("Bad: %v", qm)
	}
	if qm.LastContact != 80*time.Millisecond {
		t.Fatalf("Bad: %v", qm)
	}
	if !qm.KnownLeader {
		t.Fatalf("Bad: %v", qm)
	}
}

func TestParseWriteMeta(t *testing.T) {
	resp := &http.Response{
		Header: make(map[string][]string),
	}
	resp.Header.Set("X-Nomad-Index", "12345")

	wm := &WriteMeta{}
	if err := parseWriteMeta(resp, wm); err != nil {
		t.Fatalf("err: %v", err)
	}

	if wm.LastIndex != 12345 {
		t.Fatalf("Bad: %v", wm)
	}
}

func TestQueryString(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()

	r := c.newRequest("PUT", "/v1/abc?foo=bar&baz=zip")
	q := &WriteOptions{Region: "foo"}
	r.setWriteOptions(q)

	req, err := r.toHTTP()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if uri := req.URL.RequestURI(); uri != "/v1/abc?baz=zip&foo=bar&region=foo" {
		t.Fatalf("bad uri: %q", uri)
	}
}
