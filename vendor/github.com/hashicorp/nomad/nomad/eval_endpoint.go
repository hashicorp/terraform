package nomad

import (
	"fmt"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/watch"
)

const (
	// DefaultDequeueTimeout is used if no dequeue timeout is provided
	DefaultDequeueTimeout = time.Second
)

// Eval endpoint is used for eval interactions
type Eval struct {
	srv *Server
}

// GetEval is used to request information about a specific evaluation
func (e *Eval) GetEval(args *structs.EvalSpecificRequest,
	reply *structs.SingleEvalResponse) error {
	if done, err := e.srv.forward("Eval.GetEval", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "get_eval"}, time.Now())

	// Setup the blocking query
	opts := blockingOptions{
		queryOpts: &args.QueryOptions,
		queryMeta: &reply.QueryMeta,
		watch:     watch.NewItems(watch.Item{Eval: args.EvalID}),
		run: func() error {
			// Look for the job
			snap, err := e.srv.fsm.State().Snapshot()
			if err != nil {
				return err
			}
			out, err := snap.EvalByID(args.EvalID)
			if err != nil {
				return err
			}

			// Setup the output
			reply.Eval = out
			if out != nil {
				reply.Index = out.ModifyIndex
			} else {
				// Use the last index that affected the nodes table
				index, err := snap.Index("evals")
				if err != nil {
					return err
				}
				reply.Index = index
			}

			// Set the query response
			e.srv.setQueryMeta(&reply.QueryMeta)
			return nil
		}}
	return e.srv.blockingRPC(&opts)
}

// Dequeue is used to dequeue a pending evaluation
func (e *Eval) Dequeue(args *structs.EvalDequeueRequest,
	reply *structs.EvalDequeueResponse) error {
	if done, err := e.srv.forward("Eval.Dequeue", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "dequeue"}, time.Now())

	// Ensure there is at least one scheduler
	if len(args.Schedulers) == 0 {
		return fmt.Errorf("dequeue requires at least one scheduler type")
	}

	// Ensure there is a default timeout
	if args.Timeout <= 0 {
		args.Timeout = DefaultDequeueTimeout
	}

	// Attempt the dequeue
	eval, token, err := e.srv.evalBroker.Dequeue(args.Schedulers, args.Timeout)
	if err != nil {
		return err
	}

	// Provide the output if any
	if eval != nil {
		reply.Eval = eval
		reply.Token = token
	}

	// Set the query response
	e.srv.setQueryMeta(&reply.QueryMeta)
	return nil
}

// Ack is used to acknowledge completion of a dequeued evaluation
func (e *Eval) Ack(args *structs.EvalAckRequest,
	reply *structs.GenericResponse) error {
	if done, err := e.srv.forward("Eval.Ack", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "ack"}, time.Now())

	// Ack the EvalID
	if err := e.srv.evalBroker.Ack(args.EvalID, args.Token); err != nil {
		return err
	}
	return nil
}

// NAck is used to negative acknowledge completion of a dequeued evaluation
func (e *Eval) Nack(args *structs.EvalAckRequest,
	reply *structs.GenericResponse) error {
	if done, err := e.srv.forward("Eval.Nack", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "nack"}, time.Now())

	// Nack the EvalID
	if err := e.srv.evalBroker.Nack(args.EvalID, args.Token); err != nil {
		return err
	}
	return nil
}

// Update is used to perform an update of an Eval if it is outstanding.
func (e *Eval) Update(args *structs.EvalUpdateRequest,
	reply *structs.GenericResponse) error {
	if done, err := e.srv.forward("Eval.Update", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "update"}, time.Now())

	// Ensure there is only a single update with token
	if len(args.Evals) != 1 {
		return fmt.Errorf("only a single eval can be updated")
	}
	eval := args.Evals[0]

	// Verify the evaluation is outstanding, and that the tokens match.
	if err := e.srv.evalBroker.OutstandingReset(eval.ID, args.EvalToken); err != nil {
		return err
	}

	// Update via Raft
	_, index, err := e.srv.raftApply(structs.EvalUpdateRequestType, args)
	if err != nil {
		return err
	}

	// Update the index
	reply.Index = index
	return nil
}

// Create is used to make a new evaluation
func (e *Eval) Create(args *structs.EvalUpdateRequest,
	reply *structs.GenericResponse) error {
	if done, err := e.srv.forward("Eval.Create", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "create"}, time.Now())

	// Ensure there is only a single update with token
	if len(args.Evals) != 1 {
		return fmt.Errorf("only a single eval can be created")
	}
	eval := args.Evals[0]

	// Verify the parent evaluation is outstanding, and that the tokens match.
	if err := e.srv.evalBroker.OutstandingReset(eval.PreviousEval, args.EvalToken); err != nil {
		return err
	}

	// Look for the eval
	snap, err := e.srv.fsm.State().Snapshot()
	if err != nil {
		return err
	}
	out, err := snap.EvalByID(eval.ID)
	if err != nil {
		return err
	}
	if out != nil {
		return fmt.Errorf("evaluation already exists")
	}

	// Update via Raft
	_, index, err := e.srv.raftApply(structs.EvalUpdateRequestType, args)
	if err != nil {
		return err
	}

	// Update the index
	reply.Index = index
	return nil
}

