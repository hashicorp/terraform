package nomad

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/scheduler"
	"github.com/hashicorp/raft"
	"github.com/ugorji/go/codec"
)

const (
	// timeTableGranularity is the granularity of index to time tracking
	timeTableGranularity = 5 * time.Minute

	// timeTableLimit is the maximum limit of our tracking
	timeTableLimit = 72 * time.Hour
)

// SnapshotType is prefixed to a record in the FSM snapshot
// so that we can determine the type for restore
type SnapshotType byte

const (
	NodeSnapshot SnapshotType = iota
	JobSnapshot
	IndexSnapshot
	EvalSnapshot
	AllocSnapshot
	TimeTableSnapshot
	PeriodicLaunchSnapshot
	JobSummarySnapshot
)

// nomadFSM implements a finite state machine that is used
// along with Raft to provide strong consistency. We implement
// this outside the Server to avoid exposing this outside the package.
type nomadFSM struct {
	evalBroker         *EvalBroker
	blockedEvals       *BlockedEvals
	periodicDispatcher *PeriodicDispatch
	logOutput          io.Writer
	logger             *log.Logger
	state              *state.StateStore
	timetable          *TimeTable
}

// nomadSnapshot is used to provide a snapshot of the current
// state in a way that can be accessed concurrently with operations
// that may modify the live state.
type nomadSnapshot struct {
	snap      *state.StateSnapshot
	timetable *TimeTable
}

// snapshotHeader is the first entry in our snapshot
type snapshotHeader struct {
}

// NewFSMPath is used to construct a new FSM with a blank state
func NewFSM(evalBroker *EvalBroker, periodic *PeriodicDispatch,
	blocked *BlockedEvals, logOutput io.Writer) (*nomadFSM, error) {
	// Create a state store
	state, err := state.NewStateStore(logOutput)
	if err != nil {
		return nil, err
	}

	fsm := &nomadFSM{
		evalBroker:         evalBroker,
		periodicDispatcher: periodic,
		blockedEvals:       blocked,
		logOutput:          logOutput,
		logger:             log.New(logOutput, "", log.LstdFlags),
		state:              state,
		timetable:          NewTimeTable(timeTableGranularity, timeTableLimit),
	}
	return fsm, nil
}

// Close is used to cleanup resources associated with the FSM
func (n *nomadFSM) Close() error {
	return nil
}

// State is used to return a handle to the current state
func (n *nomadFSM) State() *state.StateStore {
	return n.state
}

// TimeTable returns the time table of transactions
func (n *nomadFSM) TimeTable() *TimeTable {
	return n.timetable
}

func (n *nomadFSM) Apply(log *raft.Log) interface{} {
	buf := log.Data
	msgType := structs.MessageType(buf[0])

	// Witness this write
	n.timetable.Witness(log.Index, time.Now().UTC())

	// Check if this message type should be ignored when unknown. This is
	// used so that new commands can be added with developer control if older
	// versions can safely ignore the command, or if they should crash.
	ignoreUnknown := false
	if msgType&structs.IgnoreUnknownTypeFlag == structs.IgnoreUnknownTypeFlag {
		msgType &= ^structs.IgnoreUnknownTypeFlag
		ignoreUnknown = true
	}

	switch msgType {
	case structs.NodeRegisterRequestType:
		return n.applyUpsertNode(buf[1:], log.Index)
	case structs.NodeDeregisterRequestType:
		return n.applyDeregisterNode(buf[1:], log.Index)
	case structs.NodeUpdateStatusRequestType:
		return n.applyStatusUpdate(buf[1:], log.Index)
	case structs.NodeUpdateDrainRequestType:
		return n.applyDrainUpdate(buf[1:], log.Index)
	case structs.JobRegisterRequestType:
		return n.applyUpsertJob(buf[1:], log.Index)
	case structs.JobDeregisterRequestType:
		return n.applyDeregisterJob(buf[1:], log.Index)
	case structs.EvalUpdateRequestType:
		return n.applyUpdateEval(buf[1:], log.Index)
	case structs.EvalDeleteRequestType:
		return n.applyDeleteEval(buf[1:], log.Index)
	case structs.AllocUpdateRequestType:
		return n.applyAllocUpdate(buf[1:], log.Index)
	case structs.AllocClientUpdateRequestType:
		return n.applyAllocClientUpdate(buf[1:], log.Index)
	case structs.ReconcileJobSummariesRequestType:
		return n.applyReconcileSummaries(buf[1:], log.Index)
	default:
		if ignoreUnknown {
			n.logger.Printf("[WARN] nomad.fsm: ignoring unknown message type (%d), upgrade to newer version", msgType)
			return nil
		} else {
			panic(fmt.Errorf("failed to apply request: %#v", buf))
		}
	}
}

