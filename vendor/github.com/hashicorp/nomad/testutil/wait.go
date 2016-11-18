package testutil

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// TravisRunEnv is an environment variable that is set if being run by
	// Travis.
	TravisRunEnv = "CI"
)

type testFn func() (bool, error)
type errorFn func(error)

func WaitForResult(test testFn, error errorFn) {
	WaitForResultRetries(2000*TestMultiplier(), test, error)
}

func WaitForResultRetries(retries int64, test testFn, error errorFn) {
	for retries > 0 {
		time.Sleep(10 * time.Millisecond)
		retries--

		success, err := test()
		if success {
			return
		}

		if retries == 0 {
			error(err)
		}
	}
}

// TestMultiplier returns a multiplier for retries and waits given environment
// the tests are being run under.
func TestMultiplier() int64 {
	if IsTravis() {
		return 3
	}

	return 1
}

func IsTravis() bool {
	_, ok := os.LookupEnv(TravisRunEnv)
	return ok
}

type rpcFn func(string, interface{}, interface{}) error

func WaitForLeader(t *testing.T, rpc rpcFn) {
	WaitForResult(func() (bool, error) {
		args := &structs.GenericRequest{}
		var leader string
		err := rpc("Status.Leader", args, &leader)
		return leader != "", err
	}, func(err error) {
		t.Fatalf("failed to find leader: %v", err)
	})
}
