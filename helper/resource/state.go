package resource

import (
	"errors"
	"fmt"
	"log"
	"math"
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
	Target         string           // Target state
	Timeout        time.Duration    // The amount of time to wait before timeout
	MinTimeout     time.Duration    // Smallest time to wait before refreshes
	NotFoundChecks int              // Number of times to allow not found
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

	// Set a default for times to check for not found
	if conf.NotFoundChecks == 0 {
		conf.NotFoundChecks = 20
	}

	var result interface{}
	var resulterr error

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		// Wait for the delay
		time.Sleep(conf.Delay)

		var err error
		for tries := 0; ; tries++ {
			// Wait between refreshes using an exponential backoff
			wait := time.Duration(math.Pow(2, float64(tries))) *
				100 * time.Millisecond
			if wait < conf.MinTimeout {
				wait = conf.MinTimeout
			} else if wait > 10*time.Second {
				wait = 10 * time.Second
			}

			log.Printf("[TRACE] Waiting %s before next try", wait)
			time.Sleep(wait)

			var currentState string
			result, currentState, err = conf.Refresh()
			if err != nil {
				resulterr = err
				return
			}

			// If we're waiting for the absence of a thing, then return
			if result == nil && conf.Target == "" {
				return
			}

			if result == nil {
				// If we didn't find the resource, check if we have been
				// not finding it for awhile, and if so, report an error.
				notfoundTick += 1
				if notfoundTick > conf.NotFoundChecks {
					resulterr = errors.New("couldn't find resource")
					return
				}
			} else {
				// Reset the counter for when a resource isn't found
				notfoundTick = 0

				if currentState == conf.Target {
					return
				}

				found := false
				for _, allowed := range conf.Pending {
					if currentState == allowed {
						found = true
						break
					}
				}

				if !found {
					resulterr = fmt.Errorf(
						"unexpected state '%s', wanted target '%s'",
						currentState,
						conf.Target)
					return
				}
			}
		}
	}()

	select {
	case <-doneCh:
		return result, resulterr
	case <-time.After(conf.Timeout):
		return nil, fmt.Errorf(
			"timeout while waiting for state to become '%s'",
			conf.Target)
	}
}
