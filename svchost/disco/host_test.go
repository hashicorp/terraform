package disco

import (
	"net/url"
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
