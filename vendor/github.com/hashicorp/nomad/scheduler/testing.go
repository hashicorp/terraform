package scheduler

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
)

// RejectPlan is used to always reject the entire plan and force a state refresh
type RejectPlan struct {
	Harness *Harness
}

func (r *RejectPlan) SubmitPlan(*structs.Plan) (*structs.PlanResult, State, error) {
	result := new(structs.PlanResult)
	result.RefreshIndex = r.Harness.NextIndex()
	return result, r.Harness.State, nil
}

func (r *RejectPlan) UpdateEval(eval *structs.Evaluation) error {
	return nil
}

func (r *RejectPlan) CreateEval(*structs.Evaluation) error {
	return nil
}

func (r *RejectPlan) ReblockEval(*structs.Evaluation) error {
	return nil
}

// Harness is a lightweight testing harness for schedulers. It manages a state
// store copy and provides the planner interface. It can be extended for various
// testing uses or for invoking the scheduler without side effects.
type Harness struct {
	State *state.StateStore

	Planner  Planner
	planLock sync.Mutex

	Plans        []*structs.Plan
	Evals        []*structs.Evaluation
	CreateEvals  []*structs.Evaluation
	ReblockEvals []*structs.Evaluation

	nextIndex     uint64
	nextIndexLock sync.Mutex
}

// NewHarness is used to make a new testing harness
func NewHarness(t *testing.T) *Harness {
	state, err := state.NewStateStore(os.Stderr)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	h := &Harness{
		State:     state,
		nextIndex: 1,
	}
	return h
}

// SubmitPlan is used to handle plan submission
func (h *Harness) SubmitPlan(plan *structs.Plan) (*structs.PlanResult, State, error) {
	// Ensure sequential plan application
	h.planLock.Lock()
	defer h.planLock.Unlock()

	// Store the plan
	h.Plans = append(h.Plans, plan)

	// Check for custom planner
	if h.Planner != nil {
		return h.Planner.SubmitPlan(plan)
	}

	// Get the index
	index := h.NextIndex()

	// Prepare the result
	result := new(structs.PlanResult)
	result.NodeUpdate = plan.NodeUpdate
	result.NodeAllocation = plan.NodeAllocation
	result.AllocIndex = index

	// Flatten evicts and allocs
	var allocs []*structs.Allocation
	for _, updateList := range plan.NodeUpdate {
		allocs = append(allocs, updateList...)
	}
	for _, allocList := range plan.NodeAllocation {
		allocs = append(allocs, allocList...)
	}

	// Attach the plan to all the allocations. It is pulled out in the
	// payload to avoid the redundancy of encoding, but should be denormalized
	// prior to being inserted into MemDB.
	if j := plan.Job; j != nil {
		for _, alloc := range allocs {
			if alloc.Job == nil {
				alloc.Job = j
			}
		}
	}

	// Apply the full plan
	err := h.State.UpsertAllocs(index, allocs)
	return result, nil, err
}

func (h *Harness) UpdateEval(eval *structs.Evaluation) error {
	// Ensure sequential plan application
	h.planLock.Lock()
	defer h.planLock.Unlock()

	// Store the eval
	h.Evals = append(h.Evals, eval)

	// Check for custom planner
	if h.Planner != nil {
		return h.Planner.UpdateEval(eval)
	}
	return nil
}

func (h *Harness) CreateEval(eval *structs.Evaluation) error {
	// Ensure sequential plan application
	h.planLock.Lock()
	defer h.planLock.Unlock()

	// Store the eval
	h.CreateEvals = append(h.CreateEvals, eval)

	// Check for custom planner
	if h.Planner != nil {
		return h.Planner.CreateEval(eval)
	}
	return nil
}

func (h *Harness) ReblockEval(eval *structs.Evaluation) error {
	// Ensure sequential plan application
	h.planLock.Lock()
	defer h.planLock.Unlock()

	// Check that the evaluation was already blocked.
	old, err := h.State.EvalByID(eval.ID)
	if err != nil {
		return err
	}

	if old == nil {
		return fmt.Errorf("evaluation does not exist to be reblocked")
	}
	if old.Status != structs.EvalStatusBlocked {
		return fmt.Errorf("evaluation %q is not already in a blocked state", old.ID)
	}

	h.ReblockEvals = append(h.ReblockEvals, eval)
	return nil
}

// NextIndex returns the next index
func (h *Harness) NextIndex() uint64 {
	h.nextIndexLock.Lock()
	defer h.nextIndexLock.Unlock()
	idx := h.nextIndex
	h.nextIndex += 1
	return idx
}

// Snapshot is used to snapshot the current state
func (h *Harness) Snapshot() State {
	snap, _ := h.State.Snapshot()
	return snap
}

// Scheduler is used to return a new scheduler from
// a snapshot of current state using the harness for planning.
func (h *Harness) Scheduler(factory Factory) Scheduler {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return factory(logger, h.Snapshot(), h)
}

// Process is used to process an evaluation given a factory
// function to create the scheduler
func (h *Harness) Process(factory Factory, eval *structs.Evaluation) error {
	sched := h.Scheduler(factory)
	return sched.Process(eval)
}

func (h *Harness) AssertEvalStatus(t *testing.T, state string) {
	if len(h.Evals) != 1 {
		t.Fatalf("bad: %#v", h.Evals)
	}
	update := h.Evals[0]

	if update.Status != state {
		t.Fatalf("bad: %#v", update)
	}
}
