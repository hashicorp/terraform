package disco

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
)

func TestMain(m *testing.M) {
	// During all tests we override the HTTP transport we use for discovery
	// so it'll tolerate the locally-generated TLS certificates we use
	// for test URLs.
	httpTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	os.Exit(m.Run())
}

func TestDiscover(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`
{
"thingy.v1": "http://example.com/foo",
"wotsit.v2": "http://example.net/bar"
}
`)
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Content-Length", strconv.Itoa(len(resp)))
			w.Write(resp)
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)
		gotURL := discovered.ServiceURL("thingy.v1")
		if gotURL == nil {
			t.Fatalf("found no URL for thingy.v1")
		}
		if got, want := gotURL.String(), "http://example.com/foo"; got != want {
			t.Fatalf("wrong result %q; want %q", got, want)
		}
	})
	t.Run("chunked encoding", func(t *testing.T) {
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`
{
"thingy.v1": "http://example.com/foo",
"wotsit.v2": "http://example.net/bar"
}
`)
			w.Header().Add("Content-Type", "application/json")
			// We're going to force chunked encoding here -- and thus prevent
			// the server from predicting the length -- so we can make sure
			// our client is tolerant of servers using this encoding.
			w.Write(resp[:5])
			w.(http.Flusher).Flush()
			w.Write(resp[5:])
			w.(http.Flusher).Flush()
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)
		gotURL := discovered.ServiceURL("wotsit.v2")
		if gotURL == nil {
			t.Fatalf("found no URL for wotsit.v2")
		}
		if got, want := gotURL.String(), "http://example.net/bar"; got != want {
			t.Fatalf("wrong result %q; want %q", got, want)
		}
	})
	t.Run("with credentials", func(t *testing.T) {
		var authHeaderText string
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`{}`)
			authHeaderText = r.Header.Get("Authorization")
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Content-Length", strconv.Itoa(len(resp)))
			w.Write(resp)
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		d.SetCredentialsSource(auth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
			host: map[string]interface{}{
				"token": "abc123",
			},
		}))
		d.Discover(host)
		if got, want := authHeaderText, "Bearer abc123"; got != want {
			t.Fatalf("wrong Authorization header\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("forced services override", func(t *testing.T) {
		forced := map[string]interface{}{
			"thingy.v1": "http://example.net/foo",
			"wotsit.v2": "/foo",
		}

		d := New()
		d.ForceHostServices(svchost.Hostname("example.com"), forced)

		givenHost := "example.com"
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		discovered := d.Discover(host)
		{
			gotURL := discovered.ServiceURL("thingy.v1")
			if gotURL == nil {
				t.Fatalf("found no URL for thingy.v1")
			}
			if got, want := gotURL.String(), "http://example.net/foo"; got != want {
				t.Fatalf("wrong result %q; want %q", got, want)
			}
		}
		{
			gotURL := discovered.ServiceURL("wotsit.v2")
			if gotURL == nil {
				t.Fatalf("found no URL for wotsit.v2")
			}
			if got, want := gotURL.String(), "https://example.com/foo"; got != want {
				t.Fatalf("wrong result %q; want %q", got, want)
			}
		}
	})
	t.Run("not JSON", func(t *testing.T) {
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`{"thingy.v1": "http://example.com/foo"}`)
			w.Header().Add("Content-Type", "application/octet-stream")
			w.Write(resp)
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)

		// result should be empty, which we can verify only by reaching into
		// its internals.
		if discovered.services != nil {
			t.Errorf("response not empty; should be")
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`{"thingy.v1": "htt`) // truncated, for example...
			w.Header().Add("Content-Type", "application/json")
			w.Write(resp)
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)

		// result should be empty, which we can verify only by reaching into
		// its internals.
		if discovered.services != nil {
			t.Errorf("response not empty; should be")
		}
	})
	t.Run("JSON with redundant charset", func(t *testing.T) {
		// The JSON RFC defines no parameters for the application/json
		// MIME type, but some servers have a weird tendency to just add
		// "charset" to everything, so we'll make sure we ignore it successfully.
		// (JSON uses content sniffing for encoding detection, not media type params.)
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`{"thingy.v1": "http://example.com/foo"}`)
			w.Header().Add("Content-Type", "application/json; charset=latin-1")
			w.Write(resp)
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)

		if discovered.services == nil {
			t.Errorf("response is empty; shouldn't be")
		}
	})
	t.Run("no discovery doc", func(t *testing.T) {
		portStr, close := testServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		})
		defer close()

		givenHost := "localhost" + portStr
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)

		// result should be empty, which we can verify only by reaching into
		// its internals.
		if discovered.services != nil {
			t.Errorf("response not empty; should be")
		}
	})
	t.Run("redirect", func(t *testing.T) {
		// For this test, we have two servers and one redirects to the other
		portStr1, close1 := testServer(func(w http.ResponseWriter, r *http.Request) {
			// This server is the one that returns a real response.
			resp := []byte(`{"thingy.v1": "http://example.com/foo"}`)
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Content-Length", strconv.Itoa(len(resp)))
			w.Write(resp)
		})
		portStr2, close2 := testServer(func(w http.ResponseWriter, r *http.Request) {
			// This server is the one that redirects.
			http.Redirect(w, r, "https://127.0.0.1"+portStr1+"/.well-known/terraform.json", 302)
		})
		defer close1()
		defer close2()

		givenHost := "localhost" + portStr2
		host, err := svchost.ForComparison(givenHost)
		if err != nil {
			t.Fatalf("test server hostname is invalid: %s", err)
		}

		d := New()
		discovered := d.Discover(host)

		gotURL := discovered.ServiceURL("thingy.v1")
		if gotURL == nil {
			t.Fatalf("found no URL for thingy.v1")
		}
		if got, want := gotURL.String(), "http://example.com/foo"; got != want {
			t.Fatalf("wrong result %q; want %q", got, want)
		}

		// The base URL for the host object should be the URL we redirected to,
		// rather than the we redirected _from_.
		gotBaseURL := discovered.discoURL.String()
		wantBaseURL := "https://127.0.0.1" + portStr1 + "/.well-known/terraform.json"
		if gotBaseURL != wantBaseURL {
			t.Errorf("incorrect base url %s; want %s", gotBaseURL, wantBaseURL)
		}

	})
}

func testServer(h func(w http.ResponseWriter, r *http.Request)) (portStr string, close func()) {
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Test server always returns 404 if the URL isn't what we expect
			if r.URL.Path != "/.well-known/terraform.json" {
				w.WriteHeader(404)
				w.Write([]byte("not found"))
				return
			}

			// If the URL is correct then the given hander decides the response
			h(w, r)
		},
	))

	serverURL, _ := url.Parse(server.URL)

	portStr = serverURL.Port()
	if portStr != "" {
		portStr = ":" + portStr
	}

	close = func() {
		server.Close()
	}

	return
}
