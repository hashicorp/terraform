package nomad

import (
	"sync"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/consul/lib"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// unblockBuffer is the buffer size for the unblock channel. The buffer
	// should be large to ensure that the FSM doesn't block when calling Unblock
	// as this would apply back-pressure on Raft.
	unblockBuffer = 8096
)

// BlockedEvals is used to track evaluations that shouldn't be queued until a
// certain class of nodes becomes available. An evaluation is put into the
// blocked state when it is run through the scheduler and produced failed
// allocations. It is unblocked when the capacity of a node that could run the
// failed allocation becomes available.
type BlockedEvals struct {
	evalBroker *EvalBroker
	enabled    bool
	stats      *BlockedStats
	l          sync.RWMutex

	// captured is the set of evaluations that are captured by computed node
	// classes.
	captured map[string]wrappedEval

	// escaped is the set of evaluations that have escaped computed node
	// classes.
	escaped map[string]wrappedEval

	// unblockCh is used to buffer unblocking of evaluations.
	capacityChangeCh chan *capacityUpdate

	// jobs is the map of blocked job and is used to ensure that only one
	// blocked eval exists for each job.
	jobs map[string]struct{}

	// unblockIndexes maps computed node classes to the index in which they were
	// unblocked. This is used to check if an evaluation could have been
	// unblocked between the time they were in the scheduler and the time they
	// are being blocked.
	unblockIndexes map[string]uint64

	// duplicates is the set of evaluations for jobs that had pre-existing
	// blocked evaluations. These should be marked as cancelled since only one
	// blocked eval is neeeded per job.
	duplicates []*structs.Evaluation

	// duplicateCh is used to signal that a duplicate eval was added to the
	// duplicate set. It can be used to unblock waiting callers looking for
	// duplicates.
	duplicateCh chan struct{}

	// stopCh is used to stop any created goroutines.
	stopCh chan struct{}
}

// capacityUpdate stores unblock data.
type capacityUpdate struct {
	computedClass string
	index         uint64
}

// wrappedEval captures both the evaluation and the optional token
type wrappedEval struct {
	eval  *structs.Evaluation
	token string
}

// BlockedStats returns all the stats about the blocked eval tracker.
type BlockedStats struct {
	// TotalEscaped is the total number of blocked evaluations that have escaped
	// computed node classes.
	TotalEscaped int

	// TotalBlocked is the total number of blocked evaluations.
	TotalBlocked int
}

// NewBlockedEvals creates a new blocked eval tracker that will enqueue
// unblocked evals into the passed broker.
func NewBlockedEvals(evalBroker *EvalBroker) *BlockedEvals {
	return &BlockedEvals{
		evalBroker:       evalBroker,
		captured:         make(map[string]wrappedEval),
		escaped:          make(map[string]wrappedEval),
		jobs:             make(map[string]struct{}),
		unblockIndexes:   make(map[string]uint64),
		capacityChangeCh: make(chan *capacityUpdate, unblockBuffer),
		duplicateCh:      make(chan struct{}, 1),
		stopCh:           make(chan struct{}),
		stats:            new(BlockedStats),
	}
}

// Enabled is used to check if the broker is enabled.
func (b *BlockedEvals) Enabled() bool {
	b.l.RLock()
	defer b.l.RUnlock()
	return b.enabled
}

// SetEnabled is used to control if the blocked eval tracker is enabled. The
// tracker should only be enabled on the active leader.
func (b *BlockedEvals) SetEnabled(enabled bool) {
	b.l.Lock()
	if b.enabled == enabled {
		// No-op
		b.l.Unlock()
		return
	} else if enabled {
		go b.watchCapacity()
	} else {
		close(b.stopCh)
	}
	b.enabled = enabled
	b.l.Unlock()
	if !enabled {
		b.Flush()
	}
}

// Block tracks the passed evaluation and enqueues it into the eval broker when
// a suitable node calls unblock.
func (b *BlockedEvals) Block(eval *structs.Evaluation) {
	b.processBlock(eval, "")
}

// Reblock tracks the passed evaluation and enqueues it into the eval broker when
// a suitable node calls unblock. Reblock should be used over Block when the
// blocking is occurring by an outstanding evaluation. The token is the
// evaluation's token.
func (b *BlockedEvals) Reblock(eval *structs.Evaluation, token string) {
	b.processBlock(eval, token)
}

