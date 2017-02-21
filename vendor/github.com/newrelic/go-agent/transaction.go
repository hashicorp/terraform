package newrelic

import "net/http"

// Transaction represents a request or a background task.
// Each Transaction should only be used in a single goroutine.
type Transaction interface {
	// If StartTransaction is called with a non-nil http.ResponseWriter then
	// the Transaction may be used in its place.  This allows
	// instrumentation of the response code and response headers.
	http.ResponseWriter

	// End finishes the current transaction, stopping all further
	// instrumentation.  Subsequent calls to End will have no effect.
	End() error

	// Ignore ensures that this transaction's data will not be recorded.
	Ignore() error

	// SetName names the transaction.  Transactions will not be grouped
	// usefully if too many unique names are used.
	SetName(name string) error

	// NoticeError records an error.  The first five errors per transaction
	// are recorded (this behavior is subject to potential change in the
	// future).
	NoticeError(err error) error

	// AddAttribute adds a key value pair to the current transaction.  This
	// information is attached to errors, transaction events, and error
	// events.  The key must contain fewer than than 255 bytes.  The value
	// must be a number, string, or boolean.  Attribute configuration is
	// applied (see config.go).
	//
	// For more information, see:
	// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/collect-custom-attributes
	AddAttribute(key string, value interface{}) error

	// StartSegmentNow allows the timing of functions, external calls, and
	// datastore calls.  The segments of each transaction MUST be used in a
	// single goroutine.  Consumers are encouraged to use the
	// `StartSegmentNow` functions which checks if the Transaction is nil.
	// See segments.go
	StartSegmentNow() SegmentStartTime
}