func (n *nomadFSM) applyUpsertNode(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "register_node"}, time.Now())
	var req structs.NodeRegisterRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.UpsertNode(index, req.Node); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpsertNode failed: %v", err)
		return err
	}

	// Unblock evals for the nodes computed node class if it is in a ready
	// state.
	if req.Node.Status == structs.NodeStatusReady {
		n.blockedEvals.Unblock(req.Node.ComputedClass, index)
	}

	return nil
}

func (n *nomadFSM) applyDeregisterNode(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "deregister_node"}, time.Now())
	var req structs.NodeDeregisterRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.DeleteNode(index, req.NodeID); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: DeleteNode failed: %v", err)
		return err
	}
	return nil
}

func (n *nomadFSM) applyStatusUpdate(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "node_status_update"}, time.Now())
	var req structs.NodeUpdateStatusRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.UpdateNodeStatus(index, req.NodeID, req.Status); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpdateNodeStatus failed: %v", err)
		return err
	}

	// Unblock evals for the nodes computed node class if it is in a ready
	// state.
	if req.Status == structs.NodeStatusReady {
		node, err := n.state.NodeByID(req.NodeID)
		if err != nil {
			n.logger.Printf("[ERR] nomad.fsm: looking up node %q failed: %v", req.NodeID, err)
			return err

		}
		n.blockedEvals.Unblock(node.ComputedClass, index)
	}

	return nil
}

func (n *nomadFSM) applyDrainUpdate(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "node_drain_update"}, time.Now())
	var req structs.NodeUpdateDrainRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.UpdateNodeDrain(index, req.NodeID, req.Drain); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpdateNodeDrain failed: %v", err)
		return err
	}
	return nil
}

func (n *nomadFSM) applyUpsertJob(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "register_job"}, time.Now())
	var req structs.JobRegisterRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	// COMPAT: Remove in 0.5
	// Empty maps and slices should be treated as nil to avoid
	// un-intended destructive updates in scheduler since we use
	// reflect.DeepEqual. Starting Nomad 0.4.1, job submission sanatizes
	// the incoming job.
	req.Job.Canonicalize()

	if err := n.state.UpsertJob(index, req.Job); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpsertJob failed: %v", err)
		return err
	}

	// We always add the job to the periodic dispatcher because there is the
	// possibility that the periodic spec was removed and then we should stop
	// tracking it.
	if err := n.periodicDispatcher.Add(req.Job); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: periodicDispatcher.Add failed: %v", err)
		return err
	}

	// If it is periodic, record the time it was inserted. This is necessary for
	// recovering during leader election. It is possible that from the time it
	// is added to when it was suppose to launch, leader election occurs and the
	// job was not launched. In this case, we use the insertion time to
	// determine if a launch was missed.
	if req.Job.IsPeriodic() {
		prevLaunch, err := n.state.PeriodicLaunchByID(req.Job.ID)
		if err != nil {
			n.logger.Printf("[ERR] nomad.fsm: PeriodicLaunchByID failed: %v", err)
			return err
		}

		// Record the insertion time as a launch. We overload the launch table
		// such that the first entry is the insertion time.
		if prevLaunch == nil {
			launch := &structs.PeriodicLaunch{ID: req.Job.ID, Launch: time.Now()}
			if err := n.state.UpsertPeriodicLaunch(index, launch); err != nil {
				n.logger.Printf("[ERR] nomad.fsm: UpsertPeriodicLaunch failed: %v", err)
				return err
			}
		}
	}

	// Check if the parent job is periodic and mark the launch time.
	parentID := req.Job.ParentID
	if parentID != "" {
		parent, err := n.state.JobByID(parentID)
		if err != nil {
			n.logger.Printf("[ERR] nomad.fsm: JobByID(%v) lookup for parent failed: %v", parentID, err)
			return err
		} else if parent == nil {
			// The parent has been deregistered.
			return nil
		}

		if parent.IsPeriodic() {
			t, err := n.periodicDispatcher.LaunchTime(req.Job.ID)
			if err != nil {
				n.logger.Printf("[ERR] nomad.fsm: LaunchTime(%v) failed: %v", req.Job.ID, err)
				return err
			}

			launch := &structs.PeriodicLaunch{ID: parentID, Launch: t}
			if err := n.state.UpsertPeriodicLaunch(index, launch); err != nil {
				n.logger.Printf("[ERR] nomad.fsm: UpsertPeriodicLaunch failed: %v", err)
				return err
			}
		}
	}

	return nil
}

