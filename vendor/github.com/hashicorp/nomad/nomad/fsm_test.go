package nomad

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
	"github.com/hashicorp/raft"
)

type MockSink struct {
	*bytes.Buffer
	cancel bool
}

func (m *MockSink) ID() string {
	return "Mock"
}

func (m *MockSink) Cancel() error {
	m.cancel = true
	return nil
}

func (m *MockSink) Close() error {
	return nil
}

func testStateStore(t *testing.T) *state.StateStore {
	state, err := state.NewStateStore(os.Stderr)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if state == nil {
		t.Fatalf("missing state")
	}
	return state
}

func testFSM(t *testing.T) *nomadFSM {
	p, _ := testPeriodicDispatcher()
	broker := testBroker(t, 0)
	blocked := NewBlockedEvals(broker)
	fsm, err := NewFSM(broker, p, blocked, os.Stderr)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if fsm == nil {
		t.Fatalf("missing fsm")
	}
	return fsm
}

func makeLog(buf []byte) *raft.Log {
	return &raft.Log{
		Index: 1,
		Term:  1,
		Type:  raft.LogCommand,
		Data:  buf,
	}
}

func TestFSM_UpsertNode(t *testing.T) {
	fsm := testFSM(t)
	fsm.blockedEvals.SetEnabled(true)

	node := mock.Node()

	// Mark an eval as blocked.
	eval := mock.Eval()
	eval.ClassEligibility = map[string]bool{node.ComputedClass: true}
	fsm.blockedEvals.Block(eval)

	req := structs.NodeRegisterRequest{
		Node: node,
	}
	buf, err := structs.Encode(structs.NodeRegisterRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	n, err := fsm.State().NodeByID(req.Node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n == nil {
		t.Fatalf("not found!")
	}
	if n.CreateIndex != 1 {
		t.Fatalf("bad index: %d", node.CreateIndex)
	}

	tt := fsm.TimeTable()
	index := tt.NearestIndex(time.Now().UTC())
	if index != 1 {
		t.Fatalf("bad: %d", index)
	}

	// Verify the eval was unblocked.
	testutil.WaitForResult(func() (bool, error) {
		bStats := fsm.blockedEvals.Stats()
		if bStats.TotalBlocked != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})

}

func TestFSM_DeregisterNode(t *testing.T) {
	fsm := testFSM(t)

	node := mock.Node()
	req := structs.NodeRegisterRequest{
		Node: node,
	}
	buf, err := structs.Encode(structs.NodeRegisterRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	req2 := structs.NodeDeregisterRequest{
		NodeID: node.ID,
	}
	buf, err = structs.Encode(structs.NodeDeregisterRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are NOT registered
	node, err = fsm.State().NodeByID(req.Node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if node != nil {
		t.Fatalf("node found!")
	}
}

func TestFSM_UpdateNodeStatus(t *testing.T) {
	fsm := testFSM(t)
	fsm.blockedEvals.SetEnabled(true)

	node := mock.Node()
	req := structs.NodeRegisterRequest{
		Node: node,
	}
	buf, err := structs.Encode(structs.NodeRegisterRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Mark an eval as blocked.
	eval := mock.Eval()
	eval.ClassEligibility = map[string]bool{node.ComputedClass: true}
	fsm.blockedEvals.Block(eval)

	req2 := structs.NodeUpdateStatusRequest{
		NodeID: node.ID,
		Status: structs.NodeStatusReady,
	}
	buf, err = structs.Encode(structs.NodeUpdateStatusRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify the status is ready.
	node, err = fsm.State().NodeByID(req.Node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if node.Status != structs.NodeStatusReady {
		t.Fatalf("bad node: %#v", node)
	}

	// Verify the eval was unblocked.
	testutil.WaitForResult(func() (bool, error) {
		bStats := fsm.blockedEvals.Stats()
		if bStats.TotalBlocked != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestFSM_UpdateNodeDrain(t *testing.T) {
	fsm := testFSM(t)

	node := mock.Node()
	req := structs.NodeRegisterRequest{
		Node: node,
	}
	buf, err := structs.Encode(structs.NodeRegisterRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	req2 := structs.NodeUpdateDrainRequest{
		NodeID: node.ID,
		Drain:  true,
	}
	buf, err = structs.Encode(structs.NodeUpdateDrainRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are NOT registered
	node, err = fsm.State().NodeByID(req.Node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !node.Drain {
		t.Fatalf("bad node: %#v", node)
	}
}

func TestFSM_RegisterJob(t *testing.T) {
	fsm := testFSM(t)

	job := mock.PeriodicJob()
	req := structs.JobRegisterRequest{
		Job: job,
	}
	buf, err := structs.Encode(structs.JobRegisterRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	jobOut, err := fsm.State().JobByID(req.Job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if jobOut == nil {
		t.Fatalf("not found!")
	}
	if jobOut.CreateIndex != 1 {
		t.Fatalf("bad index: %d", jobOut.CreateIndex)
	}

	// Verify it was added to the periodic runner.
	if _, ok := fsm.periodicDispatcher.tracked[job.ID]; !ok {
		t.Fatal("job not added to periodic runner")
	}

	// Verify the launch time was tracked.
	launchOut, err := fsm.State().PeriodicLaunchByID(req.Job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if launchOut == nil {
		t.Fatalf("not found!")
	}
	if launchOut.Launch.IsZero() {
		t.Fatalf("bad launch time: %v", launchOut.Launch)
	}
}

func TestFSM_DeregisterJob(t *testing.T) {
	fsm := testFSM(t)

	job := mock.PeriodicJob()
	req := structs.JobRegisterRequest{
		Job: job,
	}
	buf, err := structs.Encode(structs.JobRegisterRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	req2 := structs.JobDeregisterRequest{
		JobID: job.ID,
	}
	buf, err = structs.Encode(structs.JobDeregisterRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are NOT registered
	jobOut, err := fsm.State().JobByID(req.Job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if jobOut != nil {
		t.Fatalf("job found!")
	}

	// Verify it was removed from the periodic runner.
	if _, ok := fsm.periodicDispatcher.tracked[job.ID]; ok {
		t.Fatal("job not removed from periodic runner")
	}

	// Verify it was removed from the periodic launch table.
	launchOut, err := fsm.State().PeriodicLaunchByID(req.Job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if launchOut != nil {
		t.Fatalf("launch found!")
	}
}

func TestFSM_UpdateEval(t *testing.T) {
	fsm := testFSM(t)
	fsm.evalBroker.SetEnabled(true)

	req := structs.EvalUpdateRequest{
		Evals: []*structs.Evaluation{mock.Eval()},
	}
	buf, err := structs.Encode(structs.EvalUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	eval, err := fsm.State().EvalByID(req.Evals[0].ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if eval == nil {
		t.Fatalf("not found!")
	}
	if eval.CreateIndex != 1 {
		t.Fatalf("bad index: %d", eval.CreateIndex)
	}

	// Verify enqueued
	stats := fsm.evalBroker.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v %#v", stats, eval)
	}
}

func TestFSM_UpdateEval_Blocked(t *testing.T) {
	fsm := testFSM(t)
	fsm.evalBroker.SetEnabled(true)
	fsm.blockedEvals.SetEnabled(true)

	// Create a blocked eval.
	eval := mock.Eval()
	eval.Status = structs.EvalStatusBlocked

	req := structs.EvalUpdateRequest{
		Evals: []*structs.Evaluation{eval},
	}
	buf, err := structs.Encode(structs.EvalUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	out, err := fsm.State().EvalByID(eval.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out == nil {
		t.Fatalf("not found!")
	}
	if out.CreateIndex != 1 {
		t.Fatalf("bad index: %d", out.CreateIndex)
	}

	// Verify the eval wasn't enqueued
	stats := fsm.evalBroker.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v %#v", stats, out)
	}

	// Verify the eval was added to the blocked tracker.
	bStats := fsm.blockedEvals.Stats()
	if bStats.TotalBlocked != 1 {
		t.Fatalf("bad: %#v %#v", bStats, out)
	}
}

func TestFSM_DeleteEval(t *testing.T) {
	fsm := testFSM(t)

	eval := mock.Eval()
	req := structs.EvalUpdateRequest{
		Evals: []*structs.Evaluation{eval},
	}
	buf, err := structs.Encode(structs.EvalUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	req2 := structs.EvalDeleteRequest{
		Evals: []string{eval.ID},
	}
	buf, err = structs.Encode(structs.EvalDeleteRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are NOT registered
	eval, err = fsm.State().EvalByID(req.Evals[0].ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if eval != nil {
		t.Fatalf("eval found!")
	}
}

func TestFSM_UpsertAllocs(t *testing.T) {
	fsm := testFSM(t)

	alloc := mock.Alloc()
	fsm.State().UpsertJobSummary(1, mock.JobSummary(alloc.JobID))
	req := structs.AllocUpdateRequest{
		Alloc: []*structs.Allocation{alloc},
	}
	buf, err := structs.Encode(structs.AllocUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	out, err := fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	alloc.CreateIndex = out.CreateIndex
	alloc.ModifyIndex = out.ModifyIndex
	alloc.AllocModifyIndex = out.AllocModifyIndex
	if !reflect.DeepEqual(alloc, out) {
		t.Fatalf("bad: %#v %#v", alloc, out)
	}

	evictAlloc := new(structs.Allocation)
	*evictAlloc = *alloc
	evictAlloc.DesiredStatus = structs.AllocDesiredStatusEvict
	req2 := structs.AllocUpdateRequest{
		Alloc: []*structs.Allocation{evictAlloc},
	}
	buf, err = structs.Encode(structs.AllocUpdateRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are evicted
	out, err = fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.DesiredStatus != structs.AllocDesiredStatusEvict {
		t.Fatalf("alloc found!")
	}
}

func TestFSM_UpsertAllocs_SharedJob(t *testing.T) {
	fsm := testFSM(t)

	alloc := mock.Alloc()
	fsm.State().UpsertJobSummary(1, mock.JobSummary(alloc.JobID))
	job := alloc.Job
	alloc.Job = nil
	req := structs.AllocUpdateRequest{
		Job:   job,
		Alloc: []*structs.Allocation{alloc},
	}
	buf, err := structs.Encode(structs.AllocUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	out, err := fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	alloc.CreateIndex = out.CreateIndex
	alloc.ModifyIndex = out.ModifyIndex
	alloc.AllocModifyIndex = out.AllocModifyIndex

	// Job should be re-attached
	alloc.Job = job
	if !reflect.DeepEqual(alloc, out) {
		t.Fatalf("bad: %#v %#v", alloc, out)
	}

	// Ensure that the original job is used
	evictAlloc := new(structs.Allocation)
	*evictAlloc = *alloc
	job = mock.Job()
	job.Priority = 123

	evictAlloc.Job = nil
	evictAlloc.DesiredStatus = structs.AllocDesiredStatusEvict
	req2 := structs.AllocUpdateRequest{
		Job:   job,
		Alloc: []*structs.Allocation{evictAlloc},
	}
	buf, err = structs.Encode(structs.AllocUpdateRequestType, req2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp = fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are evicted
	out, err = fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.DesiredStatus != structs.AllocDesiredStatusEvict {
		t.Fatalf("alloc found!")
	}
	if out.Job == nil || out.Job.Priority == 123 {
		t.Fatalf("bad job")
	}
}

func TestFSM_UpsertAllocs_StrippedResources(t *testing.T) {
	fsm := testFSM(t)

	alloc := mock.Alloc()
	fsm.State().UpsertJobSummary(1, mock.JobSummary(alloc.JobID))
	job := alloc.Job
	resources := alloc.Resources
	alloc.Resources = nil
	req := structs.AllocUpdateRequest{
		Job:   job,
		Alloc: []*structs.Allocation{alloc},
	}
	buf, err := structs.Encode(structs.AllocUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	out, err := fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	alloc.CreateIndex = out.CreateIndex
	alloc.ModifyIndex = out.ModifyIndex
	alloc.AllocModifyIndex = out.AllocModifyIndex

	// Resources should be recomputed
	alloc.Resources = resources
	if !reflect.DeepEqual(alloc, out) {
		t.Fatalf("bad: %#v %#v", alloc, out)
	}
}

func TestFSM_UpdateAllocFromClient_Unblock(t *testing.T) {
	fsm := testFSM(t)
	fsm.blockedEvals.SetEnabled(true)
	state := fsm.State()

	node := mock.Node()
	state.UpsertNode(1, node)

	// Mark an eval as blocked.
	eval := mock.Eval()
	eval.ClassEligibility = map[string]bool{node.ComputedClass: true}
	fsm.blockedEvals.Block(eval)

	bStats := fsm.blockedEvals.Stats()
	if bStats.TotalBlocked != 1 {
		t.Fatalf("bad: %#v", bStats)
	}

	// Create a completed eval
	alloc := mock.Alloc()
	alloc.NodeID = node.ID
	alloc2 := mock.Alloc()
	alloc2.NodeID = node.ID
	state.UpsertJobSummary(8, mock.JobSummary(alloc.JobID))
	state.UpsertJobSummary(9, mock.JobSummary(alloc2.JobID))
	state.UpsertAllocs(10, []*structs.Allocation{alloc, alloc2})

	clientAlloc := new(structs.Allocation)
	*clientAlloc = *alloc
	clientAlloc.ClientStatus = structs.AllocClientStatusComplete
	update2 := &structs.Allocation{
		ID:           alloc2.ID,
		ClientStatus: structs.AllocClientStatusRunning,
	}

	req := structs.AllocUpdateRequest{
		Alloc: []*structs.Allocation{clientAlloc, update2},
	}
	buf, err := structs.Encode(structs.AllocClientUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are updated
	out, err := fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	clientAlloc.CreateIndex = out.CreateIndex
	clientAlloc.ModifyIndex = out.ModifyIndex
	if !reflect.DeepEqual(clientAlloc, out) {
		t.Fatalf("bad: %#v %#v", clientAlloc, out)
	}

	out, err = fsm.State().AllocByID(alloc2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	alloc2.CreateIndex = out.CreateIndex
	alloc2.ModifyIndex = out.ModifyIndex
	alloc2.ClientStatus = structs.AllocClientStatusRunning
	alloc2.TaskStates = nil
	if !reflect.DeepEqual(alloc2, out) {
		t.Fatalf("bad: %#v %#v", alloc2, out)
	}

	// Verify the eval was unblocked.
	testutil.WaitForResult(func() (bool, error) {
		bStats = fsm.blockedEvals.Stats()
		if bStats.TotalBlocked != 0 {
			return false, fmt.Errorf("bad: %#v %#v", bStats, out)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestFSM_UpdateAllocFromClient(t *testing.T) {
	fsm := testFSM(t)
	state := fsm.State()

	alloc := mock.Alloc()
	state.UpsertJobSummary(9, mock.JobSummary(alloc.JobID))
	state.UpsertAllocs(10, []*structs.Allocation{alloc})

	clientAlloc := new(structs.Allocation)
	*clientAlloc = *alloc
	clientAlloc.ClientStatus = structs.AllocClientStatusFailed

	req := structs.AllocUpdateRequest{
		Alloc: []*structs.Allocation{clientAlloc},
	}
	buf, err := structs.Encode(structs.AllocClientUpdateRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	// Verify we are registered
	out, err := fsm.State().AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	clientAlloc.CreateIndex = out.CreateIndex
	clientAlloc.ModifyIndex = out.ModifyIndex
	if !reflect.DeepEqual(clientAlloc, out) {
		t.Fatalf("err: %#v,%#v", clientAlloc, out)
	}
}

func testSnapshotRestore(t *testing.T, fsm *nomadFSM) *nomadFSM {
	// Snapshot
	snap, err := fsm.Snapshot()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer snap.Release()

	// Persist
	buf := bytes.NewBuffer(nil)
	sink := &MockSink{buf, false}
	if err := snap.Persist(sink); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Try to restore on a new FSM
	fsm2 := testFSM(t)

	// Do a restore
	if err := fsm2.Restore(sink); err != nil {
		t.Fatalf("err: %v", err)
	}
	return fsm2
}

func TestFSM_SnapshotRestore_Nodes(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()
	node1 := mock.Node()
	state.UpsertNode(1000, node1)
	node2 := mock.Node()
	state.UpsertNode(1001, node2)

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	out1, _ := state2.NodeByID(node1.ID)
	out2, _ := state2.NodeByID(node2.ID)
	if !reflect.DeepEqual(node1, out1) {
		t.Fatalf("bad: \n%#v\n%#v", out1, node1)
	}
	if !reflect.DeepEqual(node2, out2) {
		t.Fatalf("bad: \n%#v\n%#v", out2, node2)
	}
}

func TestFSM_SnapshotRestore_Jobs(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()
	job1 := mock.Job()
	state.UpsertJob(1000, job1)
	job2 := mock.Job()
	state.UpsertJob(1001, job2)

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	out1, _ := state2.JobByID(job1.ID)
	out2, _ := state2.JobByID(job2.ID)
	if !reflect.DeepEqual(job1, out1) {
		t.Fatalf("bad: \n%#v\n%#v", out1, job1)
	}
	if !reflect.DeepEqual(job2, out2) {
		t.Fatalf("bad: \n%#v\n%#v", out2, job2)
	}
}

func TestFSM_SnapshotRestore_Evals(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()
	eval1 := mock.Eval()
	state.UpsertEvals(1000, []*structs.Evaluation{eval1})
	eval2 := mock.Eval()
	state.UpsertEvals(1001, []*structs.Evaluation{eval2})

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	out1, _ := state2.EvalByID(eval1.ID)
	out2, _ := state2.EvalByID(eval2.ID)
	if !reflect.DeepEqual(eval1, out1) {
		t.Fatalf("bad: \n%#v\n%#v", out1, eval1)
	}
	if !reflect.DeepEqual(eval2, out2) {
		t.Fatalf("bad: \n%#v\n%#v", out2, eval2)
	}
}

func TestFSM_SnapshotRestore_Allocs(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()
	alloc1 := mock.Alloc()
	alloc2 := mock.Alloc()
	state.UpsertJobSummary(998, mock.JobSummary(alloc1.JobID))
	state.UpsertJobSummary(999, mock.JobSummary(alloc2.JobID))
	state.UpsertAllocs(1000, []*structs.Allocation{alloc1})
	state.UpsertAllocs(1001, []*structs.Allocation{alloc2})

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	out1, _ := state2.AllocByID(alloc1.ID)
	out2, _ := state2.AllocByID(alloc2.ID)
	if !reflect.DeepEqual(alloc1, out1) {
		t.Fatalf("bad: \n%#v\n%#v", out1, alloc1)
	}
	if !reflect.DeepEqual(alloc2, out2) {
		t.Fatalf("bad: \n%#v\n%#v", out2, alloc2)
	}
}

func TestFSM_SnapshotRestore_Indexes(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()
	node1 := mock.Node()
	state.UpsertNode(1000, node1)

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()

	index, err := state2.Index("nodes")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}
}

func TestFSM_SnapshotRestore_TimeTable(t *testing.T) {
	// Add some state
	fsm := testFSM(t)

	tt := fsm.TimeTable()
	start := time.Now().UTC()
	tt.Witness(1000, start)
	tt.Witness(2000, start.Add(10*time.Minute))

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)

	tt2 := fsm2.TimeTable()
	if tt2.NearestTime(1500) != start {
		t.Fatalf("bad")
	}
	if tt2.NearestIndex(start.Add(15*time.Minute)) != 2000 {
		t.Fatalf("bad")
	}
}

func TestFSM_SnapshotRestore_PeriodicLaunches(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()
	job1 := mock.Job()
	launch1 := &structs.PeriodicLaunch{ID: job1.ID, Launch: time.Now()}
	state.UpsertPeriodicLaunch(1000, launch1)
	job2 := mock.Job()
	launch2 := &structs.PeriodicLaunch{ID: job2.ID, Launch: time.Now()}
	state.UpsertPeriodicLaunch(1001, launch2)

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	out1, _ := state2.PeriodicLaunchByID(launch1.ID)
	out2, _ := state2.PeriodicLaunchByID(launch2.ID)
	if !reflect.DeepEqual(launch1, out1) {
		t.Fatalf("bad: \n%#v\n%#v", out1, job1)
	}
	if !reflect.DeepEqual(launch2, out2) {
		t.Fatalf("bad: \n%#v\n%#v", out2, job2)
	}
}

func TestFSM_SnapshotRestore_JobSummary(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()

	job1 := mock.Job()
	state.UpsertJob(1000, job1)
	js1, _ := state.JobSummaryByID(job1.ID)

	job2 := mock.Job()
	state.UpsertJob(1001, job2)
	js2, _ := state.JobSummaryByID(job2.ID)

	// Verify the contents
	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	out1, _ := state2.JobSummaryByID(job1.ID)
	out2, _ := state2.JobSummaryByID(job2.ID)
	if !reflect.DeepEqual(js1, out1) {
		t.Fatalf("bad: \n%#v\n%#v", js1, out1)
	}
	if !reflect.DeepEqual(js2, out2) {
		t.Fatalf("bad: \n%#v\n%#v", js2, out2)
	}
}

func TestFSM_SnapshotRestore_AddMissingSummary(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()

	// make an allocation
	alloc := mock.Alloc()
	state.UpsertJob(1010, alloc.Job)
	state.UpsertAllocs(1011, []*structs.Allocation{alloc})

	// Delete the summary
	state.DeleteJobSummary(1040, alloc.Job.ID)

	// Delete the index
	if err := state.RemoveIndex("job_summary"); err != nil {
		t.Fatalf("err: %v", err)
	}

	fsm2 := testSnapshotRestore(t, fsm)
	state2 := fsm2.State()
	latestIndex, _ := state.LatestIndex()

	out, _ := state2.JobSummaryByID(alloc.Job.ID)
	expected := structs.JobSummary{
		JobID: alloc.Job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Starting: 1,
			},
		},
		CreateIndex: 1010,
		ModifyIndex: latestIndex,
	}
	if !reflect.DeepEqual(&expected, out) {
		t.Fatalf("expected: %#v, actual: %#v", &expected, out)
	}
}

func TestFSM_ReconcileSummaries(t *testing.T) {
	// Add some state
	fsm := testFSM(t)
	state := fsm.State()

	// Add a node
	node := mock.Node()
	state.UpsertNode(800, node)

	// Make a job so that none of the tasks can be placed
	job1 := mock.Job()
	job1.TaskGroups[0].Tasks[0].Resources.CPU = 5000
	state.UpsertJob(1000, job1)

	// make a job which can make partial progress
	alloc := mock.Alloc()
	alloc.NodeID = node.ID
	state.UpsertJob(1010, alloc.Job)
	state.UpsertAllocs(1011, []*structs.Allocation{alloc})

	// Delete the summaries
	state.DeleteJobSummary(1030, job1.ID)
	state.DeleteJobSummary(1040, alloc.Job.ID)

	req := structs.GenericRequest{}
	buf, err := structs.Encode(structs.ReconcileJobSummariesRequestType, req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp := fsm.Apply(makeLog(buf))
	if resp != nil {
		t.Fatalf("resp: %v", resp)
	}

	out1, _ := state.JobSummaryByID(job1.ID)
	expected := structs.JobSummary{
		JobID: job1.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Queued: 10,
			},
		},
		CreateIndex: 1000,
		ModifyIndex: out1.ModifyIndex,
	}
	if !reflect.DeepEqual(&expected, out1) {
		t.Fatalf("expected: %#v, actual: %#v", &expected, out1)
	}

	// This exercises the code path which adds the allocations made by the
	// planner and the number of unplaced allocations in the reconcile summaries
	// codepath
	out2, _ := state.JobSummaryByID(alloc.Job.ID)
	expected = structs.JobSummary{
		JobID: alloc.Job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Queued:   10,
				Starting: 1,
			},
		},
		CreateIndex: 1010,
		ModifyIndex: out2.ModifyIndex,
	}
	if !reflect.DeepEqual(&expected, out2) {
		t.Fatalf("expected: %#v, actual: %#v", &expected, out2)
	}
}
