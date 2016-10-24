package state

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/watch"
)

// IndexEntry is used with the "index" table
// for managing the latest Raft index affecting a table.
type IndexEntry struct {
	Key   string
	Value uint64
}

// The StateStore is responsible for maintaining all the Nomad
// state. It is manipulated by the FSM which maintains consistency
// through the use of Raft. The goals of the StateStore are to provide
// high concurrency for read operations without blocking writes, and
// to provide write availability in the face of reads. EVERY object
// returned as a result of a read against the state store should be
// considered a constant and NEVER modified in place.
type StateStore struct {
	logger *log.Logger
	db     *memdb.MemDB
	watch  *stateWatch
}

// NewStateStore is used to create a new state store
func NewStateStore(logOutput io.Writer) (*StateStore, error) {
	// Create the MemDB
	db, err := memdb.NewMemDB(stateStoreSchema())
	if err != nil {
		return nil, fmt.Errorf("state store setup failed: %v", err)
	}

	// Create the state store
	s := &StateStore{
		logger: log.New(logOutput, "", log.LstdFlags),
		db:     db,
		watch:  newStateWatch(),
	}
	return s, nil
}

// Snapshot is used to create a point in time snapshot. Because
// we use MemDB, we just need to snapshot the state of the underlying
// database.
func (s *StateStore) Snapshot() (*StateSnapshot, error) {
	snap := &StateSnapshot{
		StateStore: StateStore{
			logger: s.logger,
			db:     s.db.Snapshot(),
			watch:  s.watch,
		},
	}
	return snap, nil
}

// Restore is used to optimize the efficiency of rebuilding
// state by minimizing the number of transactions and checking
// overhead.
func (s *StateStore) Restore() (*StateRestore, error) {
	txn := s.db.Txn(true)
	r := &StateRestore{
		txn:   txn,
		watch: s.watch,
		items: watch.NewItems(),
	}
	return r, nil
}

// Watch subscribes a channel to a set of watch items.
func (s *StateStore) Watch(items watch.Items, notify chan struct{}) {
	s.watch.watch(items, notify)
}

// StopWatch unsubscribes a channel from a set of watch items.
func (s *StateStore) StopWatch(items watch.Items, notify chan struct{}) {
	s.watch.stopWatch(items, notify)
}

// UpsertJobSummary upserts a job summary into the state store.
func (s *StateStore) UpsertJobSummary(index uint64, jobSummary *structs.JobSummary) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Update the index
	if err := txn.Insert("job_summary", *jobSummary); err != nil {
		return err
	}

	// Update the indexes table for job summary
	if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Commit()
	return nil
}

