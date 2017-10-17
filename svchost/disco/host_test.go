package disco

import (
	"net/url"
	"testing"
)

func TestHostServiceURL(t *testing.T) {
	baseURL, _ := url.Parse("https://example.com/disco/foo.json")
	host := Host{
		discoURL: baseURL,
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
		Want string
	}{
		{"absolute.v1", "http://example.net/foo/bar"},
		{"absolutewithport.v1", "http://example.net:8080/foo/bar"},
		{"relative.v1", "https://example.com/disco/stu/"},
		{"rootrelative.v1", "https://example.com/baz"},
		{"protorelative.v1", "https://example.net/"},
		{"withfragment.v1", "http://example.org/"},
		{"querystring.v1", "https://example.net/baz?foo=bar"}, // most callers will disregard query string
		{"nothttp.v1", "<nil>"},
		{"invalid.v1", "<nil>"},
	}

	for _, test := range tests {
		t.Run(test.ID, func(t *testing.T) {
			url := host.ServiceURL(test.ID)
			var got string
			if url != nil {
				got = url.String()
			} else {
				got = "<nil>"
			}

			if got != test.Want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.Want)
			}
		})
	}
}
