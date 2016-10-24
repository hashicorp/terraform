package resource

import (
	"time"
)

// StopCh is used to construct a channel that is closed when the provider
// stopCh is closed, or a timeout is reached.
//
// The first return value is the channel that is closed when the operation
// should stop. The second return value should be closed when the operation
// completes so that the stop channel is never closed and to cleanup the
// goroutine watching the channels.
func StopCh(stopCh <-chan struct{}, max time.Duration) (<-chan struct{}, chan<- struct{}) {
	ch := make(chan struct{})
	doneCh := make(chan struct{})

	// If we have a max of 0 then it is unlimited. A nil channel blocks on
	// receive forever so this ensures that behavior.
	var timeCh <-chan time.Time
	if max > 0 {
		timeCh = time.After(max)
	}

	// Start a goroutine to watch the various cases of cancellation
	go func() {
		select {
		case <-doneCh:
			// If we are done, don't trigger the cancel
			return
		case <-timeCh:
		case <-stopCh:
		}

		// Trigger the cancel
		close(ch)
	}()

	return ch, doneCh
}
