package webbrowser

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/hashicorp/terraform/httpclient"
)

// NewMockLauncher creates and returns a mock implementation of Launcher,
// with some special behavior designed for use in unit tests.
//
// See the documentation of MockLauncher itself for more information.
func NewMockLauncher(ctx context.Context) *MockLauncher {
	client := httpclient.New()
	return &MockLauncher{
		Client:  client,
		Context: ctx,
	}
}

// MockLauncher is a mock implementation of Launcher that has some special
// behavior designed for use in unit tests.
//
// When OpenURL is called, MockLauncher will make an HTTP request to the given
// URL rather than interacting with a "real" browser.
//
// In normal situations it will then return with no further action, but if
// the response to the given URL is either a standard HTTP redirect response
// or includes the custom HTTP header X-Redirect-To then MockLauncher will
// send a follow-up request to that target URL, and continue in this manner
// until it reaches a URL that is not a redirect. (The X-Redirect-To header
// is there so that a server can potentially offer a normal HTML page to
// an actual browser while also giving a next-hop hint for MockLauncher.)
//
// Since MockLauncher is not a full programmable user-agent implementation
// it can't be used for testing of real-world web applications, but it can
// be used for testing against specialized test servers that are written
// with MockLauncher in mind and know how to drive the request flow through
// whatever steps are required to complete the desired test.
//
// All of the actions taken by MockLauncher happen asynchronously in the
// background, to simulate the concurrency of a separate web browser.
// Test code using MockLauncher should provide a context which is cancelled
// when the test completes, to help avoid leaking MockLaunchers.
type MockLauncher struct {
	// Client is the HTTP client that MockLauncher will use to make requests.
	// By default (if you use NewMockLauncher) this is a new client created
	// via httpclient.New, but callers may override it if they need customized
	// behavior for a particular test.
	//
	// Do not use a client that is shared with any other subsystem, because
	// MockLauncher will customize the settings of the given client.
	Client *http.Client

	// Context can be cancelled in order to abort an OpenURL call before it
	// would naturally complete.
	Context context.Context

	// Responses is a log of all of the responses recieved from the launcher's
	// requests, in the order requested.
	Responses []*http.Response

	// done is a waitgroup used internally to signal when the async work is
	// complete, in order to make this mock more convenient to use in tests.
	done sync.WaitGroup
}

var _ Launcher = (*MockLauncher)(nil)

// OpenURL is the mock implementation of Launcher, which has the special
// behavior described for type MockLauncher.
func (l *MockLauncher) OpenURL(u string) error {
	// We run our operation in the background because it's supposed to be
	// behaving like a web browser running in a separate process.
	log.Printf("[TRACE] webbrowser.MockLauncher: OpenURL(%q) starting in the background", u)
	l.done.Add(1)
	go func() {
		err := l.openURL(u)
		if err != nil {
			// Can't really do anything with this asynchronously, so we'll
			// just log it so that someone debugging will be able to see it.
			log.Printf("[ERROR] webbrowser.MockLauncher: OpenURL(%q): %s", u, err)
		} else {
			log.Printf("[TRACE] webbrowser.MockLauncher: OpenURL(%q) has concluded", u)
		}
		l.done.Done()
	}()
	return nil
}

func (l *MockLauncher) openURL(u string) error {
	// We need to disable automatic redirect following so that we can implement
	// it ourselves below, and thus be able to see the redirects in our
	// responses log.
	l.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// We'll keep looping as long as the server keeps giving us new URLs to
	// request.
	for u != "" {
		log.Printf("[DEBUG] webbrowser.MockLauncher: requesting %s", u)
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return fmt.Errorf("failed to construct HTTP request for %s: %s", u, err)
		}
		resp, err := l.Client.Do(req)
		if err != nil {
			log.Printf("[DEBUG] webbrowser.MockLauncher: request failed: %s", err)
			return fmt.Errorf("error requesting %s: %s", u, err)
		}
		l.Responses = append(l.Responses, resp)
		if resp.StatusCode >= 400 {
			log.Printf("[DEBUG] webbrowser.MockLauncher: request failed: %s", resp.Status)
			return fmt.Errorf("error requesting %s: %s", u, resp.Status)
		}
		log.Printf("[DEBUG] webbrowser.MockLauncher: request succeeded: %s", resp.Status)

		u = "" // unless it's a redirect, we'll stop after this
		if location := resp.Header.Get("Location"); location != "" {
			u = location
		} else if redirectTo := resp.Header.Get("X-Redirect-To"); redirectTo != "" {
			u = redirectTo
		}

		if u != "" {
			// HTTP technically doesn't permit relative URLs in Location, but
			// browsers tolerate it and so real-world servers do it, and thus
			// we'll allow it here too.
			oldURL := resp.Request.URL
			givenURL, err := url.Parse(u)
			if err != nil {
				return fmt.Errorf("invalid redirect URL %s: %s", u, err)
			}
			u = oldURL.ResolveReference(givenURL).String()
			log.Printf("[DEBUG] webbrowser.MockLauncher: redirected to %s", u)
		}
	}

	log.Printf("[DEBUG] webbrowser.MockLauncher: all done")
	return nil
}

// Wait blocks until the MockLauncher has finished its asynchronous work of
// making HTTP requests and following redirects, at which point it will have
// reached a request that didn't redirect anywhere and stopped iterating.
func (l *MockLauncher) Wait() {
	log.Printf("[TRACE] webbrowser.MockLauncher: Wait() for current work to complete")
	l.done.Wait()
}
