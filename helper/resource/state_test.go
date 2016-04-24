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

func TestWaitForState_inconsistent_positive(t *testing.T) {
	conf := &StateChangeConf{
		Pending:                   []string{"replicating"},
		Target:                    []string{"done"},
		Refresh:                   InconsistentStateRefreshFunc(),
		Timeout:                   10 * time.Second,
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
	conf := &StateChangeConf{
		Pending:                   []string{"replicating"},
		Target:                    []string{"done"},
		Refresh:                   InconsistentStateRefreshFunc(),
		Timeout:                   10 * time.Second,
		ContinuousTargetOccurence: 4,
	}

	_, err := conf.WaitForState()

	if err == nil && err.Error() != "timeout while waiting for state to become 'done'" {
		t.Fatalf("err: %s", err)
	}
}

func TestWaitForState_timeout(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  []string{"running"},
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

func TestWaitForState_failure(t *testing.T) {
	conf := &StateChangeConf{
		Pending: []string{"pending", "incomplete"},
		Target:  []string{"running"},
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
