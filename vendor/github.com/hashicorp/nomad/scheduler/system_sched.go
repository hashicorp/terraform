package scheduler

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// maxSystemScheduleAttempts is used to limit the number of times
	// we will attempt to schedule if we continue to hit conflicts for system
	// jobs.
	maxSystemScheduleAttempts = 5

	// allocNodeTainted is the status used when stopping an alloc because it's
	// node is tainted.
	allocNodeTainted = "system alloc not needed as node is tainted"
)

// SystemScheduler is used for 'system' jobs. This scheduler is
// designed for services that should be run on every client.
type SystemScheduler struct {
	logger  *log.Logger
	state   State
	planner Planner

	eval       *structs.Evaluation
	job        *structs.Job
	plan       *structs.Plan
	planResult *structs.PlanResult
	ctx        *EvalContext
	stack      *SystemStack
	nodes      []*structs.Node
	nodesByDC  map[string]int

	limitReached bool
	nextEval     *structs.Evaluation

	failedTGAllocs map[string]*structs.AllocMetric
	queuedAllocs   map[string]int
}

// NewSystemScheduler is a factory function to instantiate a new system
// scheduler.
func NewSystemScheduler(logger *log.Logger, state State, planner Planner) Scheduler {
	return &SystemScheduler{
		logger:  logger,
		state:   state,
		planner: planner,
	}
}

// Process is used to handle a single evaluation.
func (s *SystemScheduler) Process(eval *structs.Evaluation) error {
	// Store the evaluation
	s.eval = eval

	// Verify the evaluation trigger reason is understood
	switch eval.TriggeredBy {
	case structs.EvalTriggerJobRegister, structs.EvalTriggerNodeUpdate,
		structs.EvalTriggerJobDeregister, structs.EvalTriggerRollingUpdate:
	default:
		desc := fmt.Sprintf("scheduler cannot handle '%s' evaluation reason",
			eval.TriggeredBy)
		return setStatus(s.logger, s.planner, s.eval, s.nextEval, nil, s.failedTGAllocs, structs.EvalStatusFailed, desc,
			s.queuedAllocs)
	}

	// Retry up to the maxSystemScheduleAttempts and reset if progress is made.
	progress := func() bool { return progressMade(s.planResult) }
	if err := retryMax(maxSystemScheduleAttempts, s.process, progress); err != nil {
		if statusErr, ok := err.(*SetStatusError); ok {
			return setStatus(s.logger, s.planner, s.eval, s.nextEval, nil, s.failedTGAllocs, statusErr.EvalStatus, err.Error(),
				s.queuedAllocs)
		}
		return err
	}

	// Update the status to complete
	return setStatus(s.logger, s.planner, s.eval, s.nextEval, nil, s.failedTGAllocs, structs.EvalStatusComplete, "",
		s.queuedAllocs)
}