// DeleteJobSummary deletes the job summary with the given ID. This is for
// testing purposes only.
func (s *StateStore) DeleteJobSummary(index uint64, id string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Delete the job summary
	if _, err := txn.DeleteAll("job_summary", "id", id); err != nil {
		return fmt.Errorf("deleting job summary failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}
	txn.Commit()
	return nil
}

// UpsertNode is used to register a node or update a node definition
// This is assumed to be triggered by the client, so we retain the value
// of drain which is set by the scheduler.
func (s *StateStore) UpsertNode(index uint64, node *structs.Node) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "nodes"})
	watcher.Add(watch.Item{Node: node.ID})

	// Check if the node already exists
	existing, err := txn.First("nodes", "id", node.ID)
	if err != nil {
		return fmt.Errorf("node lookup failed: %v", err)
	}

	// Setup the indexes correctly
	if existing != nil {
		exist := existing.(*structs.Node)
		node.CreateIndex = exist.CreateIndex
		node.ModifyIndex = index
		node.Drain = exist.Drain // Retain the drain mode
	} else {
		node.CreateIndex = index
		node.ModifyIndex = index
	}

	// Insert the node
	if err := txn.Insert("nodes", node); err != nil {
		return fmt.Errorf("node insert failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"nodes", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// DeleteNode is used to deregister a node
func (s *StateStore) DeleteNode(index uint64, nodeID string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Lookup the node
	existing, err := txn.First("nodes", "id", nodeID)
	if err != nil {
		return fmt.Errorf("node lookup failed: %v", err)
	}
	if existing == nil {
		return fmt.Errorf("node not found")
	}

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "nodes"})
	watcher.Add(watch.Item{Node: nodeID})

	// Delete the node
	if err := txn.Delete("nodes", existing); err != nil {
		return fmt.Errorf("node delete failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"nodes", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// UpdateNodeStatus is used to update the status of a node
func (s *StateStore) UpdateNodeStatus(index uint64, nodeID, status string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "nodes"})
	watcher.Add(watch.Item{Node: nodeID})

	// Lookup the node
	existing, err := txn.First("nodes", "id", nodeID)
	if err != nil {
		return fmt.Errorf("node lookup failed: %v", err)
	}
	if existing == nil {
		return fmt.Errorf("node not found")
	}

	// Copy the existing node
	existingNode := existing.(*structs.Node)
	copyNode := new(structs.Node)
	*copyNode = *existingNode

	// Update the status in the copy
	copyNode.Status = status
	copyNode.ModifyIndex = index

	// Insert the node
	if err := txn.Insert("nodes", copyNode); err != nil {
		return fmt.Errorf("node update failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"nodes", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// UpdateNodeDrain is used to update the drain of a node
func (s *StateStore) UpdateNodeDrain(index uint64, nodeID string, drain bool) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "nodes"})
	watcher.Add(watch.Item{Node: nodeID})

	// Lookup the node
	existing, err := txn.First("nodes", "id", nodeID)
	if err != nil {
		return fmt.Errorf("node lookup failed: %v", err)
	}
	if existing == nil {
		return fmt.Errorf("node not found")
	}

	// Copy the existing node
	existingNode := existing.(*structs.Node)
	copyNode := new(structs.Node)
	*copyNode = *existingNode

	// Update the drain in the copy
	copyNode.Drain = drain
	copyNode.ModifyIndex = index

	// Insert the node
	if err := txn.Insert("nodes", copyNode); err != nil {
		return fmt.Errorf("node update failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"nodes", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// NodeByID is used to lookup a node by ID
func (s *StateStore) NodeByID(nodeID string) (*structs.Node, error) {
	txn := s.db.Txn(false)

	existing, err := txn.First("nodes", "id", nodeID)
	if err != nil {
		return nil, fmt.Errorf("node lookup failed: %v", err)
	}

	if existing != nil {
		return existing.(*structs.Node), nil
	}
	return nil, nil
}

// NodesByIDPrefix is used to lookup nodes by prefix
func (s *StateStore) NodesByIDPrefix(nodeID string) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("nodes", "id_prefix", nodeID)
	if err != nil {
		return nil, fmt.Errorf("node lookup failed: %v", err)
	}

	return iter, nil
}

// Nodes returns an iterator over all the nodes
func (s *StateStore) Nodes() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Walk the entire nodes table
	iter, err := txn.Get("nodes", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// UpsertJob is used to register a job or update a job definition
func (s *StateStore) UpsertJob(index uint64, job *structs.Job) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "jobs"})
	watcher.Add(watch.Item{Job: job.ID})

	// Check if the job already exists
	existing, err := txn.First("jobs", "id", job.ID)
	if err != nil {
		return fmt.Errorf("job lookup failed: %v", err)
	}

	// Setup the indexes correctly
	if existing != nil {
		job.CreateIndex = existing.(*structs.Job).CreateIndex
		job.ModifyIndex = index
		job.JobModifyIndex = index

		// Compute the job status
		var err error
		job.Status, err = s.getJobStatus(txn, job, false)
		if err != nil {
			return fmt.Errorf("setting job status for %q failed: %v", job.ID, err)
		}
	} else {
		job.CreateIndex = index
		job.ModifyIndex = index
		job.JobModifyIndex = index

		// If we are inserting the job for the first time, we don't need to
		// calculate the jobs status as it is known.
		if job.IsPeriodic() {
			job.Status = structs.JobStatusRunning
		} else {
			job.Status = structs.JobStatusPending
		}
	}

	if err := s.updateSummaryWithJob(index, job, watcher, txn); err != nil {
		return fmt.Errorf("unable to create job summary: %v", err)
	}

	// Insert the job
	if err := txn.Insert("jobs", job); err != nil {
		return fmt.Errorf("job insert failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"jobs", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// DeleteJob is used to deregister a job
func (s *StateStore) DeleteJob(index uint64, jobID string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Lookup the node
	existing, err := txn.First("jobs", "id", jobID)
	if err != nil {
		return fmt.Errorf("job lookup failed: %v", err)
	}
	if existing == nil {
		return fmt.Errorf("job not found")
	}

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "jobs"})
	watcher.Add(watch.Item{Job: jobID})
	watcher.Add(watch.Item{Table: "job_summary"})
	watcher.Add(watch.Item{JobSummary: jobID})

	// Delete the node
	if err := txn.Delete("jobs", existing); err != nil {
		return fmt.Errorf("job delete failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"jobs", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	// Delete the job summary
	if _, err = txn.DeleteAll("job_summary", "id", jobID); err != nil {
		return fmt.Errorf("deleing job summary failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// JobByID is used to lookup a job by its ID
func (s *StateStore) JobByID(id string) (*structs.Job, error) {
	txn := s.db.Txn(false)

	existing, err := txn.First("jobs", "id", id)
	if err != nil {
		return nil, fmt.Errorf("job lookup failed: %v", err)
	}

	if existing != nil {
		return existing.(*structs.Job), nil
	}
	return nil, nil
}

// JobsByIDPrefix is used to lookup a job by prefix
func (s *StateStore) JobsByIDPrefix(id string) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("jobs", "id_prefix", id)
	if err != nil {
		return nil, fmt.Errorf("job lookup failed: %v", err)
	}

	return iter, nil
}

// Jobs returns an iterator over all the jobs
func (s *StateStore) Jobs() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Walk the entire jobs table
	iter, err := txn.Get("jobs", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// JobsByPeriodic returns an iterator over all the periodic or non-periodic jobs.
func (s *StateStore) JobsByPeriodic(periodic bool) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("jobs", "periodic", periodic)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// JobsByScheduler returns an iterator over all the jobs with the specific
// scheduler type.
func (s *StateStore) JobsByScheduler(schedulerType string) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Return an iterator for jobs with the specific type.
	iter, err := txn.Get("jobs", "type", schedulerType)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// JobsByGC returns an iterator over all jobs eligible or uneligible for garbage
// collection.
func (s *StateStore) JobsByGC(gc bool) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("jobs", "gc", gc)
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// JobSummary returns a job summary object which matches a specific id.
func (s *StateStore) JobSummaryByID(jobID string) (*structs.JobSummary, error) {
	txn := s.db.Txn(false)

	existing, err := txn.First("job_summary", "id", jobID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		summary := existing.(structs.JobSummary)
		return summary.Copy(), nil
	}

	return nil, nil
}

// JobSummaries walks the entire job summary table and returns all the job
// summary objects
func (s *StateStore) JobSummaries() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("job_summary", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// JobSummaryByPrefix is used to look up Job Summary by id prefix
func (s *StateStore) JobSummaryByPrefix(id string) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("job_summary", "id_prefix", id)
	if err != nil {
		return nil, fmt.Errorf("eval lookup failed: %v", err)
	}

	return iter, nil
}

// UpsertPeriodicLaunch is used to register a launch or update it.
func (s *StateStore) UpsertPeriodicLaunch(index uint64, launch *structs.PeriodicLaunch) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "periodic_launch"})
	watcher.Add(watch.Item{Job: launch.ID})

	// Check if the job already exists
	existing, err := txn.First("periodic_launch", "id", launch.ID)
	if err != nil {
		return fmt.Errorf("periodic launch lookup failed: %v", err)
	}

	// Setup the indexes correctly
	if existing != nil {
		launch.CreateIndex = existing.(*structs.PeriodicLaunch).CreateIndex
		launch.ModifyIndex = index
	} else {
		launch.CreateIndex = index
		launch.ModifyIndex = index
	}

	// Insert the job
	if err := txn.Insert("periodic_launch", launch); err != nil {
		return fmt.Errorf("launch insert failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"periodic_launch", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// DeletePeriodicLaunch is used to delete the periodic launch
func (s *StateStore) DeletePeriodicLaunch(index uint64, jobID string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Lookup the launch
	existing, err := txn.First("periodic_launch", "id", jobID)
	if err != nil {
		return fmt.Errorf("launch lookup failed: %v", err)
	}
	if existing == nil {
		return fmt.Errorf("launch not found")
	}

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "periodic_launch"})
	watcher.Add(watch.Item{Job: jobID})

	// Delete the launch
	if err := txn.Delete("periodic_launch", existing); err != nil {
		return fmt.Errorf("launch delete failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"periodic_launch", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// PeriodicLaunchByID is used to lookup a periodic launch by the periodic job
// ID.
func (s *StateStore) PeriodicLaunchByID(id string) (*structs.PeriodicLaunch, error) {
	txn := s.db.Txn(false)

	existing, err := txn.First("periodic_launch", "id", id)
	if err != nil {
		return nil, fmt.Errorf("periodic launch lookup failed: %v", err)
	}

	if existing != nil {
		return existing.(*structs.PeriodicLaunch), nil
	}
	return nil, nil
}

// PeriodicLaunches returns an iterator over all the periodic launches
func (s *StateStore) PeriodicLaunches() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Walk the entire table
	iter, err := txn.Get("periodic_launch", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// UpsertEvaluation is used to upsert an evaluation
func (s *StateStore) UpsertEvals(index uint64, evals []*structs.Evaluation) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "evals"})

	// Do a nested upsert
	jobs := make(map[string]string, len(evals))
	for _, eval := range evals {
		watcher.Add(watch.Item{Eval: eval.ID})
		if err := s.nestedUpsertEval(txn, index, eval); err != nil {
			return err
		}

		jobs[eval.JobID] = ""
	}

	// Set the job's status
	if err := s.setJobStatuses(index, watcher, txn, jobs, false); err != nil {
		return fmt.Errorf("setting job status failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// nestedUpsertEvaluation is used to nest an evaluation upsert within a transaction
func (s *StateStore) nestedUpsertEval(txn *memdb.Txn, index uint64, eval *structs.Evaluation) error {
	// Lookup the evaluation
	existing, err := txn.First("evals", "id", eval.ID)
	if err != nil {
		return fmt.Errorf("eval lookup failed: %v", err)
	}

	// Update the indexes
	if existing != nil {
		eval.CreateIndex = existing.(*structs.Evaluation).CreateIndex
		eval.ModifyIndex = index
	} else {
		eval.CreateIndex = index
		eval.ModifyIndex = index
	}

	// Update the job summary
	summaryRaw, err := txn.First("job_summary", "id", eval.JobID)
	if err != nil {
		return fmt.Errorf("job summary lookup failed: %v", err)
	}
	if summaryRaw != nil {
		js := summaryRaw.(structs.JobSummary)
		var hasSummaryChanged bool
		for tg, num := range eval.QueuedAllocations {
			if summary, ok := js.Summary[tg]; ok {
				if summary.Queued != num {
					summary.Queued = num
					js.Summary[tg] = summary
					hasSummaryChanged = true
				}
			} else {
				s.logger.Printf("[ERR] state_store: unable to update queued for job %q and task group %q", eval.JobID, tg)
			}
		}

		// Insert the job summary
		if hasSummaryChanged {
			js.ModifyIndex = index
			if err := txn.Insert("job_summary", js); err != nil {
				return fmt.Errorf("job summary insert failed: %v", err)
			}
			if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
				return fmt.Errorf("index update failed: %v", err)
			}
		}
	}

	// Insert the eval
	if err := txn.Insert("evals", eval); err != nil {
		return fmt.Errorf("eval insert failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"evals", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}
	return nil
}

// DeleteEval is used to delete an evaluation
func (s *StateStore) DeleteEval(index uint64, evals []string, allocs []string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()
	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "evals"})
	watcher.Add(watch.Item{Table: "allocs"})

	jobs := make(map[string]string, len(evals))
	for _, eval := range evals {
		existing, err := txn.First("evals", "id", eval)
		if err != nil {
			return fmt.Errorf("eval lookup failed: %v", err)
		}
		if existing == nil {
			continue
		}
		if err := txn.Delete("evals", existing); err != nil {
			return fmt.Errorf("eval delete failed: %v", err)
		}
		watcher.Add(watch.Item{Eval: eval})
		jobs[existing.(*structs.Evaluation).JobID] = ""
	}

	for _, alloc := range allocs {
		existing, err := txn.First("allocs", "id", alloc)
		if err != nil {
			return fmt.Errorf("alloc lookup failed: %v", err)
		}
		if existing == nil {
			continue
		}
		if err := txn.Delete("allocs", existing); err != nil {
			return fmt.Errorf("alloc delete failed: %v", err)
		}
		realAlloc := existing.(*structs.Allocation)
		watcher.Add(watch.Item{Alloc: realAlloc.ID})
		watcher.Add(watch.Item{AllocEval: realAlloc.EvalID})
		watcher.Add(watch.Item{AllocJob: realAlloc.JobID})
		watcher.Add(watch.Item{AllocNode: realAlloc.NodeID})
	}

	// Update the indexes
	if err := txn.Insert("index", &IndexEntry{"evals", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"allocs", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	// Set the job's status
	if err := s.setJobStatuses(index, watcher, txn, jobs, true); err != nil {
		return fmt.Errorf("setting job status failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// EvalByID is used to lookup an eval by its ID
func (s *StateStore) EvalByID(id string) (*structs.Evaluation, error) {
	txn := s.db.Txn(false)

	existing, err := txn.First("evals", "id", id)
	if err != nil {
		return nil, fmt.Errorf("eval lookup failed: %v", err)
	}

	if existing != nil {
		return existing.(*structs.Evaluation), nil
	}
	return nil, nil
}

// EvalsByIDPrefix is used to lookup evaluations by prefix
func (s *StateStore) EvalsByIDPrefix(id string) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("evals", "id_prefix", id)
	if err != nil {
		return nil, fmt.Errorf("eval lookup failed: %v", err)
	}

	return iter, nil
}

// EvalsByJob returns all the evaluations by job id
func (s *StateStore) EvalsByJob(jobID string) ([]*structs.Evaluation, error) {
	txn := s.db.Txn(false)

	// Get an iterator over the node allocations
	iter, err := txn.Get("evals", "job", jobID)
	if err != nil {
		return nil, err
	}

	var out []*structs.Evaluation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Evaluation))
	}
	return out, nil
}

// Evals returns an iterator over all the evaluations
func (s *StateStore) Evals() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Walk the entire table
	iter, err := txn.Get("evals", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// UpdateAllocsFromClient is used to update an allocation based on input

// from a client. While the schedulers are the authority on the allocation for
// most things, some updates are authoritative from the client. Specifically,
// the desired state comes from the schedulers, while the actual state comes
// from clients.
func (s *StateStore) UpdateAllocsFromClient(index uint64, allocs []*structs.Allocation) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Setup the watcher
	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "allocs"})

	// Handle each of the updated allocations
	for _, alloc := range allocs {
		if err := s.nestedUpdateAllocFromClient(txn, watcher, index, alloc); err != nil {
			return err
		}
	}

	// Update the indexes
	if err := txn.Insert("index", &IndexEntry{"allocs", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// nestedUpdateAllocFromClient is used to nest an update of an allocation with client status
func (s *StateStore) nestedUpdateAllocFromClient(txn *memdb.Txn, watcher watch.Items, index uint64, alloc *structs.Allocation) error {
	// Look for existing alloc
	existing, err := txn.First("allocs", "id", alloc.ID)
	if err != nil {
		return fmt.Errorf("alloc lookup failed: %v", err)
	}

	// Nothing to do if this does not exist
	if existing == nil {
		return nil
	}
	exist := existing.(*structs.Allocation)
	// Trigger the watcher
	watcher.Add(watch.Item{Alloc: alloc.ID})
	watcher.Add(watch.Item{AllocEval: exist.EvalID})
	watcher.Add(watch.Item{AllocJob: exist.JobID})
	watcher.Add(watch.Item{AllocNode: exist.NodeID})

	// Copy everything from the existing allocation
	copyAlloc := new(structs.Allocation)
	*copyAlloc = *exist

	// Pull in anything the client is the authority on
	copyAlloc.ClientStatus = alloc.ClientStatus
	copyAlloc.ClientDescription = alloc.ClientDescription
	copyAlloc.TaskStates = alloc.TaskStates

	// Update the modify index
	copyAlloc.ModifyIndex = index

	if err := s.updateSummaryWithAlloc(index, copyAlloc, exist, watcher, txn); err != nil {
		return fmt.Errorf("error updating job summary: %v", err)
	}

	// Update the allocation
	if err := txn.Insert("allocs", copyAlloc); err != nil {
		return fmt.Errorf("alloc insert failed: %v", err)
	}

	// Set the job's status
	forceStatus := ""
	if !copyAlloc.TerminalStatus() {
		forceStatus = structs.JobStatusRunning
	}
	jobs := map[string]string{exist.JobID: forceStatus}
	if err := s.setJobStatuses(index, watcher, txn, jobs, false); err != nil {
		return fmt.Errorf("setting job status failed: %v", err)
	}
	return nil
}

// UpsertAllocs is used to evict a set of allocations
// and allocate new ones at the same time.
func (s *StateStore) UpsertAllocs(index uint64, allocs []*structs.Allocation) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	watcher := watch.NewItems()
	watcher.Add(watch.Item{Table: "allocs"})

	// Handle the allocations
	jobs := make(map[string]string, 1)
	for _, alloc := range allocs {
		existing, err := txn.First("allocs", "id", alloc.ID)
		if err != nil {
			return fmt.Errorf("alloc lookup failed: %v", err)
		}
		exist, _ := existing.(*structs.Allocation)

		if exist == nil {
			alloc.CreateIndex = index
			alloc.ModifyIndex = index
			alloc.AllocModifyIndex = index
		} else {
			alloc.CreateIndex = exist.CreateIndex
			alloc.ModifyIndex = index
			alloc.AllocModifyIndex = index

			// If the scheduler is marking this allocation as lost we do not
			// want to reuse the status of the existing allocation.
			if alloc.ClientStatus != structs.AllocClientStatusLost {
				alloc.ClientStatus = exist.ClientStatus
				alloc.ClientDescription = exist.ClientDescription
			}

			// The job has been denormalized so re-attach the original job
			if alloc.Job == nil {
				alloc.Job = exist.Job
			}
		}

		if err := s.updateSummaryWithAlloc(index, alloc, exist, watcher, txn); err != nil {
			return fmt.Errorf("error updating job summary: %v", err)
		}

		if err := txn.Insert("allocs", alloc); err != nil {
			return fmt.Errorf("alloc insert failed: %v", err)
		}

		// If the allocation is running, force the job to running status.
		forceStatus := ""
		if !alloc.TerminalStatus() {
			forceStatus = structs.JobStatusRunning
		}
		jobs[alloc.JobID] = forceStatus

		watcher.Add(watch.Item{Alloc: alloc.ID})
		watcher.Add(watch.Item{AllocEval: alloc.EvalID})
		watcher.Add(watch.Item{AllocJob: alloc.JobID})
		watcher.Add(watch.Item{AllocNode: alloc.NodeID})
	}

	// Update the indexes
	if err := txn.Insert("index", &IndexEntry{"allocs", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}

	// Set the job's status
	if err := s.setJobStatuses(index, watcher, txn, jobs, false); err != nil {
		return fmt.Errorf("setting job status failed: %v", err)
	}

	txn.Defer(func() { s.watch.notify(watcher) })
	txn.Commit()
	return nil
}

// AllocByID is used to lookup an allocation by its ID
func (s *StateStore) AllocByID(id string) (*structs.Allocation, error) {
	txn := s.db.Txn(false)

	existing, err := txn.First("allocs", "id", id)
	if err != nil {
		return nil, fmt.Errorf("alloc lookup failed: %v", err)
	}

	if existing != nil {
		return existing.(*structs.Allocation), nil
	}
	return nil, nil
}

// AllocsByIDPrefix is used to lookup allocs by prefix
func (s *StateStore) AllocsByIDPrefix(id string) (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	iter, err := txn.Get("allocs", "id_prefix", id)
	if err != nil {
		return nil, fmt.Errorf("alloc lookup failed: %v", err)
	}

	return iter, nil
}

// AllocsByNode returns all the allocations by node
func (s *StateStore) AllocsByNode(node string) ([]*structs.Allocation, error) {
	txn := s.db.Txn(false)

	// Get an iterator over the node allocations, using only the
	// node prefix which ignores the terminal status
	iter, err := txn.Get("allocs", "node_prefix", node)
	if err != nil {
		return nil, err
	}

	var out []*structs.Allocation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Allocation))
	}
	return out, nil
}

// AllocsByNode returns all the allocations by node and terminal status
func (s *StateStore) AllocsByNodeTerminal(node string, terminal bool) ([]*structs.Allocation, error) {
	txn := s.db.Txn(false)

	// Get an iterator over the node allocations
	iter, err := txn.Get("allocs", "node", node, terminal)
	if err != nil {
		return nil, err
	}

	var out []*structs.Allocation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Allocation))
	}
	return out, nil
}

// AllocsByJob returns all the allocations by job id
func (s *StateStore) AllocsByJob(jobID string) ([]*structs.Allocation, error) {
	txn := s.db.Txn(false)

	// Get an iterator over the node allocations
	iter, err := txn.Get("allocs", "job", jobID)
	if err != nil {
		return nil, err
	}

	var out []*structs.Allocation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Allocation))
	}
	return out, nil
}

// AllocsByEval returns all the allocations by eval id
func (s *StateStore) AllocsByEval(evalID string) ([]*structs.Allocation, error) {
	txn := s.db.Txn(false)

	// Get an iterator over the eval allocations
	iter, err := txn.Get("allocs", "eval", evalID)
	if err != nil {
		return nil, err
	}

	var out []*structs.Allocation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Allocation))
	}
	return out, nil
}

