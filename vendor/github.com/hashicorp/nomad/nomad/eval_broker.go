package nomad

import (
	"container/heap"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// failedQueue is the queue we add Evaluations to once
	// they've reached the deliveryLimit. This allows the leader to
	// set the status to failed.
	failedQueue = "_failed"
)

var (
	// ErrNotOutstanding is returned if an evaluation is not outstanding
	ErrNotOutstanding = errors.New("evaluation is not outstanding")

	// ErrTokenMismatch is the outstanding eval has a different token
	ErrTokenMismatch = errors.New("evaluation token does not match")

	// ErrNackTimeoutReached is returned if an expired evaluation is reset
	ErrNackTimeoutReached = errors.New("evaluation nack timeout reached")
)

// EvalBroker is used to manage brokering of evaluations. When an evaluation is
// created, due to a change in a job specification or a node, we put it into the
// broker. The broker sorts by evaluations by priority and scheduler type. This
// allows us to dequeue the highest priority work first, while also allowing sub-schedulers
// to only dequeue work they know how to handle. The broker is designed to be entirely
// in-memory and is managed by the leader node.
//
// The broker must provide at-least-once delivery semantics. It relies on explicit
// Ack/Nack messages to handle this. If a delivery is not Ack'd in a sufficient time
// span, it will be assumed Nack'd.
type EvalBroker struct {
	nackTimeout   time.Duration
	deliveryLimit int

	enabled bool
	stats   *BrokerStats

	// evals tracks queued evaluations by ID to de-duplicate enqueue.
	// The counter is the number of times we've attempted delivery,
	// and is used to eventually fail an evaluation.
	evals map[string]int

	// jobEvals tracks queued evaluations by JobID to serialize them
	jobEvals map[string]string

	// blocked tracks the blocked evaluations by JobID in a priority queue
	blocked map[string]PendingEvaluations

	// ready tracks the ready jobs by scheduler in a priority queue
	ready map[string]PendingEvaluations

	// unack is a map of evalID to an un-acknowledged evaluation
	unack map[string]*unackEval

	// waiting is used to notify on a per-scheduler basis of ready work
	waiting map[string]chan struct{}

	// requeue tracks evaluations that need to be re-enqueued once the current
	// evaluation finishes by token. If the token is Nacked or rejected the
	// evaluation is dropped but if Acked successfully, the evaluation is
	// queued.
	requeue map[string]*structs.Evaluation

	// timeWait has evaluations that are waiting for time to elapse
	timeWait map[string]*time.Timer

	l sync.RWMutex
}

// unackEval tracks an unacknowledged evaluation along with the Nack timer
type unackEval struct {
	Eval      *structs.Evaluation
	Token     string
	NackTimer *time.Timer
}

// PendingEvaluations is a list of waiting evaluations.
// We implement the container/heap interface so that this is a
// priority queue
type PendingEvaluations []*structs.Evaluation

// NewEvalBroker creates a new evaluation broker. This is parameterized
// with the timeout used for messages that are not acknowledged before we
// assume a Nack and attempt to redeliver as well as the deliveryLimit
// which prevents a failing eval from being endlessly delivered.
func NewEvalBroker(timeout time.Duration, deliveryLimit int) (*EvalBroker, error) {
	if timeout < 0 {
		return nil, fmt.Errorf("timeout cannot be negative")
	}
	b := &EvalBroker{
		nackTimeout:   timeout,
		deliveryLimit: deliveryLimit,
		enabled:       false,
		stats:         new(BrokerStats),
		evals:         make(map[string]int),
		jobEvals:      make(map[string]string),
		blocked:       make(map[string]PendingEvaluations),
		ready:         make(map[string]PendingEvaluations),
		unack:         make(map[string]*unackEval),
		waiting:       make(map[string]chan struct{}),
		requeue:       make(map[string]*structs.Evaluation),
		timeWait:      make(map[string]*time.Timer),
	}
	b.stats.ByScheduler = make(map[string]*SchedulerStats)
	return b, nil
}

// Enabled is used to check if the broker is enabled.
func (b *EvalBroker) Enabled() bool {
	b.l.RLock()
	defer b.l.RUnlock()
	return b.enabled
}

// SetEnabled is used to control if the broker is enabled. The broker
// should only be enabled on the active leader.
func (b *EvalBroker) SetEnabled(enabled bool) {
	b.l.Lock()
	b.enabled = enabled
	b.l.Unlock()
	if !enabled {
		b.Flush()
	}
}