// Reblock is used to reinsert an existing blocked evaluation into the blocked
// evaluation tracker.
func (e *Eval) Reblock(args *structs.EvalUpdateRequest, reply *structs.GenericResponse) error {
	if done, err := e.srv.forward("Eval.Reblock", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "reblock"}, time.Now())

	// Ensure there is only a single update with token
	if len(args.Evals) != 1 {
		return fmt.Errorf("only a single eval can be reblocked")
	}
	eval := args.Evals[0]

	// Verify the evaluation is outstanding, and that the tokens match.
	if err := e.srv.evalBroker.OutstandingReset(eval.ID, args.EvalToken); err != nil {
		return err
	}

	// Look for the eval
	snap, err := e.srv.fsm.State().Snapshot()
	if err != nil {
		return err
	}
	out, err := snap.EvalByID(eval.ID)
	if err != nil {
		return err
	}
	if out == nil {
		return fmt.Errorf("evaluation does not exist")
	}
	if out.Status != structs.EvalStatusBlocked {
		return fmt.Errorf("evaluation not blocked")
	}

	// Reblock the eval
	e.srv.blockedEvals.Reblock(eval, args.EvalToken)
	return nil
}

// Reap is used to cleanup dead evaluations and allocations
func (e *Eval) Reap(args *structs.EvalDeleteRequest,
	reply *structs.GenericResponse) error {
	if done, err := e.srv.forward("Eval.Reap", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "reap"}, time.Now())

	// Update via Raft
	_, index, err := e.srv.raftApply(structs.EvalDeleteRequestType, args)
	if err != nil {
		return err
	}

	// Update the index
	reply.Index = index
	return nil
}

// List is used to get a list of the evaluations in the system
func (e *Eval) List(args *structs.EvalListRequest,
	reply *structs.EvalListResponse) error {
	if done, err := e.srv.forward("Eval.List", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "list"}, time.Now())

	// Setup the blocking query
	opts := blockingOptions{
		queryOpts: &args.QueryOptions,
		queryMeta: &reply.QueryMeta,
		watch:     watch.NewItems(watch.Item{Table: "evals"}),
		run: func() error {
			// Scan all the evaluations
			snap, err := e.srv.fsm.State().Snapshot()
			if err != nil {
				return err
			}
			var iter memdb.ResultIterator
			if prefix := args.QueryOptions.Prefix; prefix != "" {
				iter, err = snap.EvalsByIDPrefix(prefix)
			} else {
				iter, err = snap.Evals()
			}
			if err != nil {
				return err
			}

			var evals []*structs.Evaluation
			for {
				raw := iter.Next()
				if raw == nil {
					break
				}
				eval := raw.(*structs.Evaluation)
				evals = append(evals, eval)
			}
			reply.Evaluations = evals

			// Use the last index that affected the jobs table
			index, err := snap.Index("evals")
			if err != nil {
				return err
			}
			reply.Index = index

			// Set the query response
			e.srv.setQueryMeta(&reply.QueryMeta)
			return nil
		}}
	return e.srv.blockingRPC(&opts)
}

// Allocations is used to list the allocations for an evaluation
func (e *Eval) Allocations(args *structs.EvalSpecificRequest,
	reply *structs.EvalAllocationsResponse) error {
	if done, err := e.srv.forward("Eval.Allocations", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "eval", "allocations"}, time.Now())

	// Setup the blocking query
	opts := blockingOptions{
		queryOpts: &args.QueryOptions,
		queryMeta: &reply.QueryMeta,
		watch:     watch.NewItems(watch.Item{AllocEval: args.EvalID}),
		run: func() error {
			// Capture the allocations
			snap, err := e.srv.fsm.State().Snapshot()
			if err != nil {
				return err
			}
			allocs, err := snap.AllocsByEval(args.EvalID)
			if err != nil {
				return err
			}

			// Convert to a stub
			if len(allocs) > 0 {
				reply.Allocations = make([]*structs.AllocListStub, 0, len(allocs))
				for _, alloc := range allocs {
					reply.Allocations = append(reply.Allocations, alloc.Stub())
				}
			}

			// Use the last index that affected the allocs table
			index, err := snap.Index("allocs")
			if err != nil {
				return err
			}
			reply.Index = index

			// Set the query response
			e.srv.setQueryMeta(&reply.QueryMeta)
			return nil
		}}
	return e.srv.blockingRPC(&opts)
}
