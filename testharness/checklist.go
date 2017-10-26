package testharness

import (
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"
)

// A Checklist is the result of evaluating a test specification, describing
// which checks were run and what the result of each check was.
type Checklist []CheckItem

type CheckItem struct {
	Result  CheckResult
	Caption string
	Diags   tfdiags.Diagnostics
}

type CheckResult rune

const (
	Error   CheckResult = 0
	Success CheckResult = '☑'
	Failure CheckResult = '☒'
	Skipped CheckResult = '☐'
)

// A CheckStream is used to accumulate a Checklist in a streaming manner.
//
// The recipient of a CheckStream should write zero or more CheckItems to it
// and then close it to complete the Checklist. Leaving a CheckStream unclosed
// will hang the caller.
type CheckStream struct {
	itemCh chan<- CheckItem
	logCh  chan<- string
}

// NewCheckStream creates and returns a new CheckStream whose results are
// written to the given itemCh.
//
// If logCh is non-nil, it will receive any log messages recorded against
// the CheckStream.
//
// Both channels are closed when the CheckStream is closed. Callers should
// wait for both channels to close before terminating in case there are
// any trailing log messages after the final item.
func NewCheckStream(itemCh chan<- CheckItem, logCh chan<- string) CheckStream {
	return CheckStream{
		itemCh: itemCh,
		logCh:  logCh,
	}
}

// Close marks the end of a CheckStream, allowing a caller that is accumulating
// its results to proceed.
//
// After Close is called, all future calls to Write, Log and Logf will panic.
func (s CheckStream) Close() {
	close(s.itemCh)
	if s.logCh != nil {
		close(s.logCh)
	}
}

// Write records a CheckItem in the receiving CheckStream.
//
// This call may block if the stream consumer is busy.
func (s CheckStream) Write(result CheckItem) {
	s.itemCh <- result
}

// Log records a log message agaisnt the recieving CheckStream.
//
// Log messages are an out-of-band mechanism for passing status information
// to the stream consumer, and should be used (for example) if the producer
// is waiting for an expensive operation to complete, to give the consumer
// (or rather, the consumer's end-user) feedback on why results are delayed.
func (s CheckStream) Log(msg string) {
	if s.logCh != nil {
		s.logCh <- msg
	}
}

// Logf is a variant of Log that provides string formatting. It is a
// convenience wrapper around passing the result of fmt.Sprintf to method Log.
func (s CheckStream) Logf(format string, args ...interface{}) {
	s.Log(fmt.Sprintf(format, args...))
}

// Substream creates a new CheckStream that copies items and logs into the
// receiver, eventually closing the returned "close" Waiter when the
// substream is closed.
//
// The intended purpose of Substream is when one stream producer delegates
// to another for some of its items; in that case, the callee must be able
// to close its stream without closing the caller's stream, and the caller
// must be able to detect when the callee is finished.
func (s CheckStream) Substream() (CheckStream, Waiter) {
	proxyItemCh := make(chan CheckItem)
	proxyLogCh := make(chan string)
	closeCh := make(chan struct{})

	ret := NewCheckStream(proxyItemCh, proxyLogCh)
	go func() {
		for {
			select {
			case item, ok := <-proxyItemCh:
				s.Write(item)
				if !ok {
					proxyItemCh = nil
				}
			case msg, ok := <-proxyLogCh:
				s.Log(msg)
				if !ok {
					proxyLogCh = nil
				}
			}

			if proxyItemCh == nil && proxyLogCh == nil {
				close(closeCh)
				break
			}
		}
	}()

	return ret, Waiter(closeCh)
}
