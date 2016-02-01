package autorest

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	// DefaultPollingDelay is the default delay between polling requests (only used if the
	// http.Request lacks a well-formed Retry-After header).
	DefaultPollingDelay = 60 * time.Second

	// DefaultPollingDuration is the default total polling duration.
	DefaultPollingDuration = 15 * time.Minute
)

// PollingMode sets how, if at all, clients composed with Client will poll.
type PollingMode string

const (
	// PollUntilAttempts polling mode polls until reaching a maximum number of attempts.
	PollUntilAttempts PollingMode = "poll-until-attempts"

	// PollUntilDuration polling mode polls until a specified time.Duration has passed.
	PollUntilDuration PollingMode = "poll-until-duration"

	// DoNotPoll disables polling.
	DoNotPoll PollingMode = "not-at-all"
)

const (
	requestFormat = `HTTP Request Begin ===================================================
%s
===================================================== HTTP Request End
`
	responseFormat = `HTTP Response Begin ===================================================
%s
===================================================== HTTP Response End
`
)

// LoggingInspector implements request and response inspectors that log the full request and
// response to a supplied log.
type LoggingInspector struct {
	Logger *log.Logger
}

// WithInspection returns a PrepareDecorator that emits the http.Request to the supplied logger. The
// body is restored after being emitted.
//
// Note: Since it reads the entire Body, this decorator should not be used where body streaming is
// important. It is best used to trace JSON or similar body values.
func (li LoggingInspector) WithInspection() PrepareDecorator {
	return func(p Preparer) Preparer {
		return PreparerFunc(func(r *http.Request) (*http.Request, error) {
			var body, b bytes.Buffer

			defer r.Body.Close()

			r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &body))
			r.Write(&b)

			li.Logger.Printf(requestFormat, b.String())

			r.Body = ioutil.NopCloser(&body)
			return p.Prepare(r)
		})
	}
}

// ByInspecting returns a RespondDecorator that emits the http.Response to the supplied logger. The
// body is restored after being emitted.
//
// Note: Since it reads the entire Body, this decorator should not be used where body streaming is
// important. It is best used to trace JSON or similar body values.
func (li LoggingInspector) ByInspecting() RespondDecorator {
	return func(r Responder) Responder {
		return ResponderFunc(func(resp *http.Response) error {
			var body, b bytes.Buffer

			defer resp.Body.Close()

			resp.Body = ioutil.NopCloser(io.TeeReader(resp.Body, &body))
			resp.Write(&b)

			li.Logger.Printf(responseFormat, b.String())

			resp.Body = ioutil.NopCloser(&body)
			return r.Respond(resp)
		})
	}
}

var (
	// DefaultClient is the base from which generated clients should create a Client instance. Users
	// can then established widely used Client defaults by replacing or modifying the DefaultClient
	// before instantiating a generated client.
	DefaultClient = Client{PollingMode: PollUntilDuration, PollingDuration: DefaultPollingDuration}
)

// Client is the base for autorest generated clients. It provides default, "do nothing"
// implementations of an Authorizer, RequestInspector, and ResponseInspector. It also returns the
// standard, undecorated http.Client as a default Sender. Lastly, it supports basic request polling,
// limited to a maximum number of attempts or a specified duration.
//
// Generated clients should also use Error (see NewError and NewErrorWithError) for errors and
// return responses that compose with Response.
//
// Most customization of generated clients is best achieved by supplying a custom Authorizer, custom
// RequestInspector, and / or custom ResponseInspector. Users may log requests, implement circuit
// breakers (see https://msdn.microsoft.com/en-us/library/dn589784.aspx) or otherwise influence
// sending the request by providing a decorated Sender.
type Client struct {
	Authorizer        Authorizer
	Sender            Sender
	RequestInspector  PrepareDecorator
	ResponseInspector RespondDecorator

	PollingMode     PollingMode
	PollingAttempts int
	PollingDuration time.Duration

	// UserAgent, if not empty, will be set as the HTTP User-Agent header on all requests sent
	// through the Do method.
	UserAgent string
}

// NewClientWithUserAgent returns an instance of the DefaultClient with the UserAgent set to the
// passed string.
func NewClientWithUserAgent(ua string) Client {
	c := DefaultClient
	c.UserAgent = ua
	return c
}

// IsPollingAllowed returns an error if the client allows polling and the passed http.Response
// requires it, otherwise it returns nil.
func (c Client) IsPollingAllowed(resp *http.Response, codes ...int) error {
	if c.DoNotPoll() && ResponseRequiresPolling(resp, codes...) {
		return NewErrorWithStatusCode("autorest/Client", "IsPollingAllowed", resp.StatusCode, "Response to %s requires polling but polling is disabled",
			resp.Request.URL)
	}
	return nil
}