// process is wrapped in retryMax to iteratively run the handler until we have no
// further work or we've made the maximum number of attempts.
func (s *SystemScheduler) process() (bool, error) {
	// Lookup the Job by ID
	var err error
	s.job, err = s.state.JobByID(s.eval.JobID)
	if err != nil {
		return false, fmt.Errorf("failed to get job '%s': %v",
			s.eval.JobID, err)
	}
	numTaskGroups := 0
	if s.job != nil {
		numTaskGroups = len(s.job.TaskGroups)
	}
	s.queuedAllocs = make(map[string]int, numTaskGroups)

	// Get the ready nodes in the required datacenters
	if s.job != nil {
		s.nodes, s.nodesByDC, err = readyNodesInDCs(s.state, s.job.Datacenters)
		if err != nil {
			return false, fmt.Errorf("failed to get ready nodes: %v", err)
		}
	}

	// Create a plan
	s.plan = s.eval.MakePlan(s.job)

	// Reset the failed allocations
	s.failedTGAllocs = nil

	// Create an evaluation context
	s.ctx = NewEvalContext(s.state, s.plan, s.logger)

	// Construct the placement stack
	s.stack = NewSystemStack(s.ctx)
	if s.job != nil {
		s.stack.SetJob(s.job)
	}

	// Compute the target job allocations
	if err := s.computeJobAllocs(); err != nil {
		s.logger.Printf("[ERR] sched: %#v: %v", s.eval, err)
		return false, err
	}

	// If the plan is a no-op, we can bail. If AnnotatePlan is set submit the plan
	// anyways to get the annotations.
	if s.plan.IsNoOp() && !s.eval.AnnotatePlan {
		return true, nil
	}

	// If the limit of placements was reached we need to create an evaluation
	// to pickup from here after the stagger period.
	if s.limitReached && s.nextEval == nil {
		s.nextEval = s.eval.NextRollingEval(s.job.Update.Stagger)
		if err := s.planner.CreateEval(s.nextEval); err != nil {
			s.logger.Printf("[ERR] sched: %#v failed to make next eval for rolling update: %v", s.eval, err)
			return false, err
		}
		s.logger.Printf("[DEBUG] sched: %#v: rolling update limit reached, next eval '%s' created", s.eval, s.nextEval.ID)
	}

	// Submit the plan
	result, newState, err := s.planner.SubmitPlan(s.plan)
	s.planResult = result
	if err != nil {
		return false, err
	}

	// Decrement the number of allocations pending per task group based on the
	// number of allocations successfully placed
	adjustQueuedAllocations(s.logger, result, s.queuedAllocs)

	// If we got a state refresh, try again since we have stale data
	if newState != nil {
		s.logger.Printf("[DEBUG] sched: %#v: refresh forced", s.eval)
		s.state = newState
		return false, nil
	}

	// Try again if the plan was not fully committed, potential conflict
	fullCommit, expected, actual := result.FullCommit(s.plan)
	if !fullCommit {
		s.logger.Printf("[DEBUG] sched: %#v: attempted %d placements, %d placed",
			s.eval, expected, actual)
		return false, nil
	}

	// Success!
	return true, nil
}

// computeJobAllocs is used to reconcile differences between the job,
// existing allocations and node status to update the allocations.
func (s *SystemScheduler) computeJobAllocs() error {
	// Lookup the allocations by JobID
	allocs, err := s.state.AllocsByJob(s.eval.JobID)
	if err != nil {
		return fmt.Errorf("failed to get allocs for job '%s': %v",
			s.eval.JobID, err)
	}

	// Determine the tainted nodes containing job allocs
	tainted, err := taintedNodes(s.state, allocs)
	if err != nil {
		return fmt.Errorf("failed to get tainted nodes for job '%s': %v",
			s.eval.JobID, err)
	}

	// Update the allocations which are in pending/running state on tainted
	// nodes to lost
	updateNonTerminalAllocsToLost(s.plan, tainted, allocs)

	// Filter out the allocations in a terminal state
	allocs = structs.FilterTerminalAllocs(allocs)

	// Diff the required and existing allocations
	diff := diffSystemAllocs(s.job, s.nodes, tainted, allocs)
	s.logger.Printf("[DEBUG] sched: %#v: %#v", s.eval, diff)

	// Add all the allocs to stop
	for _, e := range diff.stop {
		s.plan.AppendUpdate(e.Alloc, structs.AllocDesiredStatusStop, allocNotNeeded, "")
	}

	// Lost allocations should be transistioned to desired status stop and client
	// status lost.
	for _, e := range diff.lost {
		s.plan.AppendUpdate(e.Alloc, structs.AllocDesiredStatusStop, allocLost, structs.AllocClientStatusLost)
	}

	// Attempt to do the upgrades in place
	destructiveUpdates, inplaceUpdates := inplaceUpdate(s.ctx, s.eval, s.job, s.stack, diff.update)
	diff.update = destructiveUpdates

	if s.eval.AnnotatePlan {
		s.plan.Annotations = &structs.PlanAnnotations{
			DesiredTGUpdates: desiredUpdates(diff, inplaceUpdates, destructiveUpdates),
		}
	}

	// Check if a rolling upgrade strategy is being used
	limit := len(diff.update)
	if s.job != nil && s.job.Update.Rolling() {
		limit = s.job.Update.MaxParallel
	}

	// Treat non in-place updates as an eviction and new placement.
	s.limitReached = evictAndPlace(s.ctx, diff, diff.update, allocUpdating, &limit)

	// Nothing remaining to do if placement is not required
	if len(diff.place) == 0 {
		if s.job != nil {
			for _, tg := range s.job.TaskGroups {
				s.queuedAllocs[tg.Name] = 0
			}
		}
		return nil
	}

	// Record the number of allocations that needs to be placed per Task Group
	for _, allocTuple := range diff.place {
		s.queuedAllocs[allocTuple.TaskGroup.Name] += 1
	}

	// Compute the placements
	return s.computePlacements(diff.place)
}

