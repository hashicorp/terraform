package winrm

import (
	. "gopkg.in/check.v1"
)

func (s *WinRMSuite) TestEndpointUrlHttp(c *C) {
	endpoint := &Endpoint{Host: "abc", Port: 123}
	c.Assert(endpoint.url(), Equals, "http://abc:123/wsman")
}

func (s *WinRMSuite) TestEndpointUrlHttps(c *C) {
	endpoint := &Endpoint{Host: "abc", Port: 123, HTTPS: true}
	c.Assert(endpoint.url(), Equals, "https://abc:123/wsman")
}
