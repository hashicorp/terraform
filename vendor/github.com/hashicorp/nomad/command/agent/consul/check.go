package consul

import (
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul/lib"
	cstructs "github.com/hashicorp/nomad/client/driver/structs"
)

// CheckRunner runs a given check in a specific interval and update a
// corresponding Consul TTL check
type CheckRunner struct {
	check    Check
	runCheck func(Check)
	logger   *log.Logger
	stop     bool
	stopCh   chan struct{}
	stopLock sync.Mutex

	started     bool
	startedLock sync.Mutex
}

// NewCheckRunner configures and returns a CheckRunner
func NewCheckRunner(check Check, runCheck func(Check), logger *log.Logger) *CheckRunner {
	cr := CheckRunner{
		check:    check,
		runCheck: runCheck,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
	return &cr
}

// Start is used to start the check. The check runs until stop is called
func (r *CheckRunner) Start() {
	r.startedLock.Lock()
	defer r.startedLock.Unlock()
	if r.started {
		return
	}
	r.stopLock.Lock()
	defer r.stopLock.Unlock()
	go r.run()
	r.started = true
}

// Stop is used to stop the check.
func (r *CheckRunner) Stop() {
	r.stopLock.Lock()
	defer r.stopLock.Unlock()
	if !r.stop {
		r.stop = true
		close(r.stopCh)
	}
}

// run is invoked by a goroutine to run until Stop() is called
func (r *CheckRunner) run() {
	// Get the randomized initial pause time
	initialPauseTime := lib.RandomStagger(r.check.Interval())
	r.logger.Printf("[DEBUG] agent: pausing %v before first invocation of %s", initialPauseTime, r.check.ID())
	next := time.NewTimer(initialPauseTime)
	for {
		select {
		case <-next.C:
			r.runCheck(r.check)
			next.Reset(r.check.Interval())
		case <-r.stopCh:
			next.Stop()
			return
		}
	}
}

// Check is an interface which check providers can implement for Nomad to run
type Check interface {
	Run() *cstructs.CheckResult
	ID() string
	Interval() time.Duration
	Timeout() time.Duration
}
