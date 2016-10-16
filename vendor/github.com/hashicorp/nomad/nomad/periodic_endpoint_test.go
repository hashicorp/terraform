package nomad

import (
	"testing"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestPeriodicEndpoint_Force(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	state := s1.fsm.State()
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create and insert a periodic job.
	job := mock.PeriodicJob()
	job.Periodic.ProhibitOverlap = true // Shouldn't affect anything.
	if err := state.UpsertJob(100, job); err != nil {
		t.Fatalf("err: %v", err)
	}
	s1.periodicDispatcher.Add(job)

	// Force launch it.
	req := &structs.PeriodicForceRequest{
		JobID:        job.ID,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}

	// Fetch the response
	var resp structs.PeriodicForceResponse
	if err := msgpackrpc.CallWithCodec(codec, "Periodic.Force", req, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index == 0 {
		t.Fatalf("bad index: %d", resp.Index)
	}

	// Lookup the evaluation
	eval, err := state.EvalByID(resp.EvalID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if eval == nil {
		t.Fatalf("expected eval")
	}
	if eval.CreateIndex != resp.EvalCreateIndex {
		t.Fatalf("index mis-match")
	}
}

func TestPeriodicEndpoint_Force_NonPeriodic(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	state := s1.fsm.State()
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create and insert a non-periodic job.
	job := mock.Job()
	if err := state.UpsertJob(100, job); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Force launch it.
	req := &structs.PeriodicForceRequest{
		JobID:        job.ID,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}

	// Fetch the response
	var resp structs.PeriodicForceResponse
	if err := msgpackrpc.CallWithCodec(codec, "Periodic.Force", req, &resp); err == nil {
		t.Fatalf("Force on non-perodic job should err")
	}
}
