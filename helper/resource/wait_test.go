package resource

import (
	"errors"
	"testing"
	"time"
)

func FailedStateRefreshFunc() StateRefreshFunc {
	return func() (interface{}, string, error) {
		return nil, "", errors.New("failed")
	}
}

func TimeoutStateRefreshFunc() StateRefreshFunc {
	return func() (interface{}, string, error) {
		time.Sleep(100 * time.Second)
		return nil, "", errors.New("failed")
	}
}

func SuccessfulStateRefreshFunc() StateRefreshFunc {
	return func() (interface{}, string, error) {
		return struct{}{}, "running", nil
	}
}

func TestWaitForState_timeout(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  "running",
		Refresh: TimeoutStateRefreshFunc(),
		Timeout: 1 * time.Millisecond,
	}

	obj, err := conf.WaitForState()

	if err == nil && err.Error() != "timeout while waiting for state to become 'running'" {
		t.Fatalf("err: %s", err)
	}

	if obj != nil {
		t.Fatalf("should not return obj")
	}

}

func TestWaitForState_success(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  "running",
		Refresh: SuccessfulStateRefreshFunc(),
		Timeout: 200 * time.Second,
	}

	obj, err := conf.WaitForState()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if obj == nil {
		t.Fatalf("should return obj")
	}
}

func TestWaitForState_successEmpty(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  "",
		Refresh: func() (interface{}, string, error) {
			return nil, "", nil
		},
		Timeout: 200 * time.Second,
	}

	obj, err := conf.WaitForState()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if obj != nil {
		t.Fatalf("obj should be nil")
	}
}

func TestWaitForState_failure(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  "running",
		Refresh: FailedStateRefreshFunc(),
		Timeout: 200 * time.Second,
	}

	obj, err := conf.WaitForState()
	if err == nil && err.Error() != "failed" {
		t.Fatalf("err: %s", err)
	}
	if obj != nil {
		t.Fatalf("should not return obj")
	}
}
