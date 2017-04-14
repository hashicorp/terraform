package resource

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// StateRefreshFunc is a function type used for StateChangeConf that is
// responsible for refreshing the item being watched for a state change.
//
// It returns three results. `result` is any object that will be returned
// as the final object after waiting for state change. This allows you to
// return the final updated object, for example an EC2 instance after refreshing
// it.
//
// `state` is the latest state of that object. And `err` is any error that
// may have happened while refreshing the state.
type StateRefreshFunc func() (result interface{}, state string, err error)

// StateChangeConf is the configuration struct used for `WaitForState`.
type StateChangeConf struct {
	Delay          time.Duration    // Wait this time before starting checks
	Pending        []string         // States that are "allowed" and will continue trying
	Refresh        StateRefreshFunc // Refreshes the current state
	Target         []string         // Target state
	Timeout        time.Duration    // The amount of time to wait before timeout
	TimeoutGrace   time.Duration    // The grace period to wait for the last refresh to finish before timing out
	MinTimeout     time.Duration    // Smallest time to wait before refreshes
	PollInterval   time.Duration    // Override MinTimeout/backoff and only poll this often
	NotFoundChecks int              // Number of times to allow not found

	// This is to work around inconsistent APIs
	ContinuousTargetOccurence int // Number of times the Target state has to occur continuously
}

// WaitForState watches an object and waits for it to achieve the state
// specified in the configuration using the specified Refresh() func,
// waiting the number of seconds specified in the timeout configuration.
//
// If the Refresh function returns a error, exit immediately with that error.
//
// If the Refresh function returns a state other than the Target state or one
// listed in Pending, return immediately with an error.
//
// If the Timeout is exceeded before reaching the Target state, return an
// error.
//
// Otherwise, result the result of the first call to the Refresh function to
// reach the target state.
func (conf *StateChangeConf) WaitForState() (interface{}, error) {
	log.Printf("[DEBUG] Waiting for state to become: %s", conf.Target)

	notfoundTick := 0
	targetOccurence := 0

	// Set a default for times to check for not found
	if conf.NotFoundChecks == 0 {
		conf.NotFoundChecks = 20
	}

	if conf.ContinuousTargetOccurence == 0 {
		conf.ContinuousTargetOccurence = 1
	}

	// We can't safely read the result values if we timeout, so store them in
	// an atomic.Value
	type Result struct {
		Result interface{}
		State  string
		Error  error
	}
	var lastResult atomic.Value
	lastResult.Store(Result{})
	var refreshMutex sync.Mutex

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		// Wait for the delay
		time.Sleep(conf.Delay)

		wait := 100 * time.Millisecond

		for {
			// Intentionally not deferring the Unlock
			// if this function returns, I don't want to let the timeout
			// code run; it should pick up on the channel closing
			// this does mean, however, that any continue statements MUST call
			// refreshMutex.Unlock() before continue
			refreshMutex.Lock()
			res, currentState, err := conf.Refresh()
			result := Result{
				Result: res,
				State:  currentState,
				Error:  err,
			}
			lastResult.Store(result)

			if err != nil {
				return
			}

			// If we're waiting for the absence of a thing, then return
			if res == nil && len(conf.Target) == 0 {
				targetOccurence += 1
				if conf.ContinuousTargetOccurence == targetOccurence {
					return
				} else {
					refreshMutex.Unlock()
					continue
				}
			}

			if res == nil {
				// If we didn't find the resource, check if we have been
				// not finding it for awhile, and if so, report an error.
				notfoundTick += 1
				if notfoundTick > conf.NotFoundChecks {
					result.Error = &NotFoundError{
						LastError: err,
						Retries:   notfoundTick,
					}
					lastResult.Store(result)
					return
				}
			} else {
				// Reset the counter for when a resource isn't found
				notfoundTick = 0
				found := false

				for _, allowed := range conf.Target {
					if currentState == allowed {
						found = true
						targetOccurence += 1
						if conf.ContinuousTargetOccurence == targetOccurence {
							return
						} else {
							// FIXME: I think this continue is buggy, it's continuing the for
							// loop just above, not continuing on the big for loop to check
							// the resource status?
							// If this is fixed, then uncomment the following line
							// refreshMutex.Unlock()
							continue
						}
					}
				}

				for _, allowed := range conf.Pending {
					if currentState == allowed {
						found = true
						targetOccurence = 0
						break
					}
				}

				if !found && len(conf.Pending) > 0 {
					result.Error = &UnexpectedStateError{
						LastError:     err,
						State:         result.State,
						ExpectedState: conf.Target,
					}
					lastResult.Store(result)
					return
				}
			}

			// If a poll interval has been specified, choose that interval.
			// Otherwise bound the default value.
			if conf.PollInterval > 0 && conf.PollInterval < 180*time.Second {
				wait = conf.PollInterval
			} else {
				if wait < conf.MinTimeout {
					wait = conf.MinTimeout
				} else if wait > 10*time.Second {
					wait = 10 * time.Second
				}
			}

			log.Printf("[TRACE] Waiting %s before next try", wait)
			time.Sleep(wait)

			// Wait between refreshes using exponential backoff, except when
			// waiting for the target state to reoccur.
			if targetOccurence == 0 {
				wait *= 2
			}
			refreshMutex.Unlock()
		}
	}()

	select {
	case <-doneCh:
		r := lastResult.Load().(Result)
		return r.Result, r.Error
	case <-time.After(conf.Timeout):
		// There is special processing to handle the case where conf.Refresh is an
		// asynchronous method that can create resources (e.g., a call to create
		// a cloud resource). It's possible that, when we hit the timeout, the
		// asynchronous process is actually going to succeed and provision a
		// resource, but we'll never see it here. So, we create a mutex around the
		// refresh attempts and then give it a grace period of 5 seconds (chosen
		// somewhat arbitrarily) to finish processing the existing call before we
		// give up. Note that we never actually release the mutex here in order to
		// prevent the refresh function from getting called yet again.
		// The length of the grace period is a tradeoff. If we don't have a grace
		// period, we could leak resources. If we have an infinite grace period,
		// then we might never timeout if conf.Refresh never finishes.
		mutexLockCh := make(chan struct{})
		go func() {
			refreshMutex.Lock()
			close(mutexLockCh)
		}()
		select {
		case <-doneCh:
			r := lastResult.Load().(Result)
			return r.Result, r.Error
		// The mutex is only unlocked at the end of the for loop, i.e., right before
		// retrying. Which means if we succeed in getting this lock, the last
		// attempt didn't succeed, so we should still process it as a timeout.
		case <-mutexLockCh:
			r := lastResult.Load().(Result)
			return nil, &TimeoutError{
				LastError:     r.Error,
				LastState:     r.State,
				Timeout:       conf.Timeout,
				ExpectedState: conf.Target,
			}
		// This case means that the existing call to conf.Refresh took over 5
		// seconds from the time we hit the timeout, so just give up
		case <-time.After(conf.TimeoutGrace):
			r := lastResult.Load().(Result)
			return nil, &TimeoutError{
				LastError:     r.Error,
				LastState:     r.State,
				Timeout:       conf.Timeout,
				ExpectedState: conf.Target,
			}
		}
	}
}