func (n *nomadFSM) applyDeregisterJob(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "deregister_job"}, time.Now())
	var req structs.JobDeregisterRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.DeleteJob(index, req.JobID); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: DeleteJob failed: %v", err)
		return err
	}

	if err := n.periodicDispatcher.Remove(req.JobID); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: periodicDispatcher.Remove failed: %v", err)
		return err
	}

	// We always delete from the periodic launch table because it is possible that
	// the job was updated to be non-perioidic, thus checking if it is periodic
	// doesn't ensure we clean it up properly.
	n.state.DeletePeriodicLaunch(index, req.JobID)

	return nil
}

func (n *nomadFSM) applyUpdateEval(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "update_eval"}, time.Now())
	var req structs.EvalUpdateRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.UpsertEvals(index, req.Evals); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpsertEvals failed: %v", err)
		return err
	}

	for _, eval := range req.Evals {
		if eval.ShouldEnqueue() {
			n.evalBroker.Enqueue(eval)
		} else if eval.ShouldBlock() {
			n.blockedEvals.Block(eval)
		}
	}
	return nil
}

func (n *nomadFSM) applyDeleteEval(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "delete_eval"}, time.Now())
	var req structs.EvalDeleteRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := n.state.DeleteEval(index, req.Evals, req.Allocs); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: DeleteEval failed: %v", err)
		return err
	}
	return nil
}

func (n *nomadFSM) applyAllocUpdate(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "alloc_update"}, time.Now())
	var req structs.AllocUpdateRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	// Attach the job to all the allocations. It is pulled out in the
	// payload to avoid the redundancy of encoding, but should be denormalized
	// prior to being inserted into MemDB.
	if j := req.Job; j != nil {
		for _, alloc := range req.Alloc {
			if alloc.Job == nil && !alloc.TerminalStatus() {
				alloc.Job = j
			}
		}
	}

	// Calculate the total resources of allocations. It is pulled out in the
	// payload to avoid encoding something that can be computed, but should be
	// denormalized prior to being inserted into MemDB.
	for _, alloc := range req.Alloc {
		if alloc.Resources != nil {
			continue
		}

		alloc.Resources = new(structs.Resources)
		for _, task := range alloc.TaskResources {
			alloc.Resources.Add(task)
		}
	}

	if err := n.state.UpsertAllocs(index, req.Alloc); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpsertAllocs failed: %v", err)
		return err
	}
	return nil
}

