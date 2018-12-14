package disco

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestHostServiceURL(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com/disco/foo.json")
	host := Host{
		discoURL: baseURL,
		hostname: "test-server",
		services: map[string]interface{}{
			"absolute.v1":         "http://example.net/foo/bar",
			"absolutewithport.v1": "http://example.net:8080/foo/bar",
			"relative.v1":         "./stu/",
			"rootrelative.v1":     "/baz",
			"protorelative.v1":    "//example.net/",
			"withfragment.v1":     "http://example.org/#foo",
			"querystring.v1":      "https://example.net/baz?foo=bar",
			"nothttp.v1":          "ftp://127.0.0.1/pub/",
			"invalid.v1":          "***not A URL at all!:/<@@@@>***",
		},
	}

	tests := []struct {
		ID   string
		want string
		err  string
	}{
		{"absolute.v1", "http://example.net/foo/bar", ""},
		{"absolutewithport.v1", "http://example.net:8080/foo/bar", ""},
		{"relative.v1", "https://example.com/disco/stu/", ""},
		{"rootrelative.v1", "https://example.com/baz", ""},
		{"protorelative.v1", "https://example.net/", ""},
		{"withfragment.v1", "http://example.org/", ""},
		{"querystring.v1", "https://example.net/baz?foo=bar", ""},
		{"nothttp.v1", "<nil>", "unsupported scheme"},
		{"invalid.v1", "<nil>", "Failed to parse service URL"},
	}

	for _, test := range tests {
		t.Run(test.ID, func(t *testing.T) {
			url, err := host.ServiceURL(test.ID)
			if (err != nil || test.err != "") &&
				(err == nil || !strings.Contains(err.Error(), test.err)) {
				t.Fatalf("unexpected service URL error: %s", err)
			}

			var got string
			if url != nil {
				got = url.String()
			} else {
				got = "<nil>"
			}

			if got != test.want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.want)
			}
		})
	}
}

func TestVersionConstrains(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com/disco/foo.json")

	t.Run("exact service version is provided", func(t *testing.T) {
		portStr, close := testVersionsServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`
{
	"service": "%s",
	"product": "%s",
	"minimum": "0.11.8",
	"maximum": "0.12.0"
}`)
			// Add the requested service and product to the response.
			service := path.Base(r.URL.Path)
			product := r.URL.Query().Get("product")
			resp = []byte(fmt.Sprintf(string(resp), service, product))

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Content-Length", strconv.Itoa(len(resp)))
			w.Write(resp)
		})
		defer close()

		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"thingy.v1":   "/api/v1/",
				"thingy.v2":   "/api/v2/",
				"versions.v1": "https://localhost" + portStr + "/v1/versions/",
			},
		}

		expected := &Constraints{
			Service: "thingy.v1",
			Product: "terraform",
			Minimum: "0.11.8",
			Maximum: "0.12.0",
		}

		actual, err := host.VersionConstraints("thingy.v1", "terraform")
		if err != nil {
			t.Fatalf("unexpected version constraints error: %s", err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected %#v, got: %#v", expected, actual)
		}
	})

	t.Run("service provided with different versions", func(t *testing.T) {
		portStr, close := testVersionsServer(func(w http.ResponseWriter, r *http.Request) {
			resp := []byte(`
{
	"service": "%s",
	"product": "%s",
	"minimum": "0.11.8",
	"maximum": "0.12.0"
}`)
			// Add the requested service and product to the response.
			service := path.Base(r.URL.Path)
			product := r.URL.Query().Get("product")
			resp = []byte(fmt.Sprintf(string(resp), service, product))

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Content-Length", strconv.Itoa(len(resp)))
			w.Write(resp)
		})
		defer close()

		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"thingy.v2":   "/api/v2/",
				"thingy.v3":   "/api/v3/",
				"versions.v1": "https://localhost" + portStr + "/v1/versions/",
			},
		}

		expected := &Constraints{
			Service: "thingy.v3",
			Product: "terraform",
			Minimum: "0.11.8",
			Maximum: "0.12.0",
		}

		actual, err := host.VersionConstraints("thingy.v1", "terraform")
		if err != nil {
			t.Fatalf("unexpected version constraints error: %s", err)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected %#v, got: %#v", expected, actual)
		}
	})

	t.Run("service not provided", func(t *testing.T) {
		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"versions.v1": "https://localhost/v1/versions/",
			},
		}

		_, err := host.VersionConstraints("thingy.v1", "terraform")
		if _, ok := err.(*ErrServiceNotProvided); !ok {
			t.Fatalf("expected service not provided error, got: %v", err)
		}
	})

	t.Run("versions service returns a 404", func(t *testing.T) {
		portStr, close := testVersionsServer(nil)
		defer close()

		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"thingy.v1":   "/api/v1/",
				"versions.v1": "https://localhost" + portStr + "/v1/non-existent/",
			},
		}

		_, err := host.VersionConstraints("thingy.v1", "terraform")
		if _, ok := err.(*ErrNoVersionConstraints); !ok {
			t.Fatalf("expected service not provided error, got: %v", err)
		}
	})

	t.Run("checkpoint is disabled", func(t *testing.T) {
		if err := os.Setenv("CHECKPOINT_DISABLE", "1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Unsetenv("CHECKPOINT_DISABLE")

		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"thingy.v1":   "/api/v1/",
				"versions.v1": "https://localhost/v1/versions/",
			},
		}

		_, err := host.VersionConstraints("thingy.v1", "terraform")
		if _, ok := err.(*ErrNoVersionConstraints); !ok {
			t.Fatalf("expected service not provided error, got: %v", err)
		}
	})

	t.Run("versions service not discovered", func(t *testing.T) {
		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"thingy.v1": "/api/v1/",
			},
		}

		_, err := host.VersionConstraints("thingy.v1", "terraform")
		if _, ok := err.(*ErrServiceNotProvided); !ok {
			t.Fatalf("expected service not provided error, got: %v", err)
		}
	})

	t.Run("versions service version not discovered", func(t *testing.T) {
		host := Host{
			discoURL:  baseURL,
			hostname:  "test-server",
			transport: httpTransport,
			services: map[string]interface{}{
				"thingy.v1":   "/api/v1/",
				"versions.v2": "https://localhost/v2/versions/",
			},
		}

		_, err := host.VersionConstraints("thingy.v1", "terraform")
		if _, ok := err.(*ErrVersionNotSupported); !ok {
			t.Fatalf("expected service not provided error, got: %v", err)
		}
	})
}

func testVersionsServer(h func(w http.ResponseWriter, r *http.Request)) (portStr string, close func()) {
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// Test server always returns 404 if the URL isn't what we expect
			if !strings.HasPrefix(r.URL.Path, "/v1/versions/") {
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

	return portStr, close
}
