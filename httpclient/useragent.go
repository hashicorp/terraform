package httpclient

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform/version"
)

const userAgentFormat = "Terraform/%s"

func UserAgentString() string {
	return fmt.Sprintf(userAgentFormat, version.Version)
}

type userAgentRoundTripper struct {
	inner     http.RoundTripper
	userAgent string
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", rt.userAgent)
	}
	return rt.inner.RoundTrip(req)
}
