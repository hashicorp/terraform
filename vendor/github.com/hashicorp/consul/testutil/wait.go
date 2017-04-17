package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul/consul/structs"
)

type testFn func() (bool, error)
type errorFn func(error)

const (
	baseWait = 1 * time.Millisecond
	maxWait  = 100 * time.Millisecond
)

func WaitForResult(try testFn, fail errorFn) {
	var err error
	wait := baseWait
	for retries := 100; retries > 0; retries-- {
		var success bool
		success, err = try()
		if success {
			time.Sleep(25 * time.Millisecond)
			return
		}

		time.Sleep(wait)
		wait *= 2
		if wait > maxWait {
			wait = maxWait
		}
	}
	fail(err)
}

type rpcFn func(string, interface{}, interface{}) error

func WaitForLeader(t *testing.T, rpc rpcFn, dc string) structs.IndexedNodes {
	var out structs.IndexedNodes
	WaitForResult(func() (bool, error) {
		// Ensure we have a leader and a node registration.
		args := &structs.DCSpecificRequest{
			Datacenter: dc,
		}
		if err := rpc("Catalog.ListNodes", args, &out); err != nil {
			return false, fmt.Errorf("Catalog.ListNodes failed: %v", err)
		}
		if !out.QueryMeta.KnownLeader {
			return false, fmt.Errorf("No leader")
		}
		if out.Index == 0 {
			return false, fmt.Errorf("Consul index is 0")
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("failed to find leader: %v", err)
	})
	return out
}
