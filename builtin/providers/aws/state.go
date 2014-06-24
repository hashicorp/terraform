package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/goamz/ec2"
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
	Pending []string
	Refresh StateRefreshFunc
	Target  string
}

// InstanceStateRefreshFunc returns a StateRefreshFunc that is used to watch
// an EC2 instance.
func InstanceStateRefreshFunc(conn *ec2.EC2, i *ec2.Instance) StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.Instances([]string{i.InstanceId}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
				// Set this to nil as if we didn't find anything.
				resp = nil
			} else {
				log.Printf("Error on InstanceStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil || len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		i = &resp.Reservations[0].Instances[0]
		return i, i.State.Name, nil
	}
}

// WaitForState watches an object and waits for it to achieve a certain
// state.
func WaitForState(conf *StateChangeConf) (i interface{}, err error) {
	log.Printf("Waiting for state to become: %s", conf.Target)

	notfoundTick := 0

	for {
		var currentState string
		i, currentState, err = conf.Refresh()
		if err != nil {
			return
		}

		if i == nil {
			// If we didn't find the resource, check if we have been
			// not finding it for awhile, and if so, report an error.
			notfoundTick += 1
			if notfoundTick > 20 {
				return nil, errors.New("couldn't find resource")
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
				fmt.Errorf("unexpected state '%s', wanted target '%s'", currentState, conf.Target)
				return
			}
		}

		time.Sleep(2 * time.Second)
	}

	return
}