// Allocs returns an iterator over all the evaluations
func (s *StateStore) Allocs() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Walk the entire table
	iter, err := txn.Get("allocs", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// LastIndex returns the greatest index value for all indexes
func (s *StateStore) LatestIndex() (uint64, error) {
	indexes, err := s.Indexes()
	if err != nil {
		return 0, err
	}

	var max uint64 = 0
	for {
		raw := indexes.Next()
		if raw == nil {
			break
		}

		// Prepare the request struct
		idx := raw.(*IndexEntry)

		// Determine the max
		if idx.Value > max {
			max = idx.Value
		}
	}

	return max, nil
}

// Index finds the matching index value
func (s *StateStore) Index(name string) (uint64, error) {
	txn := s.db.Txn(false)

	// Lookup the first matching index
	out, err := txn.First("index", "id", name)
	if err != nil {
		return 0, err
	}
	if out == nil {
		return 0, nil
	}
	return out.(*IndexEntry).Value, nil
}

// RemoveIndex is a helper method to remove an index for testing purposes
func (s *StateStore) RemoveIndex(name string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	if _, err := txn.DeleteAll("index", "id", name); err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// Indexes returns an iterator over all the indexes
func (s *StateStore) Indexes() (memdb.ResultIterator, error) {
	txn := s.db.Txn(false)

	// Walk the entire nodes table
	iter, err := txn.Get("index", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// ReconcileJobSummaries re-creates summaries for all jobs present in the state
// store
func (s *StateStore) ReconcileJobSummaries(index uint64) error {
	txn := s.db.Txn(true)
	defer txn.Abort()

	// Get all the jobs
	iter, err := txn.Get("jobs", "id")
	if err != nil {
		return err
	}
	for {
		rawJob := iter.Next()
		if rawJob == nil {
			break
		}
		job := rawJob.(*structs.Job)

		// Create a job summary for the job
		summary := structs.JobSummary{
			JobID:   job.ID,
			Summary: make(map[string]structs.TaskGroupSummary),
		}
		for _, tg := range job.TaskGroups {
			summary.Summary[tg.Name] = structs.TaskGroupSummary{}
		}

		// Find all the allocations for the jobs
		iterAllocs, err := txn.Get("allocs", "job", job.ID)
		if err != nil {
			return err
		}

		// Calculate the summary for the job
		for {
			rawAlloc := iterAllocs.Next()
			if rawAlloc == nil {
				break
			}
			alloc := rawAlloc.(*structs.Allocation)

			// Ignore the allocation if it doesn't belong to the currently
			// registered job
			if alloc.Job.CreateIndex != job.CreateIndex {
				continue
			}

			tg := summary.Summary[alloc.TaskGroup]
			switch alloc.ClientStatus {
			case structs.AllocClientStatusFailed:
				tg.Failed += 1
			case structs.AllocClientStatusLost:
				tg.Lost += 1
			case structs.AllocClientStatusComplete:
				tg.Complete += 1
			case structs.AllocClientStatusRunning:
				tg.Running += 1
			case structs.AllocClientStatusPending:
				tg.Starting += 1
			default:
				s.logger.Printf("[ERR] state_store: invalid client status: %v in allocation %q", alloc.ClientStatus, alloc.ID)
			}
			summary.Summary[alloc.TaskGroup] = tg
		}

		// Set the create index of the summary same as the job's create index
		// and the modify index to the current index
		summary.CreateIndex = job.CreateIndex
		summary.ModifyIndex = index

		// Insert the job summary
		if err := txn.Insert("job_summary", summary); err != nil {
			return fmt.Errorf("error inserting job summary: %v", err)
		}
	}

	// Update the indexes table for job summary
	if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}
	txn.Commit()
	return nil
}

// setJobStatuses is a helper for calling setJobStatus on multiple jobs by ID.
// It takes a map of job IDs to an optional forceStatus string. It returns an
// error if the job doesn't exist or setJobStatus fails.
func (s *StateStore) setJobStatuses(index uint64, watcher watch.Items, txn *memdb.Txn,
	jobs map[string]string, evalDelete bool) error {
	for job, forceStatus := range jobs {
		existing, err := txn.First("jobs", "id", job)
		if err != nil {
			return fmt.Errorf("job lookup failed: %v", err)
		}

		if existing == nil {
			continue
		}

		if err := s.setJobStatus(index, watcher, txn, existing.(*structs.Job), evalDelete, forceStatus); err != nil {
			return err
		}
	}

	return nil
}

// setJobStatus sets the status of the job by looking up associated evaluations
// and allocations. evalDelete should be set to true if setJobStatus is being
// called because an evaluation is being deleted (potentially because of garbage
// collection). If forceStatus is non-empty, the job's status will be set to the
// passed status.
func (s *StateStore) setJobStatus(index uint64, watcher watch.Items, txn *memdb.Txn,
	job *structs.Job, evalDelete bool, forceStatus string) error {

	// Capture the current status so we can check if there is a change
	oldStatus := job.Status
	newStatus := forceStatus

	// If forceStatus is not set, compute the jobs status.
	if forceStatus == "" {
		var err error
		newStatus, err = s.getJobStatus(txn, job, evalDelete)
		if err != nil {
			return err
		}
	}

	// Fast-path if nothing has changed.
	if oldStatus == newStatus {
		return nil
	}

	// The job has changed, so add to watcher.
	watcher.Add(watch.Item{Table: "jobs"})
	watcher.Add(watch.Item{Job: job.ID})

	// Copy and update the existing job
	updated := job.Copy()
	updated.Status = newStatus
	updated.ModifyIndex = index

	// Insert the job
	if err := txn.Insert("jobs", updated); err != nil {
		return fmt.Errorf("job insert failed: %v", err)
	}
	if err := txn.Insert("index", &IndexEntry{"jobs", index}); err != nil {
		return fmt.Errorf("index update failed: %v", err)
	}
	return nil
}

func (s *StateStore) getJobStatus(txn *memdb.Txn, job *structs.Job, evalDelete bool) (string, error) {
	allocs, err := txn.Get("allocs", "job", job.ID)
	if err != nil {
		return "", err
	}

	// If there is a non-terminal allocation, the job is running.
	hasAlloc := false
	for alloc := allocs.Next(); alloc != nil; alloc = allocs.Next() {
		hasAlloc = true
		if !alloc.(*structs.Allocation).TerminalStatus() {
			return structs.JobStatusRunning, nil
		}
	}

	evals, err := txn.Get("evals", "job", job.ID)
	if err != nil {
		return "", err
	}

	hasEval := false
	for eval := evals.Next(); eval != nil; eval = evals.Next() {
		hasEval = true
		if !eval.(*structs.Evaluation).TerminalStatus() {
			return structs.JobStatusPending, nil
		}
	}

	// The job is dead if all the allocations and evals are terminal or if there
	// are no evals because of garbage collection.
	if evalDelete || hasEval || hasAlloc {
		return structs.JobStatusDead, nil
	}

	// If there are no allocations or evaluations it is a new job. If the job is
	// periodic, we mark it as running as it will never have an
	// allocation/evaluation against it.
	if job.IsPeriodic() {
		return structs.JobStatusRunning, nil
	}
	return structs.JobStatusPending, nil
}

// updateSummaryWithJob creates or updates job summaries when new jobs are
// upserted or existing ones are updated
func (s *StateStore) updateSummaryWithJob(index uint64, job *structs.Job,
	watcher watch.Items, txn *memdb.Txn) error {

	existing, err := s.JobSummaryByID(job.ID)
	if err != nil {
		return fmt.Errorf("unable to retrieve summary for job: %v", err)
	}
	var hasSummaryChanged bool
	if existing == nil {
		existing = &structs.JobSummary{
			JobID:       job.ID,
			Summary:     make(map[string]structs.TaskGroupSummary),
			CreateIndex: index,
		}
		hasSummaryChanged = true
	}
	for _, tg := range job.TaskGroups {
		if _, ok := existing.Summary[tg.Name]; !ok {
			newSummary := structs.TaskGroupSummary{
				Complete: 0,
				Failed:   0,
				Running:  0,
				Starting: 0,
			}
			existing.Summary[tg.Name] = newSummary
			hasSummaryChanged = true
		}
	}

	// The job summary has changed, so add to watcher and update the modify
	// index.
	if hasSummaryChanged {
		existing.ModifyIndex = index
		watcher.Add(watch.Item{Table: "job_summary"})
		watcher.Add(watch.Item{JobSummary: job.ID})

		// Update the indexes table for job summary
		if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
			return fmt.Errorf("index update failed: %v", err)
		}
		if err := txn.Insert("job_summary", *existing); err != nil {
			return err
		}
	}

	return nil
}

// updateSummaryWithAlloc updates the job summary when allocations are updated
// or inserted
func (s *StateStore) updateSummaryWithAlloc(index uint64, alloc *structs.Allocation,
	existingAlloc *structs.Allocation, watcher watch.Items, txn *memdb.Txn) error {

	// We don't have to update the summary if the job is missing
	if alloc.Job == nil {
		return nil
	}

	summaryRaw, err := txn.First("job_summary", "id", alloc.JobID)
	if err != nil {
		return fmt.Errorf("unable to lookup job summary for job id %q: %v", err)
	}
	if summaryRaw == nil {
		// Check if the job is de-registered
		rawJob, err := txn.First("jobs", "id", alloc.JobID)
		if err != nil {
			return fmt.Errorf("unable to query job: %v", err)
		}

		// If the job is de-registered then we skip updating it's summary
		if rawJob == nil {
			return nil
		}
		return fmt.Errorf("job summary for job %q is not present", alloc.JobID)
	}
	summary := summaryRaw.(structs.JobSummary)
	jobSummary := summary.Copy()

	// Not updating the job summary because the allocation doesn't belong to the
	// currently registered job
	if jobSummary.CreateIndex != alloc.Job.CreateIndex {
		return nil
	}

	tgSummary, ok := jobSummary.Summary[alloc.TaskGroup]
	if !ok {
		return fmt.Errorf("unable to find task group in the job summary: %v", alloc.TaskGroup)
	}
	var summaryChanged bool
	if existingAlloc == nil {
		switch alloc.DesiredStatus {
		case structs.AllocDesiredStatusStop, structs.AllocDesiredStatusEvict:
			s.logger.Printf("[ERR] state_store: new allocation inserted into state store with id: %v and state: %v",
				alloc.ID, alloc.DesiredStatus)
		}
		switch alloc.ClientStatus {
		case structs.AllocClientStatusPending:
			tgSummary.Starting += 1
			if tgSummary.Queued > 0 {
				tgSummary.Queued -= 1
			}
			summaryChanged = true
		case structs.AllocClientStatusRunning, structs.AllocClientStatusFailed,
			structs.AllocClientStatusComplete:
			s.logger.Printf("[ERR] state_store: new allocation inserted into state store with id: %v and state: %v",
				alloc.ID, alloc.ClientStatus)
		}
	} else if existingAlloc.ClientStatus != alloc.ClientStatus {
		// Incrementing the client of the bin of the current state
		switch alloc.ClientStatus {
		case structs.AllocClientStatusRunning:
			tgSummary.Running += 1
		case structs.AllocClientStatusFailed:
			tgSummary.Failed += 1
		case structs.AllocClientStatusPending:
			tgSummary.Starting += 1
		case structs.AllocClientStatusComplete:
			tgSummary.Complete += 1
		case structs.AllocClientStatusLost:
			tgSummary.Lost += 1
		}

		// Decrementing the count of the bin of the last state
		switch existingAlloc.ClientStatus {
		case structs.AllocClientStatusRunning:
			tgSummary.Running -= 1
		case structs.AllocClientStatusPending:
			tgSummary.Starting -= 1
		case structs.AllocClientStatusLost:
			tgSummary.Lost -= 1
		case structs.AllocClientStatusFailed, structs.AllocClientStatusComplete:
		default:
			s.logger.Printf("[ERR] state_store: invalid old state of allocation with id: %v, and state: %v",
				existingAlloc.ID, existingAlloc.ClientStatus)
		}
		summaryChanged = true
	}
	jobSummary.Summary[alloc.TaskGroup] = tgSummary

	if summaryChanged {
		jobSummary.ModifyIndex = index
		watcher.Add(watch.Item{Table: "job_summary"})
		watcher.Add(watch.Item{JobSummary: alloc.JobID})

		// Update the indexes table for job summary
		if err := txn.Insert("index", &IndexEntry{"job_summary", index}); err != nil {
			return fmt.Errorf("index update failed: %v", err)
		}

		if err := txn.Insert("job_summary", *jobSummary); err != nil {
			return fmt.Errorf("updating job summary failed: %v", err)
		}
	}

	return nil
}

// StateSnapshot is used to provide a point-in-time snapshot
type StateSnapshot struct {
	StateStore
}

// StateRestore is used to optimize the performance when
// restoring state by only using a single large transaction
// instead of thousands of sub transactions
type StateRestore struct {
	txn   *memdb.Txn
	watch *stateWatch
	items watch.Items
}

// Abort is used to abort the restore operation
func (s *StateRestore) Abort() {
	s.txn.Abort()
}

// Commit is used to commit the restore operation
func (s *StateRestore) Commit() {
	s.txn.Defer(func() { s.watch.notify(s.items) })
	s.txn.Commit()
}

// NodeRestore is used to restore a node
func (r *StateRestore) NodeRestore(node *structs.Node) error {
	r.items.Add(watch.Item{Table: "nodes"})
	r.items.Add(watch.Item{Node: node.ID})
	if err := r.txn.Insert("nodes", node); err != nil {
		return fmt.Errorf("node insert failed: %v", err)
	}
	return nil
}

// JobRestore is used to restore a job
func (r *StateRestore) JobRestore(job *structs.Job) error {
	r.items.Add(watch.Item{Table: "jobs"})
	r.items.Add(watch.Item{Job: job.ID})
	if err := r.txn.Insert("jobs", job); err != nil {
		return fmt.Errorf("job insert failed: %v", err)
	}
	return nil
}

// EvalRestore is used to restore an evaluation
func (r *StateRestore) EvalRestore(eval *structs.Evaluation) error {
	r.items.Add(watch.Item{Table: "evals"})
	r.items.Add(watch.Item{Eval: eval.ID})
	if err := r.txn.Insert("evals", eval); err != nil {
		return fmt.Errorf("eval insert failed: %v", err)
	}
	return nil
}

// AllocRestore is used to restore an allocation
func (r *StateRestore) AllocRestore(alloc *structs.Allocation) error {
	r.items.Add(watch.Item{Table: "allocs"})
	r.items.Add(watch.Item{Alloc: alloc.ID})
	r.items.Add(watch.Item{AllocEval: alloc.EvalID})
	r.items.Add(watch.Item{AllocJob: alloc.JobID})
	r.items.Add(watch.Item{AllocNode: alloc.NodeID})
	if err := r.txn.Insert("allocs", alloc); err != nil {
		return fmt.Errorf("alloc insert failed: %v", err)
	}
	return nil
}

// IndexRestore is used to restore an index
func (r *StateRestore) IndexRestore(idx *IndexEntry) error {
	if err := r.txn.Insert("index", idx); err != nil {
		return fmt.Errorf("index insert failed: %v", err)
	}
	return nil
}

// PeriodicLaunchRestore is used to restore a periodic launch.
func (r *StateRestore) PeriodicLaunchRestore(launch *structs.PeriodicLaunch) error {
	r.items.Add(watch.Item{Table: "periodic_launch"})
	r.items.Add(watch.Item{Job: launch.ID})
	if err := r.txn.Insert("periodic_launch", launch); err != nil {
		return fmt.Errorf("periodic launch insert failed: %v", err)
	}
	return nil
}

// JobSummaryRestore is used to restore a job summary
func (r *StateRestore) JobSummaryRestore(jobSummary *structs.JobSummary) error {
	if err := r.txn.Insert("job_summary", *jobSummary); err != nil {
		return fmt.Errorf("job summary insert failed: %v", err)
	}
	return nil
}

// stateWatch holds shared state for watching updates. This is
// outside of StateStore so it can be shared with snapshots.
type stateWatch struct {
	items map[watch.Item]*NotifyGroup
	l     sync.Mutex
}

// newStateWatch creates a new stateWatch for change notification.
func newStateWatch() *stateWatch {
	return &stateWatch{
		items: make(map[watch.Item]*NotifyGroup),
	}
}

// watch subscribes a channel to the given watch items.
func (w *stateWatch) watch(items watch.Items, ch chan struct{}) {
	w.l.Lock()
	defer w.l.Unlock()

	for item, _ := range items {
		grp, ok := w.items[item]
		if !ok {
			grp = new(NotifyGroup)
			w.items[item] = grp
		}
		grp.Wait(ch)
	}
}

// stopWatch unsubscribes a channel from the given watch items.
func (w *stateWatch) stopWatch(items watch.Items, ch chan struct{}) {
	w.l.Lock()
	defer w.l.Unlock()

	for item, _ := range items {
		if grp, ok := w.items[item]; ok {
			grp.Clear(ch)
			if grp.Empty() {
				delete(w.items, item)
			}
		}
	}
}

// notify is used to fire notifications on the given watch items.
func (w *stateWatch) notify(items watch.Items) {
	w.l.Lock()
	defer w.l.Unlock()

	for wi, _ := range items {
		if grp, ok := w.items[wi]; ok {
			grp.Notify()
		}
	}
}
