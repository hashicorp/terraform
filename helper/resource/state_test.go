package resource

import (
	"errors"
	"strings"
	"sync/atomic"
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

type StateGenerator struct {
	position      int
	stateSequence []string
}

func (r *StateGenerator) NextState() (int, string, error) {
	p, v := r.position, ""
	if len(r.stateSequence)-1 >= p {
		v = r.stateSequence[p]
	} else {
		return -1, "", errors.New("No more states available")
	}

	r.position += 1

	return p, v, nil
}

func NewStateGenerator(sequence []string) *StateGenerator {
	r := &StateGenerator{}
	r.stateSequence = sequence

	return r
}

func InconsistentStateRefreshFunc() StateRefreshFunc {
	sequence := []string{
		"done", "replicating",
		"done", "done", "done",
		"replicating",
		"done", "done", "done",
	}

	r := NewStateGenerator(sequence)

	return func() (interface{}, string, error) {
		idx, s, err := r.NextState()
		if err != nil {
			return nil, "", err
		}

		return idx, s, nil
	}
}

func UnknownPendingStateRefreshFunc() StateRefreshFunc {
	sequence := []string{
		"unknown1", "unknown2", "done",
	}

	r := NewStateGenerator(sequence)

	return func() (interface{}, string, error) {
		idx, s, err := r.NextState()
		if err != nil {
			return nil, "", err
		}

		return idx, s, nil
	}
}

func TestWaitForState_inconsistent_positive(t *testing.T) {
	conf := &StateChangeConf{
		Pending:                   []string{"replicating"},
		Target:                    []string{"done"},
		Refresh:                   InconsistentStateRefreshFunc(),
		Timeout:                   90 * time.Millisecond,
		PollInterval:              10 * time.Millisecond,
		ContinuousTargetOccurence: 3,
	}

	idx, err := conf.WaitForState()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if idx != 4 {
		t.Fatalf("Expected index 4, given %d", idx.(int))
	}
}

func TestWaitForState_inconsistent_negative(t *testing.T) {
	refreshCount := int64(0)
	f := InconsistentStateRefreshFunc()
	refresh := func() (interface{}, string, error) {
		atomic.AddInt64(&refreshCount, 1)
		return f()
	}

	conf := &StateChangeConf{
		Pending:                   []string{"replicating"},
		Target:                    []string{"done"},
		Refresh:                   refresh,
		Timeout:                   85 * time.Millisecond,
		PollInterval:              10 * time.Millisecond,
		ContinuousTargetOccurence: 4,
	}

	_, err := conf.WaitForState()

	if err == nil {
		t.Fatal("Expected timeout error. No error returned.")
	}

	// we can't guarantee the exact number of refresh calls in the tests by
	// timing them, but we want to make sure the test at least went through th
	// required states.
	if atomic.LoadInt64(&refreshCount) < 6 {
		t.Fatal("refreshed called too few times")
	}

	expectedErr := "timeout while waiting for state to become 'done'"
	if !strings.HasPrefix(err.Error(), expectedErr) {
		t.Fatalf("error prefix doesn't match.\nExpected: %q\nGiven: %q\n", expectedErr, err.Error())
	}
}

func TestWaitForState_timeout(t *testing.T) {
	old := refreshGracePeriod
	refreshGracePeriod = 5 * time.Millisecond
	defer func() {
		refreshGracePeriod = old
	}()

	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  []string{"running"},
		Refresh: TimeoutStateRefreshFunc(),
		Timeout: 1 * time.Millisecond,
	}

	obj, err := conf.WaitForState()

	if err == nil {
		t.Fatal("Expected timeout error. No error returned.")
	}

	expectedErr := "timeout while waiting for state to become 'running' (timeout: 1ms)"
	if err.Error() != expectedErr {
		t.Fatalf("Errors don't match.\nExpected: %q\nGiven: %q\n", expectedErr, err.Error())
	}

	if obj != nil {
		t.Fatalf("should not return obj")
	}
}

// Make sure a timeout actually cancels the refresh goroutine and waits for its
// return.
func TestWaitForState_cancel(t *testing.T) {
	// make this refresh func block until we cancel it
	cancel := make(chan struct{})
	refresh := func() (interface{}, string, error) {
		<-cancel
		return nil, "pending", nil
	}
	conf := &StateChangeConf{
		Pending:      []string{"pending", "incomplete"},
		Target:       []string{"running"},
		Refresh:      refresh,
		Timeout:      10 * time.Millisecond,
		PollInterval: 10 * time.Second,
	}

	var obj interface{}
	var err error

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		obj, err = conf.WaitForState()
	}()

	// make sure WaitForState is blocked
	select {
	case <-waitDone:
		t.Fatal("WaitForState returned too early")
	case <-time.After(10 * time.Millisecond):
	}

	// unlock the refresh function
	close(cancel)
	// make sure WaitForState returns
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("WaitForState didn't return after refresh finished")
	}

	if err == nil {
		t.Fatal("Expected timeout error. No error returned.")
	}

	expectedErr := "timeout while waiting for state to become 'running'"
	if !strings.HasPrefix(err.Error(), expectedErr) {
		t.Fatalf("Errors don't match.\nExpected: %q\nGiven: %q\n", expectedErr, err.Error())
	}

	if obj != nil {
		t.Fatalf("should not return obj")
	}

}

func TestWaitForState_success(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  []string{"running"},
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

func TestWaitForState_successUnknownPending(t *testing.T) {
	conf := &StateChangeConf{
		Target:  []string{"done"},
		Refresh: UnknownPendingStateRefreshFunc(),
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
		Target:  []string{},
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

func TestWaitForState_failureEmpty(t *testing.T) {
	conf := &StateChangeConf{
		Pending:        []string{"pending", "incomplete"},
		Target:         []string{},
		NotFoundChecks: 1,
		Refresh: func() (interface{}, string, error) {
			return 42, "pending", nil
		},
		PollInterval: 10 * time.Millisecond,
		Timeout:      100 * time.Millisecond,
	}

	_, err := conf.WaitForState()
	if err == nil {
		t.Fatal("Expected timeout error. Got none.")
	}
	expectedErr := "timeout while waiting for resource to be gone (last state: 'pending', timeout: 100ms)"
	if err.Error() != expectedErr {
		t.Fatalf("Errors don't match.\nExpected: %q\nGiven: %q\n", expectedErr, err.Error())
	}
}

func TestWaitForState_failure(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  []string{"running"},
		Refresh: FailedStateRefreshFunc(),
		Timeout: 200 * time.Second,
	}

	obj, err := conf.WaitForState()
	if err == nil {
		t.Fatal("Expected error. No error returned.")
	}
	expectedErr := "failed"
	if err.Error() != expectedErr {
		t.Fatalf("Errors don't match.\nExpected: %q\nGiven: %q\n", expectedErr, err.Error())
	}
	if obj != nil {
		t.Fatalf("should not return obj")
	}
}
