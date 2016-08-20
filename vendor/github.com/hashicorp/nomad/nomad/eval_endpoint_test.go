package nomad

import (
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func TestEvalEndpoint_GetEval(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	eval1 := mock.Eval()
	s1.fsm.State().UpsertEvals(1000, []*structs.Evaluation{eval1})

	// Lookup the eval
	get := &structs.EvalSpecificRequest{
		EvalID:       eval1.ID,
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	var resp structs.SingleEvalResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.GetEval", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}

	if !reflect.DeepEqual(eval1, resp.Eval) {
		t.Fatalf("bad: %#v %#v", eval1, resp.Eval)
	}

	// Lookup non-existing node
	get.EvalID = structs.GenerateUUID()
	if err := msgpackrpc.CallWithCodec(codec, "Eval.GetEval", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}
	if resp.Eval != nil {
		t.Fatalf("unexpected eval")
	}
}

func TestEvalEndpoint_GetEval_Blocking(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	state := s1.fsm.State()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the evals
	eval1 := mock.Eval()
	eval2 := mock.Eval()

	// First create an unrelated eval
	time.AfterFunc(100*time.Millisecond, func() {
		err := state.UpsertEvals(100, []*structs.Evaluation{eval1})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	// Upsert the eval we are watching later
	time.AfterFunc(200*time.Millisecond, func() {
		err := state.UpsertEvals(200, []*structs.Evaluation{eval2})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	// Lookup the eval
	req := &structs.EvalSpecificRequest{
		EvalID: eval2.ID,
		QueryOptions: structs.QueryOptions{
			Region:        "global",
			MinQueryIndex: 50,
		},
	}
	var resp structs.SingleEvalResponse
	start := time.Now()
	if err := msgpackrpc.CallWithCodec(codec, "Eval.GetEval", req, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp)
	}
	if resp.Index != 200 {
		t.Fatalf("Bad index: %d %d", resp.Index, 200)
	}
	if resp.Eval == nil || resp.Eval.ID != eval2.ID {
		t.Fatalf("bad: %#v", resp.Eval)
	}

	// Eval delete triggers watches
	time.AfterFunc(100*time.Millisecond, func() {
		err := state.DeleteEval(300, []string{eval2.ID}, []string{})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	req.QueryOptions.MinQueryIndex = 250
	var resp2 structs.SingleEvalResponse
	start = time.Now()
	if err := msgpackrpc.CallWithCodec(codec, "Eval.GetEval", req, &resp2); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp2)
	}
	if resp2.Index != 300 {
		t.Fatalf("Bad index: %d %d", resp2.Index, 300)
	}
	if resp2.Eval != nil {
		t.Fatalf("bad: %#v", resp2.Eval)
	}
}

func TestEvalEndpoint_Dequeue(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)

	// Dequeue the eval
	get := &structs.EvalDequeueRequest{
		Schedulers:   defaultSched,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.EvalDequeueResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Dequeue", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(eval1, resp.Eval) {
		t.Fatalf("bad: %v %v", eval1, resp.Eval)
	}

	// Ensure outstanding
	token, ok := s1.evalBroker.Outstanding(eval1.ID)
	if !ok {
		t.Fatalf("should be outstanding")
	}
	if token != resp.Token {
		t.Fatalf("bad token: %#v %#v", token, resp.Token)
	}
}

func TestEvalEndpoint_Ack(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the register request
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)
	out, token, err := s1.evalBroker.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("missing eval")
	}

	// Ack the eval
	get := &structs.EvalAckRequest{
		EvalID:       out.ID,
		Token:        token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Ack", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure outstanding
	if _, ok := s1.evalBroker.Outstanding(eval1.ID); ok {
		t.Fatalf("should not be outstanding")
	}
}

func TestEvalEndpoint_Nack(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		// Disable all of the schedulers so we can manually dequeue
		// evals and check the queue status
		c.NumSchedulers = 0
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the register request
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)
	out, token, _ := s1.evalBroker.Dequeue(defaultSched, time.Second)
	if out == nil {
		t.Fatalf("missing eval")
	}

	// Nack the eval
	get := &structs.EvalAckRequest{
		EvalID:       out.ID,
		Token:        token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Nack", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure outstanding
	if _, ok := s1.evalBroker.Outstanding(eval1.ID); ok {
		t.Fatalf("should not be outstanding")
	}

	// Should get it back
	out2, _, _ := s1.evalBroker.Dequeue(defaultSched, time.Second)
	if out2 != out {
		t.Fatalf("nack failed")
	}
}

func TestEvalEndpoint_Update(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the register request
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)
	out, token, err := s1.evalBroker.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("missing eval")
	}

	// Update the eval
	eval2 := eval1.Copy()
	eval2.Status = structs.EvalStatusComplete

	get := &structs.EvalUpdateRequest{
		Evals:        []*structs.Evaluation{eval2},
		EvalToken:    token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Update", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure updated
	outE, err := s1.fsm.State().EvalByID(eval2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if outE.Status != structs.EvalStatusComplete {
		t.Fatalf("Bad: %#v", out)
	}
}

func TestEvalEndpoint_Create(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the register request
	prev := mock.Eval()
	s1.evalBroker.Enqueue(prev)
	out, token, err := s1.evalBroker.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("missing eval")
	}

	// Create the register request
	eval1 := mock.Eval()
	eval1.PreviousEval = prev.ID
	get := &structs.EvalUpdateRequest{
		Evals:        []*structs.Evaluation{eval1},
		EvalToken:    token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Create", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Ensure created
	outE, err := s1.fsm.State().EvalByID(eval1.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	eval1.CreateIndex = resp.Index
	eval1.ModifyIndex = resp.Index
	if !reflect.DeepEqual(eval1, outE) {
		t.Fatalf("Bad: %#v %#v", outE, eval1)
	}
}

func TestEvalEndpoint_Reap(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	eval1 := mock.Eval()
	s1.fsm.State().UpsertEvals(1000, []*structs.Evaluation{eval1})

	// Reap the eval
	get := &structs.EvalDeleteRequest{
		Evals:        []string{eval1.ID},
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Reap", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index == 0 {
		t.Fatalf("Bad index: %d", resp.Index)
	}

	// Ensure deleted
	outE, err := s1.fsm.State().EvalByID(eval1.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if outE != nil {
		t.Fatalf("Bad: %#v", outE)
	}
}

func TestEvalEndpoint_List(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	eval1 := mock.Eval()
	eval1.ID = "aaaaaaaa-3350-4b4b-d185-0e1992ed43e9"
	eval2 := mock.Eval()
	eval2.ID = "aaaabbbb-3350-4b4b-d185-0e1992ed43e9"
	s1.fsm.State().UpsertEvals(1000, []*structs.Evaluation{eval1, eval2})

	// Lookup the eval
	get := &structs.EvalListRequest{
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	var resp structs.EvalListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.List", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}

	if len(resp.Evaluations) != 2 {
		t.Fatalf("bad: %#v", resp.Evaluations)
	}

	// Lookup the eval by prefix
	get = &structs.EvalListRequest{
		QueryOptions: structs.QueryOptions{Region: "global", Prefix: "aaaabb"},
	}
	var resp2 structs.EvalListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.List", get, &resp2); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp2.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp2.Index, 1000)
	}

	if len(resp2.Evaluations) != 1 {
		t.Fatalf("bad: %#v", resp2.Evaluations)
	}

}

func TestEvalEndpoint_List_Blocking(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	state := s1.fsm.State()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the ieval
	eval := mock.Eval()

	// Upsert eval triggers watches
	time.AfterFunc(100*time.Millisecond, func() {
		if err := state.UpsertEvals(2, []*structs.Evaluation{eval}); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	req := &structs.EvalListRequest{
		QueryOptions: structs.QueryOptions{
			Region:        "global",
			MinQueryIndex: 1,
		},
	}
	start := time.Now()
	var resp structs.EvalListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.List", req, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp)
	}
	if resp.Index != 2 {
		t.Fatalf("Bad index: %d %d", resp.Index, 2)
	}
	if len(resp.Evaluations) != 1 || resp.Evaluations[0].ID != eval.ID {
		t.Fatalf("bad: %#v", resp.Evaluations)
	}

	// Eval deletion triggers watches
	time.AfterFunc(100*time.Millisecond, func() {
		if err := state.DeleteEval(3, []string{eval.ID}, nil); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	req.MinQueryIndex = 2
	start = time.Now()
	var resp2 structs.EvalListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.List", req, &resp2); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp2)
	}
	if resp2.Index != 3 {
		t.Fatalf("Bad index: %d %d", resp2.Index, 3)
	}
	if len(resp2.Evaluations) != 0 {
		t.Fatalf("bad: %#v", resp2.Evaluations)
	}
}

func TestEvalEndpoint_Allocations(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	alloc1 := mock.Alloc()
	alloc2 := mock.Alloc()
	alloc2.EvalID = alloc1.EvalID
	state := s1.fsm.State()
	state.UpsertJobSummary(998, mock.JobSummary(alloc1.JobID))
	state.UpsertJobSummary(999, mock.JobSummary(alloc2.JobID))
	err := state.UpsertAllocs(1000,
		[]*structs.Allocation{alloc1, alloc2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Lookup the eval
	get := &structs.EvalSpecificRequest{
		EvalID:       alloc1.EvalID,
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	var resp structs.EvalAllocationsResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Allocations", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}

	if len(resp.Allocations) != 2 {
		t.Fatalf("bad: %#v", resp.Allocations)
	}
}

func TestEvalEndpoint_Allocations_Blocking(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	state := s1.fsm.State()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the allocs
	alloc1 := mock.Alloc()
	alloc2 := mock.Alloc()

	// Upsert an unrelated alloc first
	time.AfterFunc(100*time.Millisecond, func() {
		state.UpsertJobSummary(99, mock.JobSummary(alloc1.JobID))
		err := state.UpsertAllocs(100, []*structs.Allocation{alloc1})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	// Upsert an alloc which will trigger the watch later
	time.AfterFunc(200*time.Millisecond, func() {
		state.UpsertJobSummary(199, mock.JobSummary(alloc2.JobID))
		err := state.UpsertAllocs(200, []*structs.Allocation{alloc2})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	// Lookup the eval
	get := &structs.EvalSpecificRequest{
		EvalID: alloc2.EvalID,
		QueryOptions: structs.QueryOptions{
			Region:        "global",
			MinQueryIndex: 50,
		},
	}
	var resp structs.EvalAllocationsResponse
	start := time.Now()
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Allocations", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp)
	}
	if resp.Index != 200 {
		t.Fatalf("Bad index: %d %d", resp.Index, 200)
	}
	if len(resp.Allocations) != 1 || resp.Allocations[0].ID != alloc2.ID {
		t.Fatalf("bad: %#v", resp.Allocations)
	}
}

func TestEvalEndpoint_Reblock_NonExistent(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the register request
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)
	out, token, err := s1.evalBroker.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("missing eval")
	}

	get := &structs.EvalUpdateRequest{
		Evals:        []*structs.Evaluation{eval1},
		EvalToken:    token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Reblock", get, &resp); err == nil {
		t.Fatalf("expect error since eval does not exist")
	}
}

func TestEvalEndpoint_Reblock_NonBlocked(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the eval
	eval1 := mock.Eval()
	s1.evalBroker.Enqueue(eval1)

	// Insert it into the state store
	if err := s1.fsm.State().UpsertEvals(1000, []*structs.Evaluation{eval1}); err != nil {
		t.Fatal(err)
	}

	out, token, err := s1.evalBroker.Dequeue(defaultSched, 2*time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("missing eval")
	}

	get := &structs.EvalUpdateRequest{
		Evals:        []*structs.Evaluation{eval1},
		EvalToken:    token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Reblock", get, &resp); err == nil {
		t.Fatalf("should error since eval was not in blocked state", err)
	}
}

func TestEvalEndpoint_Reblock(t *testing.T) {
	s1 := testServer(t, func(c *Config) {
		c.NumSchedulers = 0 // Prevent automatic dequeue
	})
	defer s1.Shutdown()
	codec := rpcClient(t, s1)

	testutil.WaitForResult(func() (bool, error) {
		return s1.evalBroker.Enabled(), nil
	}, func(err error) {
		t.Fatalf("should enable eval broker")
	})

	// Create the eval
	eval1 := mock.Eval()
	eval1.Status = structs.EvalStatusBlocked
	s1.evalBroker.Enqueue(eval1)

	// Insert it into the state store
	if err := s1.fsm.State().UpsertEvals(1000, []*structs.Evaluation{eval1}); err != nil {
		t.Fatal(err)
	}

	out, token, err := s1.evalBroker.Dequeue(defaultSched, 7*time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("bad: %v", out)
	}

	get := &structs.EvalUpdateRequest{
		Evals:        []*structs.Evaluation{eval1},
		EvalToken:    token,
		WriteRequest: structs.WriteRequest{Region: "global"},
	}
	var resp structs.GenericResponse
	if err := msgpackrpc.CallWithCodec(codec, "Eval.Reblock", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check that it is blocked
	bStats := s1.blockedEvals.Stats()
	if bStats.TotalBlocked+bStats.TotalEscaped == 0 {
		t.Fatalf("ReblockEval didn't insert eval into the blocked eval tracker")
	}
}
