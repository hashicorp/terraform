package winrm

import (
	. "gopkg.in/check.v1"
)

func (s *WinRMSuite) TestDefaultParameters(c *C) {
	params := DefaultParameters()
	c.Assert(params.Locale, Equals, "en-US")
	c.Assert(params.Timeout, Equals, "PT60S")
	c.Assert(params.EnvelopeSize, Equals, 153600)
}

func (s *WinRMSuite) TestParameters(c *C) {
	params := NewParameters("PT120S", "fr-FR", 128)
	c.Assert(params.Locale, Equals, "fr-FR")
	c.Assert(params.Timeout, Equals, "PT120S")
	c.Assert(params.EnvelopeSize, Equals, 128)
}