// Enqueue is used to enqueue a new evaluation
func (b *EvalBroker) Enqueue(eval *structs.Evaluation) {
	b.l.Lock()
	defer b.l.Unlock()
	b.processEnqueue(eval, "")
}

// EnqueueAll is used to enqueue many evaluations. The map allows evaluations
// that are being re-enqueued to include their token.
//
// When requeueing an evaluation that potentially may be already
// enqueued. The evaluation is handled in one of the following ways:
// * Evaluation not outstanding: Process as a normal Enqueue
// * Evaluation outstanding: Do not allow the evaluation to be dequeued til:
//    * Ack received:  Unblock the evaluation allowing it to be dequeued
//    * Nack received: Drop the evaluation as it was created as a result of a
//    scheduler run that was Nack'd
func (b *EvalBroker) EnqueueAll(evals map[*structs.Evaluation]string) {
	// The lock needs to be held until all evaluations are enqueued. This is so
	// that when Dequeue operations are unblocked they will pick the highest
	// priority evaluations.
	b.l.Lock()
	defer b.l.Unlock()
	for eval, token := range evals {
		b.processEnqueue(eval, token)
	}
}

// processEnqueue deduplicates evals and either enqueue immediately or enforce
// the evals wait time. If the token is passed, and the evaluation ID is
// outstanding, the evaluation is blocked til an Ack/Nack is received.
// processEnqueue must be called with the lock held.
func (b *EvalBroker) processEnqueue(eval *structs.Evaluation, token string) {
	// Check if already enqueued
	if _, ok := b.evals[eval.ID]; ok {
		if token == "" {
			return
		}

		// If the token has been passed, the evaluation is being reblocked by
		// the scheduler and should be processed once the outstanding evaluation
		// is Acked or Nacked.
		if unack, ok := b.unack[eval.ID]; ok && unack.Token == token {
			b.requeue[token] = eval
		}
		return
	} else if b.enabled {
		b.evals[eval.ID] = 0
	}

	// Check if we need to enforce a wait
	if eval.Wait > 0 {
		timer := time.AfterFunc(eval.Wait, func() {
			b.enqueueWaiting(eval)
		})
		b.timeWait[eval.ID] = timer
		b.stats.TotalWaiting += 1
		return
	}

	b.enqueueLocked(eval, eval.Type)
}

// enqueueWaiting is used to enqueue a waiting evaluation
func (b *EvalBroker) enqueueWaiting(eval *structs.Evaluation) {
	b.l.Lock()
	defer b.l.Unlock()
	delete(b.timeWait, eval.ID)
	b.stats.TotalWaiting -= 1
	b.enqueueLocked(eval, eval.Type)
}

// enqueueLocked is used to enqueue with the lock held
func (b *EvalBroker) enqueueLocked(eval *structs.Evaluation, queue string) {
	// Do nothing if not enabled
	if !b.enabled {
		return
	}

	// Check if there is an evaluation for this JobID pending
	pendingEval := b.jobEvals[eval.JobID]
	if pendingEval == "" {
		b.jobEvals[eval.JobID] = eval.ID
	} else if pendingEval != eval.ID {
		blocked := b.blocked[eval.JobID]
		heap.Push(&blocked, eval)
		b.blocked[eval.JobID] = blocked
		b.stats.TotalBlocked += 1
		return
	}

	// Find the pending by scheduler class
	pending, ok := b.ready[queue]
	if !ok {
		pending = make([]*structs.Evaluation, 0, 16)
		if _, ok := b.waiting[queue]; !ok {
			b.waiting[queue] = make(chan struct{}, 1)
		}
	}

	// Push onto the heap
	heap.Push(&pending, eval)
	b.ready[queue] = pending

	// Update the stats
	b.stats.TotalReady += 1
	bySched, ok := b.stats.ByScheduler[queue]
	if !ok {
		bySched = &SchedulerStats{}
		b.stats.ByScheduler[queue] = bySched
	}
	bySched.Ready += 1

	// Unblock any blocked dequeues
	select {
	case b.waiting[queue] <- struct{}{}:
	default:
	}
}

// Dequeue is used to perform a blocking dequeue
func (b *EvalBroker) Dequeue(schedulers []string, timeout time.Duration) (*structs.Evaluation, string, error) {
	var timeoutTimer *time.Timer
	var timeoutCh <-chan time.Time
SCAN:
	// Scan for work
	eval, token, err := b.scanForSchedulers(schedulers)
	if err != nil {
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}
		return nil, "", err
	}

	// Check if we have something
	if eval != nil {
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}
		return eval, token, nil
	}

	// Setup the timeout channel the first time around
	if timeoutTimer == nil && timeout != 0 {
		timeoutTimer = time.NewTimer(timeout)
		timeoutCh = timeoutTimer.C
	}

	// Block until we get work
	scan := b.waitForSchedulers(schedulers, timeoutCh)
	if scan {
		goto SCAN
	}
	return nil, "", nil
}