// processBlock is the implementation of blocking an evaluation. It supports
// taking an optional evaluation token to use when reblocking an evaluation that
// may be outstanding.
func (b *BlockedEvals) processBlock(eval *structs.Evaluation, token string) {
	b.l.Lock()
	defer b.l.Unlock()

	// Do nothing if not enabled
	if !b.enabled {
		return
	}

	// Check if the job already has a blocked evaluation. If it does add it to
	// the list of duplicates. We omly ever want one blocked evaluation per job,
	// otherwise we would create unnecessary work for the scheduler as multiple
	// evals for the same job would be run, all producing the same outcome.
	if _, existing := b.jobs[eval.JobID]; existing {
		b.duplicates = append(b.duplicates, eval)

		// Unblock any waiter.
		select {
		case b.duplicateCh <- struct{}{}:
		default:
		}

		return
	}

	// Check if the eval missed an unblock while it was in the scheduler at an
	// older index. The scheduler could have been invoked with a snapshot of
	// state that was prior to additional capacity being added or allocations
	// becoming terminal.
	if b.missedUnblock(eval) {
		// Just re-enqueue the eval immediately. We pass the token so that the
		// eval_broker can properly handle the case in which the evaluation is
		// still outstanding.
		b.evalBroker.EnqueueAll(map[*structs.Evaluation]string{eval: token})
		return
	}

	// Mark the job as tracked.
	b.stats.TotalBlocked++
	b.jobs[eval.JobID] = struct{}{}

	// Wrap the evaluation, capturing its token.
	wrapped := wrappedEval{
		eval:  eval,
		token: token,
	}

	// If the eval has escaped, meaning computed node classes could not capture
	// the constraints of the job, we store the eval separately as we have to
	// unblock it whenever node capacity changes. This is because we don't know
	// what node class is feasible for the jobs constraints.
	if eval.EscapedComputedClass {
		b.escaped[eval.ID] = wrapped
		b.stats.TotalEscaped++
		return
	}

	// Add the eval to the set of blocked evals whose jobs constraints are
	// captured by computed node class.
	b.captured[eval.ID] = wrapped
}

// missedUnblock returns whether an evaluation missed an unblock while it was in
// the scheduler. Since the scheduler can operate at an index in the past, the
// evaluation may have been processed missing data that would allow it to
// complete. This method returns if that is the case and should be called with
// the lock held.
func (b *BlockedEvals) missedUnblock(eval *structs.Evaluation) bool {
	var max uint64 = 0
	for class, index := range b.unblockIndexes {
		// Calculate the max unblock index
		if max < index {
			max = index
		}

		elig, ok := eval.ClassEligibility[class]
		if !ok && eval.SnapshotIndex < index {
			// The evaluation was processed and did not encounter this class
			// because it was added after it was processed. Thus for correctness
			// we need to unblock it.
			return true
		}

		// The evaluation could use the computed node class and the eval was
		// processed before the last unblock.
		if elig && eval.SnapshotIndex < index {
			return true
		}
	}

	// If the evaluation has escaped, and the map contains an index older than
	// the evaluations, it should be unblocked.
	if eval.EscapedComputedClass && eval.SnapshotIndex < max {
		return true
	}

	// The evaluation is ahead of all recent unblocks.
	return false
}

// Unblock causes any evaluation that could potentially make progress on a
// capacity change on the passed computed node class to be enqueued into the
// eval broker.
func (b *BlockedEvals) Unblock(computedClass string, index uint64) {
	b.l.Lock()

	// Do nothing if not enabled
	if !b.enabled {
		b.l.Unlock()
		return
	}

	// Store the index in which the unblock happened. We use this on subsequent
	// block calls in case the evaluation was in the scheduler when a trigger
	// occurred.
	b.unblockIndexes[computedClass] = index
	b.l.Unlock()

	b.capacityChangeCh <- &capacityUpdate{
		computedClass: computedClass,
		index:         index,
	}
}

// watchCapacity is a long lived function that watches for capacity changes in
// nodes and unblocks the correct set of evals.
func (b *BlockedEvals) watchCapacity() {
	for {
		select {
		case <-b.stopCh:
			return
		case update := <-b.capacityChangeCh:
			b.unblock(update.computedClass, update.index)
		}
	}
}

