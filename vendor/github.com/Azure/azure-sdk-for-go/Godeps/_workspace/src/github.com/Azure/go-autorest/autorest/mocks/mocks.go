/*
Package mocks provides mocks and helpers used in testing.
*/
package mocks

import (
	"fmt"
	"io"
	"net/http"
)

// Body implements acceptable body over a string.
type Body struct {
	s             string
	b             []byte
	isOpen        bool
	closeAttempts int
}

// NewBody creates a new instance of Body.
func NewBody(s string) *Body {
	return (&Body{s: s}).reset()
}

// Read reads into the passed byte slice and returns the bytes read.
func (body *Body) Read(b []byte) (n int, err error) {
	if !body.IsOpen() {
		return 0, fmt.Errorf("ERROR: Body has been closed\n")
	}
	if len(body.b) == 0 {
		return 0, io.EOF
	}
	n = copy(b, body.b)
	body.b = body.b[n:]
	return n, nil
}

// Close closes the body.
func (body *Body) Close() error {
	if body.isOpen {
		body.isOpen = false
		body.closeAttempts++
	}
	return nil
}

// CloseAttempts returns the number of times Close was called.
func (body *Body) CloseAttempts() int {
	return body.closeAttempts
}

// IsOpen returns true if the Body has not been closed, false otherwise.
func (body *Body) IsOpen() bool {
	return body.isOpen
}

func (body *Body) reset() *Body {
	body.isOpen = true
	body.b = []byte(body.s)
	return body
}

// Sender implements a simple null sender.
type Sender struct {
	attempts      int
	pollAttempts  int
	content       string
	reuseResponse bool
	resp          *http.Response
	status        string
	statusCode    int
	emitErrors    int
	err           error
}

// NewSender creates a new instance of Sender.
func NewSender() *Sender {
	return &Sender{status: "200 OK", statusCode: 200}
}

// Do accepts the passed request and, based on settings, emits a response and possible error.
func (c *Sender) Do(r *http.Request) (*http.Response, error) {
	c.attempts++

	if !c.reuseResponse || c.resp == nil {
		resp := NewResponse()
		resp.Request = r
		resp.Body = NewBody(c.content)
		resp.Status = c.status
		resp.StatusCode = c.statusCode
		c.resp = resp
	} else {
		c.resp.Body.(*Body).reset()
	}

	if c.pollAttempts > 0 {
		c.pollAttempts--
		c.resp.Status = "Accepted"
		c.resp.StatusCode = http.StatusAccepted
		SetAcceptedHeaders(c.resp)
	}

	if c.emitErrors > 0 || c.emitErrors < 0 {
		c.emitErrors--
		if c.err == nil {
			return c.resp, fmt.Errorf("Faux Error")
		}
		return c.resp, c.err
	}
	return c.resp, nil
}

// Attempts returns the number of times Do was called.
func (c *Sender) Attempts() int {
	return c.attempts
}

// EmitErrors sets the number times Do should emit an error.
func (c *Sender) EmitErrors(emit int) {
	c.emitErrors = emit
}

// SetError sets the error Do should return.
func (c *Sender) SetError(err error) {
	c.err = err
}

// ClearError clears the error Do emits.
func (c *Sender) ClearError() {
	c.SetError(nil)
}

// EmitContent sets the content to be returned by Do in the response body.
func (c *Sender) EmitContent(s string) {
	c.content = s
}

// EmitStatus sets the status of the response Do emits.
func (c *Sender) EmitStatus(status string, code int) {
	c.status = status
	c.statusCode = code
}

// SetPollAttempts sets the number of times the returned response emits the default polling
// status code (i.e., 202 Accepted).
func (c *Sender) SetPollAttempts(pa int) {
	c.pollAttempts = pa
}

// ReuseResponse sets if the just one response object should be reused by all calls to Do.
func (c *Sender) ReuseResponse(reuseResponse bool) {
	c.reuseResponse = reuseResponse
}

// SetResponse sets the response from Do.
func (c *Sender) SetResponse(resp *http.Response) {
	c.resp = resp
	c.reuseResponse = true
}

// T is a simple testing struct.
type T struct {
	Name string `json:"name" xml:"Name"`
	Age  int    `json:"age" xml:"Age"`
}
