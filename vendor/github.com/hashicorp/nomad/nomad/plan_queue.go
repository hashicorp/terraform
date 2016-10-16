package nomad

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/nomad/nomad/structs"
)

var (
	// planQueueFlushed is the error used for all pending plans
	// when the queue is flushed or disabled
	planQueueFlushed = fmt.Errorf("plan queue flushed")
)

// PlanFuture is used to return a future for an enqueue
type PlanFuture interface {
	Wait() (*structs.PlanResult, error)
}

// PlanQueue is used to submit commit plans for task allocations
// to the current leader. The leader verifies that resources are not
// over-committed and commits to Raft. This allows sub-schedulers to
// be optimistically concurrent. In the case of an overcommit, the plan
// may be partially applied if allowed, or completely rejected (gang commit).
type PlanQueue struct {
	enabled bool
	stats   *QueueStats

	ready  PendingPlans
	waitCh chan struct{}

	l sync.RWMutex
}

// NewPlanQueue is used to construct and return a new plan queue
func NewPlanQueue() (*PlanQueue, error) {
	q := &PlanQueue{
		enabled: false,
		stats:   new(QueueStats),
		ready:   make([]*pendingPlan, 0, 16),
		waitCh:  make(chan struct{}, 1),
	}
	return q, nil
}

// pendingPlan is used to wrap a plan that is enqueued
// so that we can re-use it as a future.
type pendingPlan struct {
	plan        *structs.Plan
	enqueueTime time.Time
	result      *structs.PlanResult
	errCh       chan error
}

// Wait is used to block for the plan result or potential error
func (p *pendingPlan) Wait() (*structs.PlanResult, error) {
	err := <-p.errCh
	return p.result, err
}

// respond is used to set the response and error for the future
func (p *pendingPlan) respond(result *structs.PlanResult, err error) {
	p.result = result
	p.errCh <- err
}

// PendingPlans is a list of waiting plans.
// We implement the container/heap interface so that this is a
// priority queue
type PendingPlans []*pendingPlan

// Enabled is used to check if the queue is enabled.
func (q *PlanQueue) Enabled() bool {
	q.l.RLock()
	defer q.l.RUnlock()
	return q.enabled
}

// SetEnabled is used to control if the queue is enabled. The queue
// should only be enabled on the active leader.
func (q *PlanQueue) SetEnabled(enabled bool) {
	q.l.Lock()
	q.enabled = enabled
	q.l.Unlock()
	if !enabled {
		q.Flush()
	}
}

// Enqueue is used to enqueue a plan
func (q *PlanQueue) Enqueue(plan *structs.Plan) (PlanFuture, error) {
	q.l.Lock()
	defer q.l.Unlock()

	// Do nothing if not enabled
	if !q.enabled {
		return nil, fmt.Errorf("plan queue is disabled")
	}

	// Wrap the pending plan
	pending := &pendingPlan{
		plan:        plan,
		enqueueTime: time.Now(),
		errCh:       make(chan error, 1),
	}

	// Push onto the heap
	heap.Push(&q.ready, pending)

	// Update the stats
	q.stats.Depth += 1

	// Unblock any blocked reader
	select {
	case q.waitCh <- struct{}{}:
	default:
	}
	return pending, nil
}

// Dequeue is used to perform a blocking dequeue
func (q *PlanQueue) Dequeue(timeout time.Duration) (*pendingPlan, error) {
SCAN:
	q.l.Lock()

	// Do nothing if not enabled
	if !q.enabled {
		q.l.Unlock()
		return nil, fmt.Errorf("plan queue is disabled")
	}

	// Look for available work
	if len(q.ready) > 0 {
		raw := heap.Pop(&q.ready)
		pending := raw.(*pendingPlan)
		q.stats.Depth -= 1
		q.l.Unlock()
		return pending, nil
	}
	q.l.Unlock()

	// Setup the timeout timer
	var timerCh <-chan time.Time
	if timerCh == nil && timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timerCh = timer.C
	}

	// Wait for timeout or new work
	select {
	case <-q.waitCh:
		goto SCAN
	case <-timerCh:
		return nil, nil
	}
}

// Flush is used to reset the state of the plan queue
func (q *PlanQueue) Flush() {
	q.l.Lock()
	defer q.l.Unlock()

	// Error out all the futures
	for _, pending := range q.ready {
		pending.respond(nil, planQueueFlushed)
	}

	// Reset the broker
	q.stats.Depth = 0
	q.ready = make([]*pendingPlan, 0, 16)

	// Unblock any waiters
	select {
	case q.waitCh <- struct{}{}:
	default:
	}
}

// Stats is used to query the state of the queue
func (q *PlanQueue) Stats() *QueueStats {
	// Allocate a new stats struct
	stats := new(QueueStats)

	q.l.RLock()
	defer q.l.RUnlock()

	// Copy all the stats
	*stats = *q.stats
	return stats
}

// EmitStats is used to export metrics about the broker while enabled
func (q *PlanQueue) EmitStats(period time.Duration, stopCh chan struct{}) {
	for {
		select {
		case <-time.After(period):
			stats := q.Stats()
			metrics.SetGauge([]string{"nomad", "plan", "queue_depth"}, float32(stats.Depth))

		case <-stopCh:
			return
		}
	}
}

// QueueStats returns all the stats about the plan queue
type QueueStats struct {
	Depth int
}

// Len is for the sorting interface
func (p PendingPlans) Len() int {
	return len(p)
}

// Less is for the sorting interface. We flip the check
// so that the "min" in the min-heap is the element with the
// highest priority. For the same priority, we use the enqueue
// time of the evaluation to give a FIFO ordering.
func (p PendingPlans) Less(i, j int) bool {
	if p[i].plan.Priority != p[j].plan.Priority {
		return !(p[i].plan.Priority < p[j].plan.Priority)
	}
	return p[i].enqueueTime.Before(p[j].enqueueTime)
}

// Swap is for the sorting interface
func (p PendingPlans) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// Push is used to add a new evalution to the slice
func (p *PendingPlans) Push(e interface{}) {
	*p = append(*p, e.(*pendingPlan))
}

// Pop is used to remove an evaluation from the slice
func (p *PendingPlans) Pop() interface{} {
	n := len(*p)
	e := (*p)[n-1]
	(*p)[n-1] = nil
	*p = (*p)[:n-1]
	return e
}

// Peek is used to peek at the next element that would be popped
func (p PendingPlans) Peek() *pendingPlan {
	n := len(p)
	if n == 0 {
		return nil
	}
	return p[n-1]
}
