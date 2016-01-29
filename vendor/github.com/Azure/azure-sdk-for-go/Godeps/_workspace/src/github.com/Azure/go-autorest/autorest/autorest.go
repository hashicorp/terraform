/*
Package autorest implements an HTTP request pipeline suitable for use across multiple go-routines
and provides the shared routines relied on by AutoRest (see https://github.com/Azure/autorest/)
generated Go code.

The package breaks sending and responding to HTTP requests into three phases: Preparing, Sending,
and Responding. A typical pattern is:

  req, err := Prepare(&http.Request{},
    token.WithAuthorization())

  resp, err := Send(req,
    WithLogging(logger),
    DoErrorIfStatusCode(http.StatusInternalServerError),
    DoCloseIfError(),
    DoRetryForAttempts(5, time.Second))

  err = Respond(resp,
    ByClosing())

Each phase relies on decorators to modify and / or manage processing. Decorators may first modify
and then pass the data along, pass the data first and then modify the result, or wrap themselves
around passing the data (such as a logger might do). Decorators run in the order provided. For
example, the following:

  req, err := Prepare(&http.Request{},
    WithBaseURL("https://microsoft.com/"),
    WithPath("a"),
    WithPath("b"),
    WithPath("c"))

will set the URL to:

  https://microsoft.com/a/b/c

Preparers and Responders may be shared and re-used (assuming the underlying decorators support
sharing and re-use). Performant use is obtained by creating one or more Preparers and Responders
shared among multiple go-routines, and a single Sender shared among multiple sending go-routines,
all bound together by means of input / output channels.

Decorators hold their passed state within a closure (such as the path components in the example
above). Be careful to share Preparers and Responders only in a context where such held state
applies. For example, it may not make sense to share a Preparer that applies a query string from a
fixed set of values. Similarly, sharing a Responder that reads the response body into a passed
struct (e.g., ByUnmarshallingJson) is likely incorrect.

Lastly, the Swagger specification (https://swagger.io) that drives AutoRest
(https://github.com/Azure/autorest/) precisely defines two date forms: date and date-time. The
github.com/Azure/go-autorest/autorest/date package provides time.Time derivations to ensure
correct parsing and formatting.

Errors raised by autorest objects and methods will conform to the autorest.Error interface.

See the included examples for more detail. For details on the suggested use of this package by
generated clients, see the Client described below.
*/
package autorest

import (
	"net/http"
	"time"
)

const (
	headerLocation   = "Location"
	headerRetryAfter = "Retry-After"
)

// ResponseHasStatusCode returns true if the status code in the HTTP Response is in the passed set
// and false otherwise.
func ResponseHasStatusCode(resp *http.Response, codes ...int) bool {
	return containsInt(codes, resp.StatusCode)
}

// ResponseRequiresPolling returns true if the passed http.Response requires polling follow-up
// request (as determined by the status code being in the passed set, which defaults to HTTP 202
// Accepted).
func ResponseRequiresPolling(resp *http.Response, codes ...int) bool {
	if resp.StatusCode == http.StatusOK {
		return false
	}

	if len(codes) == 0 {
		codes = []int{http.StatusAccepted}
	}

	return ResponseHasStatusCode(resp, codes...)
}

// NewPollingRequest allocates and returns a new http.Request to poll for the passed response. If
// it successfully creates the request, it will also close the body of the passed response,
// otherwise the body remains open.
func NewPollingRequest(resp *http.Response, authorizer Authorizer) (*http.Request, error) {
	location := GetPollingLocation(resp)
	if location == "" {
		return nil, NewErrorWithStatusCode("autorest", "NewPollingRequest", resp.StatusCode, "Location header missing from response that requires polling")
	}

	req, err := Prepare(&http.Request{},
		AsGet(),
		WithBaseURL(location),
		authorizer.WithAuthorization())
	if err != nil {
		return nil, NewErrorWithError(err, "autorest", "NewPollingRequest", UndefinedStatusCode, "Failure creating poll request to %s", location)
	}

	Respond(resp,
		ByClosing())

	return req, nil
}

// GetPollingDelay extracts the polling delay from the Retry-After header of the passed response. If
// the header is absent or is malformed, it will return the supplied default delay time.Duration.
func GetPollingDelay(resp *http.Response, defaultDelay time.Duration) time.Duration {
	retry := resp.Header.Get(headerRetryAfter)
	if retry == "" {
		return defaultDelay
	}

	d, err := time.ParseDuration(retry + "s")
	if err != nil {
		return defaultDelay
	}

	return d
}

// GetPollingLocation retrieves the polling URL from the Location header of the passed response.
func GetPollingLocation(resp *http.Response) string {
	return resp.Header.Get(headerLocation)
}

// PollForAttempts will retry the passed http.Request until it receives an HTTP status code outside
// the passed set or has made the specified number of attempts. The set of status codes defaults to
// HTTP 202 Accepted.
func PollForAttempts(s Sender, req *http.Request, defaultDelay time.Duration, attempts int, codes ...int) (*http.Response, error) {
	return SendWithSender(
		decorateForPolling(s, defaultDelay, codes...),
		req,
		DoRetryForAttempts(attempts, time.Duration(0)))
}

// PollForDuration will retry the passed http.Request until it receives an HTTP status code outside
// the passed set or the total time meets or exceeds the specified duration. The set of status codes
// defaults to HTTP 202 Accepted.
func PollForDuration(s Sender, req *http.Request, defaultDelay time.Duration, total time.Duration, codes ...int) (*http.Response, error) {
	return SendWithSender(
		decorateForPolling(s, defaultDelay, codes...),
		req,
		DoRetryForDuration(total, time.Duration(0)))
}

func decorateForPolling(s Sender, defaultDelay time.Duration, codes ...int) Sender {
	if len(codes) == 0 {
		codes = []int{http.StatusAccepted}
	}

	return DecorateSender(s,
		AfterRetryDelay(defaultDelay),
		DoErrorIfStatusCode(codes...),
		DoCloseIfError())
}
