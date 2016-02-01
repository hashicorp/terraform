package testutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type HTTPServer struct {
	URL      string
	Timeout  time.Duration
	started  bool
	request  chan *http.Request
	response chan ResponseFunc
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    string
}

func NewHTTPServer() *HTTPServer {
	return &HTTPServer{URL: "http://localhost:4444", Timeout: 5 * time.Second}
}

type ResponseFunc func(path string) Response

func (s *HTTPServer) Start() {
	if s.started {
		return
	}
	s.started = true
	s.request = make(chan *http.Request, 1024)
	s.response = make(chan ResponseFunc, 1024)
	u, err := url.Parse(s.URL)
	if err != nil {
		panic(err)
	}
	l, err := net.Listen("tcp", u.Host)
	if err != nil {
		panic(err)
	}
	go http.Serve(l, s)

	s.Response(203, nil, "")
	for {
		// Wait for it to be up.
		resp, err := http.Get(s.URL)
		if err == nil && resp.StatusCode == 203 {
			break
		}
		time.Sleep(1e8)
	}
	s.WaitRequest() // Consume dummy request.
}

// Flush discards all pending requests and responses.
func (s *HTTPServer) Flush() {
	for {
		select {
		case <-s.request:
		case <-s.response:
		default:
			return
		}
	}
}

func body(req *http.Request) string {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.ParseMultipartForm(1e6)
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(data))
	s.request <- req
	var resp Response
	select {
	case respFunc := <-s.response:
		resp = respFunc(req.URL.Path)
	case <-time.After(s.Timeout):
		const msg = "ERROR: Timeout waiting for test to prepare a response\n"
		fmt.Fprintf(os.Stderr, msg)
		resp = Response{500, nil, msg}
	}
	if resp.Headers != nil {
		h := w.Header()
		for k, v := range resp.Headers {
			h.Set(k, v)
		}
	}
	if resp.Status != 0 {
		w.WriteHeader(resp.Status)
	}
	w.Write([]byte(resp.Body))
}

// WaitRequests returns the next n requests made to the http server from
// the queue. If not enough requests were previously made, it waits until
// the timeout value for them to be made.
func (s *HTTPServer) WaitRequests(n int) []*http.Request {
	reqs := make([]*http.Request, 0, n)
	for i := 0; i < n; i++ {
		select {
		case req := <-s.request:
			reqs = append(reqs, req)
		case <-time.After(s.Timeout):
			panic("Timeout waiting for request")
		}
	}
	return reqs
}

// WaitRequest returns the next request made to the http server from
// the queue. If no requests were previously made, it waits until the
// timeout value for one to be made.
func (s *HTTPServer) WaitRequest() *http.Request {
	return s.WaitRequests(1)[0]
}

// ResponseFunc prepares the test server to respond the following n
// requests using f to build each response.
func (s *HTTPServer) ResponseFunc(n int, f ResponseFunc) {
	for i := 0; i < n; i++ {
		s.response <- f
	}
}

// ResponseMap maps request paths to responses.
type ResponseMap map[string]Response

// ResponseMap prepares the test server to respond the following n
// requests using the m to obtain the responses.
func (s *HTTPServer) ResponseMap(n int, m ResponseMap) {
	f := func(path string) Response {
		for rpath, resp := range m {
			if rpath == path {
				return resp
			}
		}
		body := "Path not found in response map: " + path
		return Response{Status: 500, Body: body}
	}
	s.ResponseFunc(n, f)
}

// Responses prepares the test server to respond the following n requests
// using the provided response parameters.
func (s *HTTPServer) Responses(n int, status int, headers map[string]string, body string) {
	f := func(path string) Response {
		return Response{status, headers, body}
	}
	s.ResponseFunc(n, f)
}

// Response prepares the test server to respond the following request
// using the provided response parameters.
func (s *HTTPServer) Response(status int, headers map[string]string, body string) {
	s.Responses(1, status, headers, body)
}
