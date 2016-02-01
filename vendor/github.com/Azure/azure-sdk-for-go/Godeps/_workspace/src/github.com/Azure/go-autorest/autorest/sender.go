package autorest

import (
	"log"
	"math"
	"net/http"
	"time"
)

// Sender is the interface that wraps the Do method to send HTTP requests.
//
// The standard http.Client conforms to this interface.
type Sender interface {
	Do(*http.Request) (*http.Response, error)
}

// SenderFunc is a method that implements the Sender interface.
type SenderFunc func(*http.Request) (*http.Response, error)

// Do implements the Sender interface on SenderFunc.
func (sf SenderFunc) Do(r *http.Request) (*http.Response, error) {
	return sf(r)
}

// SendDecorator takes and possibily decorates, by wrapping, a Sender. Decorators may affect the
// http.Request and pass it along or, first, pass the http.Request along then react to the
// http.Response result.
type SendDecorator func(Sender) Sender

// CreateSender creates, decorates, and returns, as a Sender, the default http.Client.
func CreateSender(decorators ...SendDecorator) Sender {
	return DecorateSender(&http.Client{}, decorators...)
}

// DecorateSender accepts a Sender and a, possibly empty, set of SendDecorators, which is applies to
// the Sender. Decorators are applied in the order received, but their affect upon the request
// depends on whether they are a pre-decorator (change the http.Request and then pass it along) or a
// post-decorator (pass the http.Request along and react to the results in http.Response).
func DecorateSender(s Sender, decorators ...SendDecorator) Sender {
	for _, decorate := range decorators {
		s = decorate(s)
	}
	return s
}

// Send sends, by means of the default http.Client, the passed http.Request, returning the
// http.Response and possible error. It also accepts a, possibly empty, set of SendDecorators which
// it will apply the http.Client before invoking the Do method.
//
// Send is a convenience method and not recommended for production. Advanced users should use
// SendWithSender, passing and sharing their own Sender (e.g., instance of http.Client).
//
// Send will not poll or retry requests.
func Send(r *http.Request, decorators ...SendDecorator) (*http.Response, error) {
	return SendWithSender(&http.Client{}, r, decorators...)
}

// SendWithSender sends the passed http.Request, through the provided Sender, returning the
// http.Response and possible error. It also accepts a, possibly empty, set of SendDecorators which
// it will apply the http.Client before invoking the Do method.
//
// SendWithSender will not poll or retry requests.
func SendWithSender(s Sender, r *http.Request, decorators ...SendDecorator) (*http.Response, error) {
	return DecorateSender(s, decorators...).Do(r)
}

// AfterDelay returns a SendDecorator that delays for the passed time.Duration before
// invoking the Sender.
func AfterDelay(d time.Duration) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			time.Sleep(d)
			return s.Do(r)
		})
	}
}

// AfterRetryDelay returns a SendDecorator that delays for the number of seconds specified in the
// Retry-After header of the prior response when polling is required.
func AfterRetryDelay(defaultDelay time.Duration, codes ...int) SendDecorator {
	delay := time.Duration(0)
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			if delay > time.Duration(0) {
				time.Sleep(delay)
			}
			resp, err := s.Do(r)
			if ResponseRequiresPolling(resp, codes...) {
				delay = GetPollingDelay(resp, defaultDelay)
			} else {
				delay = time.Duration(0)
			}
			return resp, err
		})
	}
}

// AsIs returns a SendDecorator that invokes the passed Sender without modifying the http.Request.
func AsIs() SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			return s.Do(r)
		})
	}
}

// WithLogging returns a SendDecorator that implements simple before and after logging of the
// request.
func WithLogging(logger *log.Logger) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			logger.Printf("Sending %s %s\n", r.Method, r.URL)
			resp, err := s.Do(r)
			logger.Printf("%s %s received %s\n", r.Method, r.URL, resp.Status)
			return resp, err
		})
	}
}

// DoCloseIfError returns a SendDecorator that first invokes the passed Sender after which
// it closes the response if the passed Sender returns an error and the response body exists.
func DoCloseIfError() SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := s.Do(r)
			if err != nil {
				Respond(resp, ByClosing())
			}
			return resp, err
		})
	}
}

// DoErrorIfStatusCode returns a SendDecorator that emits an error if the response StatusCode is
// among the set passed. Since these are artificial errors, the response body may still require
// closing.
func DoErrorIfStatusCode(codes ...int) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := s.Do(r)
			if err == nil && ResponseHasStatusCode(resp, codes...) {
				err = NewErrorWithStatusCode("autorest", "DoErrorIfStatusCode", resp.StatusCode, "%v %v failed with %s",
					resp.Request.Method,
					resp.Request.URL,
					resp.Status)
			}
			return resp, err
		})
	}
}

// DoErrorUnlessStatusCode returns a SendDecorator that emits an error unless the response
// StatusCode is among the set passed. Since these are artificial errors, the response body
// may still require closing.
func DoErrorUnlessStatusCode(codes ...int) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (*http.Response, error) {
			resp, err := s.Do(r)
			if err == nil && !ResponseHasStatusCode(resp, codes...) {
				err = NewErrorWithStatusCode("autorest", "DoErrorUnlessStatusCode", resp.StatusCode, "%v %v failed with %s",
					resp.Request.Method,
					resp.Request.URL,
					resp.Status)
			}
			return resp, err
		})
	}
}

// DoRetryForAttempts returns a SendDecorator that retries the request for up to the specified
// number of attempts, exponentially backing off between requests using the supplied backoff
// time.Duration (which may be zero).
func DoRetryForAttempts(attempts int, backoff time.Duration) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (resp *http.Response, err error) {
			for attempt := 0; attempt < attempts; attempt++ {
				resp, err = s.Do(r)
				if err == nil {
					return resp, err
				}
				DelayForBackoff(backoff, attempt)
			}
			return resp, err
		})
	}
}

// DoRetryForDuration returns a SendDecorator that retries the request until the total time is equal
// to or greater than the specified duration, exponentially backing off between requests using the
// supplied backoff time.Duration (which may be zero).
func DoRetryForDuration(d time.Duration, backoff time.Duration) SendDecorator {
	return func(s Sender) Sender {
		return SenderFunc(func(r *http.Request) (resp *http.Response, err error) {
			end := time.Now().Add(d)
			for attempt := 0; time.Now().Before(end); attempt++ {
				resp, err = s.Do(r)
				if err == nil {
					return resp, err
				}
				DelayForBackoff(backoff, attempt)
			}
			return resp, err
		})
	}
}

// DelayForBackoff invokes time.Sleep for the supplied backoff duration raised to the power of
// passed attempt (i.e., an exponential backoff delay). Backoff may be zero.
func DelayForBackoff(backoff time.Duration, attempt int) {
	time.Sleep(time.Duration(math.Pow(float64(backoff), float64(attempt))))
}