func (n *nomadFSM) applyAllocClientUpdate(buf []byte, index uint64) interface{} {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "alloc_client_update"}, time.Now())
	var req structs.AllocUpdateRequest
	if err := structs.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}
	if len(req.Alloc) == 0 {
		return nil
	}

	// Updating the allocs with the job id and task group name
	for _, alloc := range req.Alloc {
		if existing, _ := n.state.AllocByID(alloc.ID); existing != nil {
			alloc.JobID = existing.JobID
			alloc.TaskGroup = existing.TaskGroup
		}
	}

	// Update all the client allocations
	if err := n.state.UpdateAllocsFromClient(index, req.Alloc); err != nil {
		n.logger.Printf("[ERR] nomad.fsm: UpdateAllocFromClient failed: %v", err)
		return err
	}

	// Unblock evals for the nodes computed node class if the client has
	// finished running an allocation.
	for _, alloc := range req.Alloc {
		if alloc.ClientStatus == structs.AllocClientStatusComplete ||
			alloc.ClientStatus == structs.AllocClientStatusFailed {
			nodeID := alloc.NodeID
			node, err := n.state.NodeByID(nodeID)
			if err != nil || node == nil {
				n.logger.Printf("[ERR] nomad.fsm: looking up node %q failed: %v", nodeID, err)
				return err

			}
			n.blockedEvals.Unblock(node.ComputedClass, index)
		}
	}

	return nil
}

// applyReconcileSummaries reconciles summaries for all the jobs
func (n *nomadFSM) applyReconcileSummaries(buf []byte, index uint64) interface{} {
	if err := n.state.ReconcileJobSummaries(index); err != nil {
		return err
	}
	return n.reconcileQueuedAllocations(index)
}

func (n *nomadFSM) Snapshot() (raft.FSMSnapshot, error) {
	// Create a new snapshot
	snap, err := n.state.Snapshot()
	if err != nil {
		return nil, err
	}

	ns := &nomadSnapshot{
		snap:      snap,
		timetable: n.timetable,
	}
	return ns, nil
}

func (n *nomadFSM) Restore(old io.ReadCloser) error {
	defer old.Close()

	// Create a new state store
	newState, err := state.NewStateStore(n.logOutput)
	if err != nil {
		return err
	}
	n.state = newState

	// Start the state restore
	restore, err := newState.Restore()
	if err != nil {
		return err
	}
	defer restore.Abort()

	// Create a decoder
	dec := codec.NewDecoder(old, structs.MsgpackHandle)

	// Read in the header
	var header snapshotHeader
	if err := dec.Decode(&header); err != nil {
		return err
	}

	// Populate the new state
	msgType := make([]byte, 1)
	for {
		// Read the message type
		_, err := old.Read(msgType)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// Decode
		switch SnapshotType(msgType[0]) {
		case TimeTableSnapshot:
			if err := n.timetable.Deserialize(dec); err != nil {
				return fmt.Errorf("time table deserialize failed: %v", err)
			}

		case NodeSnapshot:
			node := new(structs.Node)
			if err := dec.Decode(node); err != nil {
				return err
			}
			if err := restore.NodeRestore(node); err != nil {
				return err
			}

		case JobSnapshot:
			job := new(structs.Job)
			if err := dec.Decode(job); err != nil {
				return err
			}

			// COMPAT: Remove in 0.5
			// Empty maps and slices should be treated as nil to avoid
			// un-intended destructive updates in scheduler since we use
			// reflect.DeepEqual. Starting Nomad 0.4.1, job submission sanatizes
			// the incoming job.
			job.Canonicalize()

			if err := restore.JobRestore(job); err != nil {
				return err
			}

		case EvalSnapshot:
			eval := new(structs.Evaluation)
			if err := dec.Decode(eval); err != nil {
				return err
			}
			if err := restore.EvalRestore(eval); err != nil {
				return err
			}

		case AllocSnapshot:
			alloc := new(structs.Allocation)
			if err := dec.Decode(alloc); err != nil {
				return err
			}
			if err := restore.AllocRestore(alloc); err != nil {
				return err
			}

		case IndexSnapshot:
			idx := new(state.IndexEntry)
			if err := dec.Decode(idx); err != nil {
				return err
			}
			if err := restore.IndexRestore(idx); err != nil {
				return err
			}

		case PeriodicLaunchSnapshot:
			launch := new(structs.PeriodicLaunch)
			if err := dec.Decode(launch); err != nil {
				return err
			}
			if err := restore.PeriodicLaunchRestore(launch); err != nil {
				return err
			}

		case JobSummarySnapshot:
			summary := new(structs.JobSummary)
			if err := dec.Decode(summary); err != nil {
				return err
			}
			if err := restore.JobSummaryRestore(summary); err != nil {
				return err
			}

		default:
			return fmt.Errorf("Unrecognized snapshot type: %v", msgType)
		}
	}

	restore.Commit()

	// Create Job Summaries
	// COMPAT 0.4 -> 0.4.1
	// We can remove this in 0.5. This exists so that the server creates job
	// summaries if they were not present previously. When users upgrade to 0.5
	// from 0.4.1, the snapshot will contain job summaries so it will be safe to
	// remove this block.
	index, err := n.state.Index("job_summary")
	if err != nil {
		return fmt.Errorf("couldn't fetch index of job summary table: %v", err)
	}

	// If the index is 0 that means there is no job summary in the snapshot so
	// we will have to create them
	if index == 0 {
		// query the latest index
		latestIndex, err := n.state.LatestIndex()
		if err != nil {
			return fmt.Errorf("unable to query latest index: %v", index)
		}
		if err := n.state.ReconcileJobSummaries(latestIndex); err != nil {
			return fmt.Errorf("error reconciling summaries: %v", err)
		}
	}

	return nil
}