// scanForSchedulers scans for work on any of the schedulers. The highest priority work
// is dequeued first. This may return nothing if there is no work waiting.
func (b *EvalBroker) scanForSchedulers(schedulers []string) (*structs.Evaluation, string, error) {
	b.l.Lock()
	defer b.l.Unlock()

	// Do nothing if not enabled
	if !b.enabled {
		return nil, "", fmt.Errorf("eval broker disabled")
	}

	// Scan for eligible work
	var eligibleSched []string
	var eligiblePriority int
	for _, sched := range schedulers {
		// Get the pending queue
		pending, ok := b.ready[sched]
		if !ok {
			continue
		}

		// Peek at the next item
		ready := pending.Peek()
		if ready == nil {
			continue
		}

		// Add to eligible if equal or greater priority
		if len(eligibleSched) == 0 || ready.Priority > eligiblePriority {
			eligibleSched = []string{sched}
			eligiblePriority = ready.Priority

		} else if eligiblePriority > ready.Priority {
			continue

		} else if eligiblePriority == ready.Priority {
			eligibleSched = append(eligibleSched, sched)
		}
	}

	// Determine behavior based on eligible work
	switch n := len(eligibleSched); n {
	case 0:
		// No work to do!
		return nil, "", nil

	case 1:
		// Only a single task, dequeue
		return b.dequeueForSched(eligibleSched[0])

	default:
		// Multiple tasks. We pick a random task so that we fairly
		// distribute work.
		offset := rand.Intn(n)
		return b.dequeueForSched(eligibleSched[offset])
	}
}

// dequeueForSched is used to dequeue the next work item for a given scheduler.
// This assumes locks are held and that this scheduler has work
func (b *EvalBroker) dequeueForSched(sched string) (*structs.Evaluation, string, error) {
	// Get the pending queue
	pending := b.ready[sched]
	raw := heap.Pop(&pending)
	b.ready[sched] = pending
	eval := raw.(*structs.Evaluation)

	// Generate a UUID for the token
	token := structs.GenerateUUID()

	// Setup Nack timer
	nackTimer := time.AfterFunc(b.nackTimeout, func() {
		b.Nack(eval.ID, token)
	})

	// Add to the unack queue
	b.unack[eval.ID] = &unackEval{
		Eval:      eval,
		Token:     token,
		NackTimer: nackTimer,
	}

	// Increment the dequeue count
	b.evals[eval.ID] += 1

	// Update the stats
	b.stats.TotalReady -= 1
	b.stats.TotalUnacked += 1
	bySched := b.stats.ByScheduler[sched]
	bySched.Ready -= 1
	bySched.Unacked += 1

	return eval, token, nil
}

// waitForSchedulers is used to wait for work on any of the scheduler or until a timeout.
// Returns if there is work waiting potentially.
func (b *EvalBroker) waitForSchedulers(schedulers []string, timeoutCh <-chan time.Time) bool {
	doneCh := make(chan struct{})
	readyCh := make(chan struct{}, 1)
	defer close(doneCh)

	// Start all the watchers
	b.l.Lock()
	for _, sched := range schedulers {
		waitCh, ok := b.waiting[sched]
		if !ok {
			waitCh = make(chan struct{}, 1)
			b.waiting[sched] = waitCh
		}

		// Start a goroutine that either waits for the waitCh on this scheduler
		// to unblock or for this waitForSchedulers call to return
		go func() {
			select {
			case <-waitCh:
				select {
				case readyCh <- struct{}{}:
				default:
				}
			case <-doneCh:
			}
		}()
	}
	b.l.Unlock()

	// Block until we have ready work and should scan, or until we timeout
	// and should not make an attempt to scan for work
	select {
	case <-readyCh:
		return true
	case <-timeoutCh:
		return false
	}
}

// Outstanding checks if an EvalID has been delivered but not acknowledged
// and returns the associated token for the evaluation.
func (b *EvalBroker) Outstanding(evalID string) (string, bool) {
	b.l.RLock()
	defer b.l.RUnlock()
	unack, ok := b.unack[evalID]
	if !ok {
		return "", false
	}
	return unack.Token, true
}