// unblock unblocks all blocked evals that could run on the passed computed node
// class.
func (b *BlockedEvals) unblock(computedClass string, index uint64) {
	b.l.Lock()
	defer b.l.Unlock()

	// Protect against the case of a flush.
	if !b.enabled {
		return
	}

	// Every eval that has escaped computed node class has to be unblocked
	// because any node could potentially be feasible.
	numEscaped := len(b.escaped)
	unblocked := make(map[*structs.Evaluation]string, lib.MaxInt(numEscaped, 4))
	if numEscaped != 0 {
		for id, wrapped := range b.escaped {
			unblocked[wrapped.eval] = wrapped.token
			delete(b.escaped, id)
			delete(b.jobs, wrapped.eval.JobID)
		}
	}

	// We unblock any eval that is explicitly eligible for the computed class
	// and also any eval that is not eligible or uneligible. This signifies that
	// when the evaluation was originally run through the scheduler, that it
	// never saw a node with the given computed class and thus needs to be
	// unblocked for correctness.
	for id, wrapped := range b.captured {
		if elig, ok := wrapped.eval.ClassEligibility[computedClass]; ok && !elig {
			// Can skip because the eval has explicitly marked the node class
			// as ineligible.
			continue
		}

		// The computed node class has never been seen by the eval so we unblock
		// it.
		unblocked[wrapped.eval] = wrapped.token
		delete(b.jobs, wrapped.eval.JobID)
		delete(b.captured, id)
	}

	if l := len(unblocked); l != 0 {
		// Update the counters
		b.stats.TotalEscaped = 0
		b.stats.TotalBlocked -= l

		// Enqueue all the unblocked evals into the broker.
		b.evalBroker.EnqueueAll(unblocked)
	}
}

// UnblockFailed unblocks all blocked evaluation that were due to scheduler
// failure.
func (b *BlockedEvals) UnblockFailed() {
	b.l.Lock()
	defer b.l.Unlock()

	// Do nothing if not enabled
	if !b.enabled {
		return
	}

	unblocked := make(map[*structs.Evaluation]string, 4)
	for id, wrapped := range b.captured {
		if wrapped.eval.TriggeredBy == structs.EvalTriggerMaxPlans {
			unblocked[wrapped.eval] = wrapped.token
			delete(b.captured, id)
			delete(b.jobs, wrapped.eval.JobID)
		}
	}

	for id, wrapped := range b.escaped {
		if wrapped.eval.TriggeredBy == structs.EvalTriggerMaxPlans {
			unblocked[wrapped.eval] = wrapped.token
			delete(b.escaped, id)
			delete(b.jobs, wrapped.eval.JobID)
			b.stats.TotalEscaped -= 1
		}
	}

	if l := len(unblocked); l > 0 {
		b.stats.TotalBlocked -= l
		b.evalBroker.EnqueueAll(unblocked)
	}
}

// GetDuplicates returns all the duplicate evaluations and blocks until the
// passed timeout.
func (b *BlockedEvals) GetDuplicates(timeout time.Duration) []*structs.Evaluation {
	var timeoutTimer *time.Timer
	var timeoutCh <-chan time.Time
SCAN:
	b.l.Lock()
	if len(b.duplicates) != 0 {
		dups := b.duplicates
		b.duplicates = nil
		b.l.Unlock()
		return dups
	}
	b.l.Unlock()

	// Create the timer
	if timeoutTimer == nil && timeout != 0 {
		timeoutTimer = time.NewTimer(timeout)
		timeoutCh = timeoutTimer.C
		defer timeoutTimer.Stop()
	}

	select {
	case <-b.stopCh:
		return nil
	case <-timeoutCh:
		return nil
	case <-b.duplicateCh:
		goto SCAN
	}
}

// Flush is used to clear the state of blocked evaluations.
func (b *BlockedEvals) Flush() {
	b.l.Lock()
	defer b.l.Unlock()

	// Reset the blocked eval tracker.
	b.stats.TotalEscaped = 0
	b.stats.TotalBlocked = 0
	b.captured = make(map[string]wrappedEval)
	b.escaped = make(map[string]wrappedEval)
	b.jobs = make(map[string]struct{})
	b.duplicates = nil
	b.capacityChangeCh = make(chan *capacityUpdate, unblockBuffer)
	b.stopCh = make(chan struct{})
	b.duplicateCh = make(chan struct{}, 1)
}

// Stats is used to query the state of the blocked eval tracker.
func (b *BlockedEvals) Stats() *BlockedStats {
	// Allocate a new stats struct
	stats := new(BlockedStats)

	b.l.RLock()
	defer b.l.RUnlock()

	// Copy all the stats
	stats.TotalEscaped = b.stats.TotalEscaped
	stats.TotalBlocked = b.stats.TotalBlocked
	return stats
}

// EmitStats is used to export metrics about the blocked eval tracker while enabled
func (b *BlockedEvals) EmitStats(period time.Duration, stopCh chan struct{}) {
	for {
		select {
		case <-time.After(period):
			stats := b.Stats()
			metrics.SetGauge([]string{"nomad", "blocked_evals", "total_blocked"}, float32(stats.TotalBlocked))
			metrics.SetGauge([]string{"nomad", "blocked_evals", "total_escaped"}, float32(stats.TotalEscaped))
		case <-stopCh:
			return
		}
	}
}