// reconcileSummaries re-calculates the queued allocations for every job that we
// created a Job Summary during the snap shot restore
func (n *nomadFSM) reconcileQueuedAllocations(index uint64) error {
	// Get all the jobs
	iter, err := n.state.Jobs()
	if err != nil {
		return err
	}

	snap, err := n.state.Snapshot()
	if err != nil {
		return fmt.Errorf("unable to create snapshot: %v", err)
	}

	// Invoking the scheduler for every job so that we can populate the number
	// of queued allocations for every job
	for {
		rawJob := iter.Next()
		if rawJob == nil {
			break
		}
		job := rawJob.(*structs.Job)
		planner := &scheduler.Harness{
			State: &snap.StateStore,
		}
		// Create an eval and mark it as requiring annotations and insert that as well
		eval := &structs.Evaluation{
			ID:             structs.GenerateUUID(),
			Priority:       job.Priority,
			Type:           job.Type,
			TriggeredBy:    structs.EvalTriggerJobRegister,
			JobID:          job.ID,
			JobModifyIndex: job.JobModifyIndex + 1,
			Status:         structs.EvalStatusPending,
			AnnotatePlan:   true,
		}

		// Create the scheduler and run it
		sched, err := scheduler.NewScheduler(eval.Type, n.logger, snap, planner)
		if err != nil {
			return err
		}

		if err := sched.Process(eval); err != nil {
			return err
		}

		// Get the job summary from the fsm state store
		summary, err := n.state.JobSummaryByID(job.ID)
		if err != nil {
			return err
		}

		// Add the allocations scheduler has made to queued since these
		// allocations are never getting placed until the scheduler is invoked
		// with a real planner
		if l := len(planner.Plans); l != 1 {
			return fmt.Errorf("unexpected number of plans during restore %d. Please file an issue including the logs", l)
		}
		for _, allocations := range planner.Plans[0].NodeAllocation {
			for _, allocation := range allocations {
				tgSummary, ok := summary.Summary[allocation.TaskGroup]
				if !ok {
					return fmt.Errorf("task group %q not found while updating queued count", allocation.TaskGroup)
				}
				tgSummary.Queued += 1
				summary.Summary[allocation.TaskGroup] = tgSummary
			}
		}

		// Add the queued allocations attached to the evaluation to the queued
		// counter of the job summary
		if l := len(planner.Evals); l != 1 {
			return fmt.Errorf("unexpected number of evals during restore %d. Please file an issue including the logs", l)
		}
		for tg, queued := range planner.Evals[0].QueuedAllocations {
			tgSummary, ok := summary.Summary[tg]
			if !ok {
				return fmt.Errorf("task group %q not found while updating queued count", tg)
			}
			tgSummary.Queued += queued
			summary.Summary[tg] = tgSummary
		}

		if err := n.state.UpsertJobSummary(index, summary); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) Persist(sink raft.SnapshotSink) error {
	defer metrics.MeasureSince([]string{"nomad", "fsm", "persist"}, time.Now())
	// Register the nodes
	encoder := codec.NewEncoder(sink, structs.MsgpackHandle)

	// Write the header
	header := snapshotHeader{}
	if err := encoder.Encode(&header); err != nil {
		sink.Cancel()
		return err
	}

	// Write the time table
	sink.Write([]byte{byte(TimeTableSnapshot)})
	if err := s.timetable.Serialize(encoder); err != nil {
		sink.Cancel()
		return err
	}

	// Write all the data out
	if err := s.persistIndexes(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	if err := s.persistNodes(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	if err := s.persistJobs(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	if err := s.persistEvals(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	if err := s.persistAllocs(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	if err := s.persistPeriodicLaunches(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	if err := s.persistJobSummaries(sink, encoder); err != nil {
		sink.Cancel()
		return err
	}
	return nil
}

func (s *nomadSnapshot) persistIndexes(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {
	// Get all the indexes
	iter, err := s.snap.Indexes()
	if err != nil {
		return err
	}

	for {
		// Get the next item
		raw := iter.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		idx := raw.(*state.IndexEntry)

		// Write out a node registration
		sink.Write([]byte{byte(IndexSnapshot)})
		if err := encoder.Encode(idx); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) persistNodes(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {
	// Get all the nodes
	nodes, err := s.snap.Nodes()
	if err != nil {
		return err
	}

	for {
		// Get the next item
		raw := nodes.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		node := raw.(*structs.Node)

		// Write out a node registration
		sink.Write([]byte{byte(NodeSnapshot)})
		if err := encoder.Encode(node); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) persistJobs(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {
	// Get all the jobs
	jobs, err := s.snap.Jobs()
	if err != nil {
		return err
	}

	for {
		// Get the next item
		raw := jobs.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		job := raw.(*structs.Job)

		// Write out a job registration
		sink.Write([]byte{byte(JobSnapshot)})
		if err := encoder.Encode(job); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) persistEvals(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {
	// Get all the evaluations
	evals, err := s.snap.Evals()
	if err != nil {
		return err
	}

	for {
		// Get the next item
		raw := evals.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		eval := raw.(*structs.Evaluation)

		// Write out the evaluation
		sink.Write([]byte{byte(EvalSnapshot)})
		if err := encoder.Encode(eval); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) persistAllocs(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {
	// Get all the allocations
	allocs, err := s.snap.Allocs()
	if err != nil {
		return err
	}

	for {
		// Get the next item
		raw := allocs.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		alloc := raw.(*structs.Allocation)

		// Write out the evaluation
		sink.Write([]byte{byte(AllocSnapshot)})
		if err := encoder.Encode(alloc); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) persistPeriodicLaunches(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {
	// Get all the jobs
	launches, err := s.snap.PeriodicLaunches()
	if err != nil {
		return err
	}

	for {
		// Get the next item
		raw := launches.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		launch := raw.(*structs.PeriodicLaunch)

		// Write out a job registration
		sink.Write([]byte{byte(PeriodicLaunchSnapshot)})
		if err := encoder.Encode(launch); err != nil {
			return err
		}
	}
	return nil
}

func (s *nomadSnapshot) persistJobSummaries(sink raft.SnapshotSink,
	encoder *codec.Encoder) error {

	summaries, err := s.snap.JobSummaries()
	if err != nil {
		return err
	}

	for {
		raw := summaries.Next()
		if raw == nil {
			break
		}

		jobSummary := raw.(structs.JobSummary)

		sink.Write([]byte{byte(JobSummarySnapshot)})
		if err := encoder.Encode(jobSummary); err != nil {
			return err
		}
	}
	return nil
}

// Release is a no-op, as we just need to GC the pointer
// to the state store snapshot. There is nothing to explicitly
// cleanup.
func (s *nomadSnapshot) Release() {}