// OutstandingReset resets the Nack timer for the EvalID if the
// token matches and the eval is outstanding
func (b *EvalBroker) OutstandingReset(evalID, token string) error {
	b.l.RLock()
	defer b.l.RUnlock()
	unack, ok := b.unack[evalID]
	if !ok {
		return ErrNotOutstanding
	}
	if unack.Token != token {
		return ErrTokenMismatch
	}
	if !unack.NackTimer.Reset(b.nackTimeout) {
		return ErrNackTimeoutReached
	}
	return nil
}

// Ack is used to positively acknowledge handling an evaluation
func (b *EvalBroker) Ack(evalID, token string) error {
	b.l.Lock()
	defer b.l.Unlock()

	// Always delete the requeued evaluation. Either the Ack is successful and
	// we requeue it or it isn't and we want to remove it.
	defer delete(b.requeue, token)

	// Lookup the unack'd eval
	unack, ok := b.unack[evalID]
	if !ok {
		return fmt.Errorf("Evaluation ID not found")
	}
	if unack.Token != token {
		return fmt.Errorf("Token does not match for Evaluation ID")
	}
	jobID := unack.Eval.JobID

	// Ensure we were able to stop the timer
	if !unack.NackTimer.Stop() {
		return fmt.Errorf("Evaluation ID Ack'd after Nack timer expiration")
	}

	// Update the stats
	b.stats.TotalUnacked -= 1
	queue := unack.Eval.Type
	if b.evals[evalID] > b.deliveryLimit {
		queue = failedQueue
	}
	bySched := b.stats.ByScheduler[queue]
	bySched.Unacked -= 1

	// Cleanup
	delete(b.unack, evalID)
	delete(b.evals, evalID)
	delete(b.jobEvals, jobID)

	// Check if there are any blocked evaluations
	if blocked := b.blocked[jobID]; len(blocked) != 0 {
		raw := heap.Pop(&blocked)
		if len(blocked) > 0 {
			b.blocked[jobID] = blocked
		} else {
			delete(b.blocked, jobID)
		}
		eval := raw.(*structs.Evaluation)
		b.stats.TotalBlocked -= 1
		b.enqueueLocked(eval, eval.Type)
	}

	// Re-enqueue the evaluation.
	if eval, ok := b.requeue[token]; ok {
		b.processEnqueue(eval, "")
	}

	return nil
}

// Nack is used to negatively acknowledge handling an evaluation
func (b *EvalBroker) Nack(evalID, token string) error {
	b.l.Lock()
	defer b.l.Unlock()

	// Always delete the requeued evaluation since the Nack means the requeue is
	// invalid.
	delete(b.requeue, token)

	// Lookup the unack'd eval
	unack, ok := b.unack[evalID]
	if !ok {
		return fmt.Errorf("Evaluation ID not found")
	}
	if unack.Token != token {
		return fmt.Errorf("Token does not match for Evaluation ID")
	}

	// Stop the timer, doesn't matter if we've missed it
	unack.NackTimer.Stop()

	// Cleanup
	delete(b.unack, evalID)

	// Update the stats
	b.stats.TotalUnacked -= 1
	bySched := b.stats.ByScheduler[unack.Eval.Type]
	bySched.Unacked -= 1

	// Check if we've hit the delivery limit, and re-enqueue
	// in the failedQueue
	if b.evals[evalID] >= b.deliveryLimit {
		b.enqueueLocked(unack.Eval, failedQueue)
	} else {
		b.enqueueLocked(unack.Eval, unack.Eval.Type)
	}
	return nil
}

// PauseNackTimeout is used to pause the Nack timeout for an eval that is making
// progress but is in a potentially unbounded operation such as the plan queue.
func (b *EvalBroker) PauseNackTimeout(evalID, token string) error {
	b.l.RLock()
	defer b.l.RUnlock()
	unack, ok := b.unack[evalID]
	if !ok {
		return ErrNotOutstanding
	}
	if unack.Token != token {
		return ErrTokenMismatch
	}
	if !unack.NackTimer.Stop() {
		return ErrNackTimeoutReached
	}
	return nil
}

// ResumeNackTimeout is used to resume the Nack timeout for an eval that was
// paused. It should be resumed after leaving an unbounded operation.
func (b *EvalBroker) ResumeNackTimeout(evalID, token string) error {
	b.l.Lock()
	defer b.l.Unlock()
	unack, ok := b.unack[evalID]
	if !ok {
		return ErrNotOutstanding
	}
	if unack.Token != token {
		return ErrTokenMismatch
	}
	unack.NackTimer.Reset(b.nackTimeout)
	return nil
}

