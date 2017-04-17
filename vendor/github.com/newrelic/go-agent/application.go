package newrelic

import (
	"net/http"
	"time"
)

// Application represents your application.
type Application interface {
	// StartTransaction begins a Transaction.
	// * The Transaction should only be used in a single goroutine.
	// * This method never returns nil.
	// * If an http.Request is provided then the Transaction is considered
	//   a web transaction.
	// * If an http.ResponseWriter is provided then the Transaction can be
	//   used in its place.  This allows instrumentation of the response
	//   code and response headers.
	StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction

	// RecordCustomEvent adds a custom event to the application.  This
	// feature is incompatible with high security mode.
	//
	// eventType must consist of alphanumeric characters, underscores, and
	// colons, and must contain fewer than 255 bytes.
	//
	// Each value in the params map must be a number, string, or boolean.
	// Keys must be less than 255 bytes.  The params map may not contain
	// more than 64 attributes.  For more information, and a set of
	// restricted keywords, see:
	//
	// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
	RecordCustomEvent(eventType string, params map[string]interface{}) error

	// WaitForConnection blocks until the application is connected, is
	// incapable of being connected, or the timeout has been reached.  This
	// method is useful for short-lived processes since the application will
	// not gather data until it is connected.  nil is returned if the
	// application is connected successfully.
	WaitForConnection(timeout time.Duration) error

	// Shutdown flushes data to New Relic's servers and stops all
	// agent-related goroutines managing this application.  After Shutdown
	// is called, the application is disabled and no more data will be
	// collected.  This method will block until all final data is sent to
	// New Relic or the timeout has elapsed.
	Shutdown(timeout time.Duration)
}

// NewApplication creates an Application and spawns goroutines to manage the
// aggregation and harvesting of data.  On success, a non-nil Application and a
// nil error are returned. On failure, a nil Application and a non-nil error
// are returned.
//
// Applications do not share global state (other than the shared log.Logger).
// Therefore, it is safe to create multiple applications.
func NewApplication(c Config) (Application, error) {
	return newApp(c)
}
