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

func TestAllocEndpoint_List(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	alloc := mock.Alloc()
	summary := mock.JobSummary(alloc.JobID)
	state := s1.fsm.State()

	if err := state.UpsertJobSummary(999, summary); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := state.UpsertAllocs(1000, []*structs.Allocation{alloc}); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Lookup the allocations
	get := &structs.AllocListRequest{
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	var resp structs.AllocListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.List", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}

	if len(resp.Allocations) != 1 {
		t.Fatalf("bad: %#v", resp.Allocations)
	}
	if resp.Allocations[0].ID != alloc.ID {
		t.Fatalf("bad: %#v", resp.Allocations[0])
	}

	// Lookup the allocations by prefix
	get = &structs.AllocListRequest{
		QueryOptions: structs.QueryOptions{Region: "global", Prefix: alloc.ID[:4]},
	}

	var resp2 structs.AllocListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.List", get, &resp2); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp2.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp2.Index, 1000)
	}

	if len(resp2.Allocations) != 1 {
		t.Fatalf("bad: %#v", resp2.Allocations)
	}
	if resp2.Allocations[0].ID != alloc.ID {
		t.Fatalf("bad: %#v", resp2.Allocations[0])
	}
}

func TestAllocEndpoint_List_Blocking(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	state := s1.fsm.State()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the alloc
	alloc := mock.Alloc()

	summary := mock.JobSummary(alloc.JobID)
	if err := state.UpsertJobSummary(1, summary); err != nil {
		t.Fatalf("err: %v", err)
	}
	// Upsert alloc triggers watches
	time.AfterFunc(100*time.Millisecond, func() {
		if err := state.UpsertAllocs(2, []*structs.Allocation{alloc}); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	req := &structs.AllocListRequest{
		QueryOptions: structs.QueryOptions{
			Region:        "global",
			MinQueryIndex: 1,
		},
	}
	start := time.Now()
	var resp structs.AllocListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.List", req, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp)
	}
	if resp.Index != 2 {
		t.Fatalf("Bad index: %d %d", resp.Index, 2)
	}
	if len(resp.Allocations) != 1 || resp.Allocations[0].ID != alloc.ID {
		t.Fatalf("bad: %#v", resp.Allocations)
	}

	// Client updates trigger watches
	alloc2 := mock.Alloc()
	alloc2.ID = alloc.ID
	alloc2.ClientStatus = structs.AllocClientStatusRunning
	time.AfterFunc(100*time.Millisecond, func() {
		state.UpsertJobSummary(3, mock.JobSummary(alloc2.JobID))
		if err := state.UpdateAllocsFromClient(4, []*structs.Allocation{alloc2}); err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	req.MinQueryIndex = 3
	start = time.Now()
	var resp2 structs.AllocListResponse
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.List", req, &resp2); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp2)
	}
	if resp2.Index != 4 {
		t.Fatalf("Bad index: %d %d", resp2.Index, 4)
	}
	if len(resp2.Allocations) != 1 || resp.Allocations[0].ID != alloc.ID ||
		resp2.Allocations[0].ClientStatus != structs.AllocClientStatusRunning {
		t.Fatalf("bad: %#v", resp2.Allocations)
	}
}

func TestAllocEndpoint_GetAlloc(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	alloc := mock.Alloc()
	state := s1.fsm.State()
	state.UpsertJobSummary(999, mock.JobSummary(alloc.JobID))
	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Lookup the jobs
	get := &structs.AllocSpecificRequest{
		AllocID:      alloc.ID,
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	var resp structs.SingleAllocResponse
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.GetAlloc", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}

	if !reflect.DeepEqual(alloc, resp.Alloc) {
		t.Fatalf("bad: %#v", resp.Alloc)
	}
}

func TestAllocEndpoint_GetAlloc_Blocking(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	state := s1.fsm.State()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the allocs
	alloc1 := mock.Alloc()
	alloc2 := mock.Alloc()

	// First create an unrelated alloc
	time.AfterFunc(100*time.Millisecond, func() {
		state.UpsertJobSummary(99, mock.JobSummary(alloc1.JobID))
		err := state.UpsertAllocs(100, []*structs.Allocation{alloc1})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	// Create the alloc we are watching later
	time.AfterFunc(200*time.Millisecond, func() {
		state.UpsertJobSummary(999, mock.JobSummary(alloc2.JobID))
		err := state.UpsertAllocs(200, []*structs.Allocation{alloc2})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	})

	// Lookup the allocs
	get := &structs.AllocSpecificRequest{
		AllocID: alloc2.ID,
		QueryOptions: structs.QueryOptions{
			Region:        "global",
			MinQueryIndex: 50,
		},
	}
	var resp structs.SingleAllocResponse
	start := time.Now()
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.GetAlloc", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}

	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("should block (returned in %s) %#v", elapsed, resp)
	}
	if resp.Index != 200 {
		t.Fatalf("Bad index: %d %d", resp.Index, 200)
	}
	if resp.Alloc == nil || resp.Alloc.ID != alloc2.ID {
		t.Fatalf("bad: %#v", resp.Alloc)
	}
}

func TestAllocEndpoint_GetAllocs(t *testing.T) {
	s1 := testServer(t, nil)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	testutil.WaitForLeader(t, s1.RPC)

	// Create the register request
	alloc := mock.Alloc()
	alloc2 := mock.Alloc()
	state := s1.fsm.State()
	state.UpsertJobSummary(998, mock.JobSummary(alloc.JobID))
	state.UpsertJobSummary(999, mock.JobSummary(alloc2.JobID))
	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc, alloc2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Lookup the allocs
	get := &structs.AllocsGetRequest{
		AllocIDs:     []string{alloc.ID, alloc2.ID},
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	var resp structs.AllocsGetResponse
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.GetAllocs", get, &resp); err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp.Index != 1000 {
		t.Fatalf("Bad index: %d %d", resp.Index, 1000)
	}

	if len(resp.Allocs) != 2 {
		t.Fatalf("bad: %#v", resp.Allocs)
	}

	// Lookup non-existent allocs.
	get = &structs.AllocsGetRequest{
		AllocIDs:     []string{"foo"},
		QueryOptions: structs.QueryOptions{Region: "global"},
	}
	if err := msgpackrpc.CallWithCodec(codec, "Alloc.GetAllocs", get, &resp); err == nil {
		t.Fatalf("expect error")
	}
}