// Flush is used to clear the state of the broker
func (b *EvalBroker) Flush() {
	b.l.Lock()
	defer b.l.Unlock()

	// Unblock any waiters
	for _, waitCh := range b.waiting {
		close(waitCh)
	}
	b.waiting = make(map[string]chan struct{})

	// Cancel any Nack timers
	for _, unack := range b.unack {
		unack.NackTimer.Stop()
	}

	// Cancel any time wait evals
	for _, wait := range b.timeWait {
		wait.Stop()
	}

	// Reset the broker
	b.stats.TotalReady = 0
	b.stats.TotalUnacked = 0
	b.stats.TotalBlocked = 0
	b.stats.TotalWaiting = 0
	b.stats.ByScheduler = make(map[string]*SchedulerStats)
	b.evals = make(map[string]int)
	b.jobEvals = make(map[string]string)
	b.blocked = make(map[string]PendingEvaluations)
	b.ready = make(map[string]PendingEvaluations)
	b.unack = make(map[string]*unackEval)
	b.timeWait = make(map[string]*time.Timer)
}

// Stats is used to query the state of the broker
func (b *EvalBroker) Stats() *BrokerStats {
	// Allocate a new stats struct
	stats := new(BrokerStats)
	stats.ByScheduler = make(map[string]*SchedulerStats)

	b.l.RLock()
	defer b.l.RUnlock()

	// Copy all the stats
	stats.TotalReady = b.stats.TotalReady
	stats.TotalUnacked = b.stats.TotalUnacked
	stats.TotalBlocked = b.stats.TotalBlocked
	stats.TotalWaiting = b.stats.TotalWaiting
	for sched, subStat := range b.stats.ByScheduler {
		subStatCopy := new(SchedulerStats)
		*subStatCopy = *subStat
		stats.ByScheduler[sched] = subStatCopy
	}
	return stats
}

// EmitStats is used to export metrics about the broker while enabled
func (b *EvalBroker) EmitStats(period time.Duration, stopCh chan struct{}) {
	for {
		select {
		case <-time.After(period):
			stats := b.Stats()
			metrics.SetGauge([]string{"nomad", "broker", "total_ready"}, float32(stats.TotalReady))
			metrics.SetGauge([]string{"nomad", "broker", "total_unacked"}, float32(stats.TotalUnacked))
			metrics.SetGauge([]string{"nomad", "broker", "total_blocked"}, float32(stats.TotalBlocked))
			metrics.SetGauge([]string{"nomad", "broker", "total_waiting"}, float32(stats.TotalWaiting))
			for sched, schedStats := range stats.ByScheduler {
				metrics.SetGauge([]string{"nomad", "broker", sched, "ready"}, float32(schedStats.Ready))
				metrics.SetGauge([]string{"nomad", "broker", sched, "unacked"}, float32(schedStats.Unacked))
			}

		case <-stopCh:
			return
		}
	}
}

// BrokerStats returns all the stats about the broker
type BrokerStats struct {
	TotalReady   int
	TotalUnacked int
	TotalBlocked int
	TotalWaiting int
	ByScheduler  map[string]*SchedulerStats
}

// SchedulerStats returns the stats per scheduler
type SchedulerStats struct {
	Ready   int
	Unacked int
}

// Len is for the sorting interface
func (p PendingEvaluations) Len() int {
	return len(p)
}

// Less is for the sorting interface. We flip the check
// so that the "min" in the min-heap is the element with the
// highest priority
func (p PendingEvaluations) Less(i, j int) bool {
	if p[i].JobID != p[j].JobID && p[i].Priority != p[j].Priority {
		return !(p[i].Priority < p[j].Priority)
	}
	return p[i].CreateIndex < p[j].CreateIndex
}

// Swap is for the sorting interface
func (p PendingEvaluations) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// Push is used to add a new evalution to the slice
func (p *PendingEvaluations) Push(e interface{}) {
	*p = append(*p, e.(*structs.Evaluation))
}

// Pop is used to remove an evaluation from the slice
func (p *PendingEvaluations) Pop() interface{} {
	n := len(*p)
	e := (*p)[n-1]
	(*p)[n-1] = nil
	*p = (*p)[:n-1]
	return e
}

// Peek is used to peek at the next element that would be popped
func (p PendingEvaluations) Peek() *structs.Evaluation {
	n := len(p)
	if n == 0 {
		return nil
	}
	return p[n-1]
}