// computePlacements computes placements for allocations
func (s *SystemScheduler) computePlacements(place []allocTuple) error {
	nodeByID := make(map[string]*structs.Node, len(s.nodes))
	for _, node := range s.nodes {
		nodeByID[node.ID] = node
	}

	nodes := make([]*structs.Node, 1)

	// nodesFiltered holds the number of nodes filtered by the stack due to
	// constrain mismatches while we are trying to place allocations on node
	var nodesFiltered int
	for _, missing := range place {
		node, ok := nodeByID[missing.Alloc.NodeID]
		if !ok {
			return fmt.Errorf("could not find node %q", missing.Alloc.NodeID)
		}

		// Update the set of placement nodes
		nodes[0] = node
		s.stack.SetNodes(nodes)

		// Attempt to match the task group
		option, _ := s.stack.Select(missing.TaskGroup)

		if option == nil {
			// If nodes were filtered because of constain mismatches and we
			// couldn't create an allocation then decrementing queued for that
			// task group
			if s.ctx.metrics.NodesFiltered > nodesFiltered {
				s.queuedAllocs[missing.TaskGroup.Name] -= 1

				// If we are annotating the plan, then decrement the desired
				// placements based on whether the node meets the constraints
				if s.eval.AnnotatePlan && s.plan.Annotations != nil &&
					s.plan.Annotations.DesiredTGUpdates != nil {
					desired := s.plan.Annotations.DesiredTGUpdates[missing.TaskGroup.Name]
					desired.Place -= 1
				}
			}

			// Record the current number of nodes filtered in this iteration
			nodesFiltered = s.ctx.metrics.NodesFiltered

			// Check if this task group has already failed
			if metric, ok := s.failedTGAllocs[missing.TaskGroup.Name]; ok {
				metric.CoalescedFailures += 1
				continue
			}
		}

		// Store the available nodes by datacenter
		s.ctx.Metrics().NodesAvailable = s.nodesByDC

		// Set fields based on if we found an allocation option
		if option != nil {
			// Create an allocation for this
			alloc := &structs.Allocation{
				ID:            structs.GenerateUUID(),
				EvalID:        s.eval.ID,
				Name:          missing.Name,
				JobID:         s.job.ID,
				TaskGroup:     missing.TaskGroup.Name,
				Metrics:       s.ctx.Metrics(),
				NodeID:        option.Node.ID,
				TaskResources: option.TaskResources,
				DesiredStatus: structs.AllocDesiredStatusRun,
				ClientStatus:  structs.AllocClientStatusPending,
			}

			s.plan.AppendAlloc(alloc)
		} else {
			// Lazy initialize the failed map
			if s.failedTGAllocs == nil {
				s.failedTGAllocs = make(map[string]*structs.AllocMetric)
			}

			s.failedTGAllocs[missing.TaskGroup.Name] = s.ctx.Metrics()
		}
	}

	return nil
}
