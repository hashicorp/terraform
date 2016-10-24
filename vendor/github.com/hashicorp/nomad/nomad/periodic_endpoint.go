package nomad

import (
	"fmt"
	"time"

	"github.com/armon/go-metrics"
	"github.com/hashicorp/nomad/nomad/structs"
)

// Periodic endpoint is used for periodic job interactions
type Periodic struct {
	srv *Server
}

// Force is used to force a new instance of a periodic job
func (p *Periodic) Force(args *structs.PeriodicForceRequest, reply *structs.PeriodicForceResponse) error {
	if done, err := p.srv.forward("Periodic.Force", args, args, reply); done {
		return err
	}
	defer metrics.MeasureSince([]string{"nomad", "periodic", "force"}, time.Now())

	// Validate the arguments
	if args.JobID == "" {
		return fmt.Errorf("missing job ID for evaluation")
	}

	// Lookup the job
	snap, err := p.srv.fsm.State().Snapshot()
	if err != nil {
		return err
	}
	job, err := snap.JobByID(args.JobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if !job.IsPeriodic() {
		return fmt.Errorf("can't force launch non-periodic job")
	}

	// Force run the job.
	eval, err := p.srv.periodicDispatcher.ForceRun(job.ID)
	if err != nil {
		return fmt.Errorf("force launch for job %q failed: %v", job.ID, err)
	}

	reply.EvalID = eval.ID
	reply.EvalCreateIndex = eval.CreateIndex
	reply.Index = eval.CreateIndex
	return nil
}
