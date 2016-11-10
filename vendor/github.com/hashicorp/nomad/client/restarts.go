package client

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	cstructs "github.com/hashicorp/nomad/client/driver/structs"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// jitter is the percent of jitter added to restart delays.
	jitter = 0.25

	ReasonNoRestartsAllowed   = "Policy allows no restarts"
	ReasonUnrecoverableErrror = "Error was unrecoverable"
	ReasonWithinPolicy        = "Restart within policy"
	ReasonDelay               = "Exceeded allowed attempts, applying a delay"
)

func newRestartTracker(policy *structs.RestartPolicy, jobType string) *RestartTracker {
	onSuccess := true
	if jobType == structs.JobTypeBatch {
		onSuccess = false
	}
	return &RestartTracker{
		startTime: time.Now(),
		onSuccess: onSuccess,
		policy:    policy,
		rand:      rand.New(rand.NewSource(time.Now().Unix())),
	}
}

type RestartTracker struct {
	waitRes   *cstructs.WaitResult
	startErr  error
	count     int       // Current number of attempts.
	onSuccess bool      // Whether to restart on successful exit code.
	startTime time.Time // When the interval began
	reason    string    // The reason for the last state
	policy    *structs.RestartPolicy
	rand      *rand.Rand
	lock      sync.Mutex
}

// SetPolicy updates the policy used to determine restarts.
func (r *RestartTracker) SetPolicy(policy *structs.RestartPolicy) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.policy = policy
}

// SetStartError is used to mark the most recent start error. If starting was
// successful the error should be nil.
func (r *RestartTracker) SetStartError(err error) *RestartTracker {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.startErr = err
	return r
}

// SetWaitResult is used to mark the most recent wait result.
func (r *RestartTracker) SetWaitResult(res *cstructs.WaitResult) *RestartTracker {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.waitRes = res
	return r
}

// GetReason returns a human-readable description for the last state returned by
// GetState.
func (r *RestartTracker) GetReason() string {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.reason
}

// GetState returns the tasks next state given the set exit code and start
// error. One of the following states are returned:
// * TaskRestarting - Task should be restarted
// * TaskNotRestarting - Task should not be restarted and has exceeded its
//   restart policy.
// * TaskTerminated - Task has terminated successfully and does not need a
//   restart.
//
// If TaskRestarting is returned, the duration is how long to wait until
// starting the task again.
func (r *RestartTracker) GetState() (string, time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Hot path if no attempts are expected
	if r.policy.Attempts == 0 {
		r.reason = ReasonNoRestartsAllowed
		if r.waitRes != nil && r.waitRes.Successful() {
			return structs.TaskTerminated, 0
		}

		return structs.TaskNotRestarting, 0
	}

	r.count++

	// Check if we have entered a new interval.
	end := r.startTime.Add(r.policy.Interval)
	now := time.Now()
	if now.After(end) {
		r.count = 0
		r.startTime = now
	}

	if r.startErr != nil {
		return r.handleStartError()
	} else if r.waitRes != nil {
		return r.handleWaitResult()
	} else {
		return "", 0
	}
}

// handleStartError returns the new state and potential wait duration for
// restarting the task after it was not successfully started. On start errors,
// the restart policy is always treated as fail mode to ensure we don't
// infinitely try to start a task.
func (r *RestartTracker) handleStartError() (string, time.Duration) {
	// If the error is not recoverable, do not restart.
	if rerr, ok := r.startErr.(*cstructs.RecoverableError); !(ok && rerr.Recoverable) {
		r.reason = ReasonUnrecoverableErrror
		return structs.TaskNotRestarting, 0
	}

	if r.count > r.policy.Attempts {
		if r.policy.Mode == structs.RestartPolicyModeFail {
			r.reason = fmt.Sprintf(
				`Exceeded allowed attempts %d in interval %v and mode is "fail"`,
				r.policy.Attempts, r.policy.Interval)
			return structs.TaskNotRestarting, 0
		} else {
			r.reason = ReasonDelay
			return structs.TaskRestarting, r.getDelay()
		}
	}

	r.reason = ReasonWithinPolicy
	return structs.TaskRestarting, r.jitter()
}

// handleWaitResult returns the new state and potential wait duration for
// restarting the task after it has exited.
func (r *RestartTracker) handleWaitResult() (string, time.Duration) {
	// If the task started successfully and restart on success isn't specified,
	// don't restart but don't mark as failed.
	if r.waitRes.Successful() && !r.onSuccess {
		r.reason = "Restart unnecessary as task terminated successfully"
		return structs.TaskTerminated, 0
	}

	if r.count > r.policy.Attempts {
		if r.policy.Mode == structs.RestartPolicyModeFail {
			r.reason = fmt.Sprintf(
				`Exceeded allowed attempts %d in interval %v and mode is "fail"`,
				r.policy.Attempts, r.policy.Interval)
			return structs.TaskNotRestarting, 0
		} else {
			r.reason = ReasonDelay
			return structs.TaskRestarting, r.getDelay()
		}
	}

	r.reason = ReasonWithinPolicy
	return structs.TaskRestarting, r.jitter()
}

// getDelay returns the delay time to enter the next interval.
func (r *RestartTracker) getDelay() time.Duration {
	end := r.startTime.Add(r.policy.Interval)
	now := time.Now()
	return end.Sub(now)
}

// jitter returns the delay time plus a jitter.
func (r *RestartTracker) jitter() time.Duration {
	// Get the delay and ensure it is valid.
	d := r.policy.Delay.Nanoseconds()
	if d == 0 {
		d = 1
	}

	j := float64(r.rand.Int63n(d)) * jitter
	return time.Duration(d + int64(j))
}

// Returns a tracker that never restarts.
func noRestartsTracker() *RestartTracker {
	policy := &structs.RestartPolicy{Attempts: 0, Mode: structs.RestartPolicyModeFail}
	return newRestartTracker(policy, structs.JobTypeBatch)
}
