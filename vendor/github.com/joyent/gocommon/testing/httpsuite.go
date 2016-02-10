package testing

// This package provides an HTTPSuite infrastructure that lets you bring up an
// HTTP server. The server will handle requests based on whatever Handlers are
// attached to HTTPSuite.Mux. This Mux is reset after every test case, and the
// server is shut down at the end of the test suite.

import (
	gc "launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
)

var _ = gc.Suite(&HTTPSuite{})

type HTTPSuite struct {
	Server     *httptest.Server
	Mux        *http.ServeMux
	oldHandler http.Handler
	UseTLS     bool
}

func (s *HTTPSuite) SetUpSuite(c *gc.C) {
	if s.UseTLS {
		s.Server = httptest.NewTLSServer(nil)
	} else {
		s.Server = httptest.NewServer(nil)
	}
}

func (s *HTTPSuite) SetUpTest(c *gc.C) {
	s.oldHandler = s.Server.Config.Handler
	s.Mux = http.NewServeMux()
	s.Server.Config.Handler = s.Mux
}

func (s *HTTPSuite) TearDownTest(c *gc.C) {
	s.Mux = nil
	s.Server.Config.Handler = s.oldHandler
}

func (s *HTTPSuite) TearDownSuite(c *gc.C) {
	if s.Server != nil {
		s.Server.Close()
	}
}
