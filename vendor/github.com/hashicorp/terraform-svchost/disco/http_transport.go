package disco

import (
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
)

const DefaultUserAgent = "terraform-svchost/1.0"

func defaultHttpTransport() http.RoundTripper {
	t := cleanhttp.DefaultPooledTransport()
	return &userAgentRoundTripper{
		innerRt:   t,
		userAgent: DefaultUserAgent,
	}
}

type userAgentRoundTripper struct {
	innerRt   http.RoundTripper
	userAgent string
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", rt.userAgent)
	}

	return rt.innerRt.RoundTrip(req)
}
