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
	Pending []string         // States that are "allowed" and will continue trying
	Refresh StateRefreshFunc // Refreshes the current state
	Target  string           // Target state
	Timeout time.Duration    // The amount of time to wait before timeout
}

type waitResult struct {
	obj interface{}
	err error
}

// WaitForState watches an object and waits for it to achieve the state
// specified in the configuration using the specified Refresh() func,
// waiting the number of seconds specified in the timeout configuration.
func (conf *StateChangeConf) WaitForState() (i interface{}, err error) {
	log.Printf("[DEBUG] Waiting for state to become: %s", conf.Target)

	notfoundTick := 0

	result := make(chan waitResult, 1)

	go func() {
		for tries := 0; ; tries++ {
			// Wait between refreshes
			wait := time.Duration(math.Pow(2, float64(tries))) *
				100 * time.Millisecond
			log.Printf("[TRACE] Waiting %s before next try", wait)
			time.Sleep(wait)

			var currentState string
			i, currentState, err = conf.Refresh()
			if err != nil {
				result <- waitResult{nil, err}
				return
			}

			// If we're waiting for the absense of a thing, then return
			if i == nil && conf.Target == "" {
				result <- waitResult{nil, nil}
				return
			}

			if i == nil {
				// If we didn't find the resource, check if we have been
				// not finding it for awhile, and if so, report an error.
				notfoundTick += 1
				if notfoundTick > 20 {
					result <- waitResult{nil, errors.New("couldn't find resource")}
					return
				}
			} else {
				// Reset the counter for when a resource isn't found
				notfoundTick = 0

				if currentState == conf.Target {
					result <- waitResult{i, nil}
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
					result <- waitResult{nil, fmt.Errorf("unexpected state '%s', wanted target '%s'", currentState, conf.Target)}
					return
				}
			}
		}
	}()

	select {
	case waitResult := <-result:
		err := waitResult.err
		i = waitResult.obj
		return i, err
	case <-time.After(conf.Timeout):
		err := fmt.Errorf("timeout while waiting for state to become '%s'", conf.Target)
		i = nil
		return i, err
	}
}
