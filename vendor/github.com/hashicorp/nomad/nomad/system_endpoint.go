package nomad

import (
	"fmt"

	"github.com/hashicorp/nomad/nomad/structs"
)

// System endpoint is used to call invoke system tasks.
type System struct {
	srv *Server
}

// GarbageCollect is used to trigger the system to immediately garbage collect nodes, evals
// and jobs.
func (s *System) GarbageCollect(args *structs.GenericRequest, reply *structs.GenericResponse) error {
	if done, err := s.srv.forward("System.GarbageCollect", args, args, reply); done {
		return err
	}

	// Get the states current index
	snapshotIndex, err := s.srv.fsm.State().LatestIndex()
	if err != nil {
		return fmt.Errorf("failed to determine state store's index: %v", err)
	}

	s.srv.evalBroker.Enqueue(s.srv.coreJobEval(structs.CoreJobForceGC, snapshotIndex))
	return nil
}

// ReconcileSummaries reconciles the summaries of all the jobs in the state
// store
func (s *System) ReconcileJobSummaries(args *structs.GenericRequest, reply *structs.GenericResponse) error {
	if done, err := s.srv.forward("System.ReconcileJobSummaries", args, args, reply); done {
		return err
	}

	_, index, err := s.srv.raftApply(structs.ReconcileJobSummariesRequestType, args)
	if err != nil {
		return fmt.Errorf("reconciliation of job summaries failed: %v", err)
	}
	reply.Index = index
	return nil
}