// PollAsNeeded is a convenience method that will poll if the passed http.Response requires it.
func (c Client) PollAsNeeded(resp *http.Response, codes ...int) (*http.Response, error) {
	if !ResponseRequiresPolling(resp, codes...) {
		return resp, nil
	}

	if c.DoNotPoll() {
		return resp, NewErrorWithStatusCode("autorest/Client", "PollAsNeeded", resp.StatusCode, "Polling for %s is required, but polling is disabled",
			resp.Request.URL)
	}

	req, err := NewPollingRequest(resp, c)
	if err != nil {
		return resp, NewErrorWithError(err, "autorest/Client", "PollAsNeeded", resp.StatusCode, "Unable to create polling request for response to %s",
			resp.Request.URL)
	}

	Prepare(req,
		c.WithInspection())

	if c.PollForAttempts() {
		return PollForAttempts(c, req, DefaultPollingDelay, c.PollingAttempts, codes...)
	}
	return PollForDuration(c, req, DefaultPollingDelay, c.PollingDuration, codes...)
}

// DoNotPoll returns true if the client should not poll, false otherwise.
func (c Client) DoNotPoll() bool {
	return len(c.PollingMode) == 0 || c.PollingMode == DoNotPoll
}

// PollForAttempts returns true if the PollingMode is set to ForAttempts, false otherwise.
func (c Client) PollForAttempts() bool {
	return c.PollingMode == PollUntilAttempts
}

// PollForDuration return true if the PollingMode is set to ForDuration, false otherwise.
func (c Client) PollForDuration() bool {
	return c.PollingMode == PollUntilDuration
}

// Send sends the passed http.Request after applying authorization. It will poll if the client
// allows polling and the http.Response status code requires it. It will close the http.Response
// Body if the request returns an error.
func (c Client) Send(req *http.Request, codes ...int) (*http.Response, error) {
	if len(codes) == 0 {
		codes = []int{http.StatusOK}
	}

	req, err := Prepare(req,
		c.WithAuthorization(),
		c.WithInspection())
	if err != nil {
		return nil, NewErrorWithError(err, "autorest/Client", "Send", UndefinedStatusCode, "Preparing request failed")
	}

	resp, err := SendWithSender(c, req,
		DoErrorUnlessStatusCode(codes...))
	if err == nil {
		err = c.IsPollingAllowed(resp)
		if err == nil {
			resp, err = c.PollAsNeeded(resp)
		}
	}

	if err != nil {
		Respond(resp,
			ByClosing())
	}

	return resp, err
}

// Do implements the Sender interface by invoking the active Sender. If Sender is not set, it uses
// a new instance of http.Client. In both cases it will, if UserAgent is set, apply set the
// User-Agent header.
func (c Client) Do(r *http.Request) (*http.Response, error) {
	if len(c.UserAgent) > 0 {
		r, _ = Prepare(r, WithUserAgent(c.UserAgent))
	}
	return c.sender().Do(r)
}

// sender returns the Sender to which to send requests.
func (c Client) sender() Sender {
	if c.Sender == nil {
		return http.DefaultClient
	}
	return c.Sender
}

// WithAuthorization is a convenience method that returns the WithAuthorization PrepareDecorator
// from the current Authorizer. If not Authorizer is set, it uses the NullAuthorizer.
func (c Client) WithAuthorization() PrepareDecorator {
	return c.authorizer().WithAuthorization()
}

// authorizer returns the Authorizer to use.
func (c Client) authorizer() Authorizer {
	if c.Authorizer == nil {
		return NullAuthorizer{}
	}
	return c.Authorizer
}

// WithInspection is a convenience method that passes the request to the supplied RequestInspector,
// if present, or returns the WithNothing PrepareDecorator otherwise.
func (c Client) WithInspection() PrepareDecorator {
	if c.RequestInspector == nil {
		return WithNothing()
	}
	return c.RequestInspector
}

// ByInspecting is a convenience method that passes the response to the supplied ResponseInspector,
// if present, or returns the ByIgnoring RespondDecorator otherwise.
func (c Client) ByInspecting() RespondDecorator {
	if c.ResponseInspector == nil {
		return ByIgnoring()
	}
	return c.ResponseInspector
}

// Response serves as the base for all responses from generated clients. It provides access to the
// last http.Response.
type Response struct {
	*http.Response `json:"-"`
}

// GetPollingDelay extracts the polling delay from the Retry-After header of the response. If
// the header is absent or is malformed, it will return the supplied default delay time.Duration.
func (r Response) GetPollingDelay(defaultDelay time.Duration) time.Duration {
	return GetPollingDelay(r.Response, defaultDelay)
}

// GetPollingLocation retrieves the polling URL from the Location header of the response.
func (r Response) GetPollingLocation() string {
	return GetPollingLocation(r.Response)
}
