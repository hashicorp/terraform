package state

import (
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/watch"
)

func testStateStore(t *testing.T) *StateStore {
	state, err := NewStateStore(os.Stderr)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if state == nil {
		t.Fatalf("missing state")
	}
	return state
}

func TestStateStore_UpsertNode_Node(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "nodes"},
		watch.Item{Node: node.ID})

	err := state.UpsertNode(1000, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.NodeByID(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(node, out) {
		t.Fatalf("bad: %#v %#v", node, out)
	}

	index, err := state.Index("nodes")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_DeleteNode_Node(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "nodes"},
		watch.Item{Node: node.ID})

	err := state.UpsertNode(1000, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.DeleteNode(1001, node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.NodeByID(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", node, out)
	}

	index, err := state.Index("nodes")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_UpdateNodeStatus_Node(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "nodes"},
		watch.Item{Node: node.ID})

	err := state.UpsertNode(800, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.UpdateNodeStatus(801, node.ID, structs.NodeStatusReady)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.NodeByID(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out.Status != structs.NodeStatusReady {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 801 {
		t.Fatalf("bad: %#v", out)
	}

	index, err := state.Index("nodes")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 801 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_UpdateNodeDrain_Node(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "nodes"},
		watch.Item{Node: node.ID})

	err := state.UpsertNode(1000, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.UpdateNodeDrain(1001, node.ID, true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.NodeByID(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !out.Drain {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 1001 {
		t.Fatalf("bad: %#v", out)
	}

	index, err := state.Index("nodes")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_Nodes(t *testing.T) {
	state := testStateStore(t)
	var nodes []*structs.Node

	for i := 0; i < 10; i++ {
		node := mock.Node()
		nodes = append(nodes, node)

		err := state.UpsertNode(1000+uint64(i), node)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.Nodes()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out []*structs.Node
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Node))
	}

	sort.Sort(NodeIDSort(nodes))
	sort.Sort(NodeIDSort(out))

	if !reflect.DeepEqual(nodes, out) {
		t.Fatalf("bad: %#v %#v", nodes, out)
	}
}

func TestStateStore_NodesByIDPrefix(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	node.ID = "11111111-662e-d0ab-d1c9-3e434af7bdb4"
	err := state.UpsertNode(1000, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err := state.NodesByIDPrefix(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	gatherNodes := func(iter memdb.ResultIterator) []*structs.Node {
		var nodes []*structs.Node
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			node := raw.(*structs.Node)
			nodes = append(nodes, node)
		}
		return nodes
	}

	nodes := gatherNodes(iter)
	if len(nodes) != 1 {
		t.Fatalf("err: %v", err)
	}

	iter, err = state.NodesByIDPrefix("11")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	nodes = gatherNodes(iter)
	if len(nodes) != 1 {
		t.Fatalf("err: %v", err)
	}

	node = mock.Node()
	node.ID = "11222222-662e-d0ab-d1c9-3e434af7bdb4"
	err = state.UpsertNode(1001, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err = state.NodesByIDPrefix("11")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	nodes = gatherNodes(iter)
	if len(nodes) != 2 {
		t.Fatalf("err: %v", err)
	}

	iter, err = state.NodesByIDPrefix("1111")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	nodes = gatherNodes(iter)
	if len(nodes) != 1 {
		t.Fatalf("err: %v", err)
	}
}

func TestStateStore_RestoreNode(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "nodes"},
		watch.Item{Node: node.ID})

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.NodeRestore(node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	restore.Commit()

	out, err := state.NodeByID(node.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(out, node) {
		t.Fatalf("Bad: %#v %#v", out, node)
	}

	notify.verify(t)
}

func TestStateStore_UpsertJob_Job(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "jobs"},
		watch.Item{Job: job.ID})

	err := state.UpsertJob(1000, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.JobByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(job, out) {
		t.Fatalf("bad: %#v %#v", job, out)
	}

	index, err := state.Index("jobs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}

	summary, err := state.JobSummaryByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if summary == nil {
		t.Fatalf("nil summary")
	}
	if summary.JobID != job.ID {
		t.Fatalf("bad summary id: %v", summary.JobID)
	}
	_, ok := summary.Summary["web"]
	if !ok {
		t.Fatalf("nil summary for task group")
	}
	notify.verify(t)
}

func TestStateStore_UpdateUpsertJob_Job(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "jobs"},
		watch.Item{Job: job.ID})

	err := state.UpsertJob(1000, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	job2 := mock.Job()
	job2.ID = job.ID
	err = state.UpsertJob(1001, job2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.JobByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(job2, out) {
		t.Fatalf("bad: %#v %#v", job2, out)
	}

	if out.CreateIndex != 1000 {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 1001 {
		t.Fatalf("bad: %#v", out)
	}

	index, err := state.Index("jobs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	// Test that the job summary remains the same if the job is updated but
	// count remains same
	summary, err := state.JobSummaryByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if summary == nil {
		t.Fatalf("nil summary")
	}
	if summary.JobID != job.ID {
		t.Fatalf("bad summary id: %v", summary.JobID)
	}
	_, ok := summary.Summary["web"]
	if !ok {
		t.Fatalf("nil summary for task group")
	}

	notify.verify(t)
}

func TestStateStore_DeleteJob_Job(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "jobs"},
		watch.Item{Job: job.ID})

	err := state.UpsertJob(1000, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.DeleteJob(1001, job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.JobByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", job, out)
	}

	index, err := state.Index("jobs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	summary, err := state.JobSummaryByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if summary != nil {
		t.Fatalf("expected summary to be nil, but got: %v", summary)
	}

	notify.verify(t)
}

func TestStateStore_Jobs(t *testing.T) {
	state := testStateStore(t)
	var jobs []*structs.Job

	for i := 0; i < 10; i++ {
		job := mock.Job()
		jobs = append(jobs, job)

		err := state.UpsertJob(1000+uint64(i), job)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.Jobs()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out []*structs.Job
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Job))
	}

	sort.Sort(JobIDSort(jobs))
	sort.Sort(JobIDSort(out))

	if !reflect.DeepEqual(jobs, out) {
		t.Fatalf("bad: %#v %#v", jobs, out)
	}
}

func TestStateStore_JobsByIDPrefix(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	job.ID = "redis"
	err := state.UpsertJob(1000, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err := state.JobsByIDPrefix(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	gatherJobs := func(iter memdb.ResultIterator) []*structs.Job {
		var jobs []*structs.Job
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			jobs = append(jobs, raw.(*structs.Job))
		}
		return jobs
	}

	jobs := gatherJobs(iter)
	if len(jobs) != 1 {
		t.Fatalf("err: %v", err)
	}

	iter, err = state.JobsByIDPrefix("re")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	jobs = gatherJobs(iter)
	if len(jobs) != 1 {
		t.Fatalf("err: %v", err)
	}

	job = mock.Job()
	job.ID = "riak"
	err = state.UpsertJob(1001, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err = state.JobsByIDPrefix("r")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	jobs = gatherJobs(iter)
	if len(jobs) != 2 {
		t.Fatalf("err: %v", err)
	}

	iter, err = state.JobsByIDPrefix("ri")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	jobs = gatherJobs(iter)
	if len(jobs) != 1 {
		t.Fatalf("err: %v", err)
	}
}

func TestStateStore_JobsByPeriodic(t *testing.T) {
	state := testStateStore(t)
	var periodic, nonPeriodic []*structs.Job

	for i := 0; i < 10; i++ {
		job := mock.Job()
		nonPeriodic = append(nonPeriodic, job)

		err := state.UpsertJob(1000+uint64(i), job)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		job := mock.PeriodicJob()
		periodic = append(periodic, job)

		err := state.UpsertJob(2000+uint64(i), job)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.JobsByPeriodic(true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var outPeriodic []*structs.Job
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		outPeriodic = append(outPeriodic, raw.(*structs.Job))
	}

	iter, err = state.JobsByPeriodic(false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var outNonPeriodic []*structs.Job
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		outNonPeriodic = append(outNonPeriodic, raw.(*structs.Job))
	}

	sort.Sort(JobIDSort(periodic))
	sort.Sort(JobIDSort(nonPeriodic))
	sort.Sort(JobIDSort(outPeriodic))
	sort.Sort(JobIDSort(outNonPeriodic))

	if !reflect.DeepEqual(periodic, outPeriodic) {
		t.Fatalf("bad: %#v %#v", periodic, outPeriodic)
	}

	if !reflect.DeepEqual(nonPeriodic, outNonPeriodic) {
		t.Fatalf("bad: %#v %#v", nonPeriodic, outNonPeriodic)
	}
}

func TestStateStore_JobsByScheduler(t *testing.T) {
	state := testStateStore(t)
	var serviceJobs []*structs.Job
	var sysJobs []*structs.Job

	for i := 0; i < 10; i++ {
		job := mock.Job()
		serviceJobs = append(serviceJobs, job)

		err := state.UpsertJob(1000+uint64(i), job)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		job := mock.SystemJob()
		sysJobs = append(sysJobs, job)

		err := state.UpsertJob(2000+uint64(i), job)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.JobsByScheduler("service")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var outService []*structs.Job
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		outService = append(outService, raw.(*structs.Job))
	}

	iter, err = state.JobsByScheduler("system")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var outSystem []*structs.Job
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		outSystem = append(outSystem, raw.(*structs.Job))
	}

	sort.Sort(JobIDSort(serviceJobs))
	sort.Sort(JobIDSort(sysJobs))
	sort.Sort(JobIDSort(outService))
	sort.Sort(JobIDSort(outSystem))

	if !reflect.DeepEqual(serviceJobs, outService) {
		t.Fatalf("bad: %#v %#v", serviceJobs, outService)
	}

	if !reflect.DeepEqual(sysJobs, outSystem) {
		t.Fatalf("bad: %#v %#v", sysJobs, outSystem)
	}
}

func TestStateStore_JobsByGC(t *testing.T) {
	state := testStateStore(t)
	var gc, nonGc []*structs.Job

	for i := 0; i < 20; i++ {
		var job *structs.Job
		if i%2 == 0 {
			job = mock.Job()
		} else {
			job = mock.PeriodicJob()
		}
		nonGc = append(nonGc, job)

		if err := state.UpsertJob(1000+uint64(i), job); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		job := mock.Job()
		job.Type = structs.JobTypeBatch
		gc = append(gc, job)

		if err := state.UpsertJob(2000+uint64(i), job); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.JobsByGC(true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var outGc []*structs.Job
	for i := iter.Next(); i != nil; i = iter.Next() {
		outGc = append(outGc, i.(*structs.Job))
	}

	iter, err = state.JobsByGC(false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var outNonGc []*structs.Job
	for i := iter.Next(); i != nil; i = iter.Next() {
		outNonGc = append(outNonGc, i.(*structs.Job))
	}

	sort.Sort(JobIDSort(gc))
	sort.Sort(JobIDSort(nonGc))
	sort.Sort(JobIDSort(outGc))
	sort.Sort(JobIDSort(outNonGc))

	if !reflect.DeepEqual(gc, outGc) {
		t.Fatalf("bad: %#v %#v", gc, outGc)
	}

	if !reflect.DeepEqual(nonGc, outNonGc) {
		t.Fatalf("bad: %#v %#v", nonGc, outNonGc)
	}
}

func TestStateStore_RestoreJob(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "jobs"},
		watch.Item{Job: job.ID})

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.JobRestore(job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	restore.Commit()

	out, err := state.JobByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(out, job) {
		t.Fatalf("Bad: %#v %#v", out, job)
	}

	notify.verify(t)
}

func TestStateStore_UpsertPeriodicLaunch(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()
	launch := &structs.PeriodicLaunch{ID: job.ID, Launch: time.Now()}

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "periodic_launch"},
		watch.Item{Job: job.ID})

	err := state.UpsertPeriodicLaunch(1000, launch)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.PeriodicLaunchByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.CreateIndex != 1000 {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 1000 {
		t.Fatalf("bad: %#v", out)
	}

	if !reflect.DeepEqual(launch, out) {
		t.Fatalf("bad: %#v %#v", job, out)
	}

	index, err := state.Index("periodic_launch")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_UpdateUpsertPeriodicLaunch(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()
	launch := &structs.PeriodicLaunch{ID: job.ID, Launch: time.Now()}

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "periodic_launch"},
		watch.Item{Job: job.ID})

	err := state.UpsertPeriodicLaunch(1000, launch)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	launch2 := &structs.PeriodicLaunch{
		ID:     job.ID,
		Launch: launch.Launch.Add(1 * time.Second),
	}
	err = state.UpsertPeriodicLaunch(1001, launch2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.PeriodicLaunchByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.CreateIndex != 1000 {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 1001 {
		t.Fatalf("bad: %#v", out)
	}

	if !reflect.DeepEqual(launch2, out) {
		t.Fatalf("bad: %#v %#v", launch2, out)
	}

	index, err := state.Index("periodic_launch")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_DeletePeriodicLaunch(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()
	launch := &structs.PeriodicLaunch{ID: job.ID, Launch: time.Now()}

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "periodic_launch"},
		watch.Item{Job: job.ID})

	err := state.UpsertPeriodicLaunch(1000, launch)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.DeletePeriodicLaunch(1001, job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.PeriodicLaunchByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", job, out)
	}

	index, err := state.Index("periodic_launch")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_PeriodicLaunches(t *testing.T) {
	state := testStateStore(t)
	var launches []*structs.PeriodicLaunch

	for i := 0; i < 10; i++ {
		job := mock.Job()
		launch := &structs.PeriodicLaunch{ID: job.ID, Launch: time.Now()}
		launches = append(launches, launch)

		err := state.UpsertPeriodicLaunch(1000+uint64(i), launch)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.PeriodicLaunches()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out := make(map[string]*structs.PeriodicLaunch, 10)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		launch := raw.(*structs.PeriodicLaunch)
		if _, ok := out[launch.ID]; ok {
			t.Fatalf("duplicate: %v", launch.ID)
		}

		out[launch.ID] = launch
	}

	for _, launch := range launches {
		l, ok := out[launch.ID]
		if !ok {
			t.Fatalf("bad %v", launch.ID)
		}

		if !reflect.DeepEqual(launch, l) {
			t.Fatalf("bad: %#v %#v", launch, l)
		}

		delete(out, launch.ID)
	}

	if len(out) != 0 {
		t.Fatalf("leftover: %#v", out)
	}
}

func TestStateStore_RestorePeriodicLaunch(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()
	launch := &structs.PeriodicLaunch{ID: job.ID, Launch: time.Now()}

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "periodic_launch"},
		watch.Item{Job: job.ID})

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.PeriodicLaunchRestore(launch)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	restore.Commit()

	out, err := state.PeriodicLaunchByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(out, launch) {
		t.Fatalf("Bad: %#v %#v", out, job)
	}

	notify.verify(t)
}

func TestStateStore_RestoreJobSummary(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()
	jobSummary := &structs.JobSummary{
		JobID: job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Starting: 10,
			},
		},
	}
	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.JobSummaryRestore(jobSummary)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	restore.Commit()

	out, err := state.JobSummaryByID(job.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(out, jobSummary) {
		t.Fatalf("Bad: %#v %#v", out, jobSummary)
	}
}

func TestStateStore_Indexes(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()

	err := state.UpsertNode(1000, node)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err := state.Indexes()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out []*IndexEntry
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*IndexEntry))
	}

	expect := []*IndexEntry{
		&IndexEntry{"nodes", 1000},
	}

	if !reflect.DeepEqual(expect, out) {
		t.Fatalf("bad: %#v %#v", expect, out)
	}
}

func TestStateStore_LatestIndex(t *testing.T) {
	state := testStateStore(t)

	if err := state.UpsertNode(1000, mock.Node()); err != nil {
		t.Fatalf("err: %v", err)
	}

	exp := uint64(2000)
	if err := state.UpsertJob(exp, mock.Job()); err != nil {
		t.Fatalf("err: %v", err)
	}

	latest, err := state.LatestIndex()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if latest != exp {
		t.Fatalf("LatestIndex() returned %d; want %d", latest, exp)
	}
}

func TestStateStore_RestoreIndex(t *testing.T) {
	state := testStateStore(t)

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	index := &IndexEntry{"jobs", 1000}
	err = restore.IndexRestore(index)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	restore.Commit()

	out, err := state.Index("jobs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != 1000 {
		t.Fatalf("Bad: %#v %#v", out, 1000)
	}
}

func TestStateStore_UpsertEvals_Eval(t *testing.T) {
	state := testStateStore(t)
	eval := mock.Eval()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "evals"},
		watch.Item{Eval: eval.ID})

	err := state.UpsertEvals(1000, []*structs.Evaluation{eval})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.EvalByID(eval.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(eval, out) {
		t.Fatalf("bad: %#v %#v", eval, out)
	}

	index, err := state.Index("evals")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_Update_UpsertEvals_Eval(t *testing.T) {
	state := testStateStore(t)
	eval := mock.Eval()

	err := state.UpsertEvals(1000, []*structs.Evaluation{eval})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "evals"},
		watch.Item{Eval: eval.ID})

	eval2 := mock.Eval()
	eval2.ID = eval.ID
	err = state.UpsertEvals(1001, []*structs.Evaluation{eval2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.EvalByID(eval.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(eval2, out) {
		t.Fatalf("bad: %#v %#v", eval2, out)
	}

	if out.CreateIndex != 1000 {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 1001 {
		t.Fatalf("bad: %#v", out)
	}

	index, err := state.Index("evals")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_DeleteEval_Eval(t *testing.T) {
	state := testStateStore(t)
	eval1 := mock.Eval()
	eval2 := mock.Eval()
	alloc1 := mock.Alloc()
	alloc2 := mock.Alloc()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "evals"},
		watch.Item{Table: "allocs"},
		watch.Item{Eval: eval1.ID},
		watch.Item{Eval: eval2.ID},
		watch.Item{Alloc: alloc1.ID},
		watch.Item{Alloc: alloc2.ID},
		watch.Item{AllocEval: alloc1.EvalID},
		watch.Item{AllocEval: alloc2.EvalID},
		watch.Item{AllocJob: alloc1.JobID},
		watch.Item{AllocJob: alloc2.JobID},
		watch.Item{AllocNode: alloc1.NodeID},
		watch.Item{AllocNode: alloc2.NodeID})

	state.UpsertJobSummary(900, mock.JobSummary(eval1.JobID))
	state.UpsertJobSummary(901, mock.JobSummary(eval2.JobID))
	state.UpsertJobSummary(902, mock.JobSummary(alloc1.JobID))
	state.UpsertJobSummary(903, mock.JobSummary(alloc2.JobID))
	err := state.UpsertEvals(1000, []*structs.Evaluation{eval1, eval2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.UpsertAllocs(1001, []*structs.Allocation{alloc1, alloc2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = state.DeleteEval(1002, []string{eval1.ID, eval2.ID}, []string{alloc1.ID, alloc2.ID})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.EvalByID(eval1.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", eval1, out)
	}

	out, err = state.EvalByID(eval2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", eval1, out)
	}

	outA, err := state.AllocByID(alloc1.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", alloc1, outA)
	}

	outA, err = state.AllocByID(alloc2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out != nil {
		t.Fatalf("bad: %#v %#v", alloc1, outA)
	}

	index, err := state.Index("evals")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1002 {
		t.Fatalf("bad: %d", index)
	}

	index, err = state.Index("allocs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1002 {
		t.Fatalf("bad: %d", index)
	}

	notify.verify(t)
}

func TestStateStore_EvalsByJob(t *testing.T) {
	state := testStateStore(t)

	eval1 := mock.Eval()
	eval2 := mock.Eval()
	eval2.JobID = eval1.JobID
	eval3 := mock.Eval()
	evals := []*structs.Evaluation{eval1, eval2}

	err := state.UpsertEvals(1000, evals)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	err = state.UpsertEvals(1001, []*structs.Evaluation{eval3})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.EvalsByJob(eval1.JobID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sort.Sort(EvalIDSort(evals))
	sort.Sort(EvalIDSort(out))

	if !reflect.DeepEqual(evals, out) {
		t.Fatalf("bad: %#v %#v", evals, out)
	}
}

func TestStateStore_Evals(t *testing.T) {
	state := testStateStore(t)
	var evals []*structs.Evaluation

	for i := 0; i < 10; i++ {
		eval := mock.Eval()
		evals = append(evals, eval)

		err := state.UpsertEvals(1000+uint64(i), []*structs.Evaluation{eval})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	iter, err := state.Evals()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out []*structs.Evaluation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Evaluation))
	}

	sort.Sort(EvalIDSort(evals))
	sort.Sort(EvalIDSort(out))

	if !reflect.DeepEqual(evals, out) {
		t.Fatalf("bad: %#v %#v", evals, out)
	}
}

func TestStateStore_EvalsByIDPrefix(t *testing.T) {
	state := testStateStore(t)
	var evals []*structs.Evaluation

	ids := []string{
		"aaaaaaaa-7bfb-395d-eb95-0685af2176b2",
		"aaaaaaab-7bfb-395d-eb95-0685af2176b2",
		"aaaaaabb-7bfb-395d-eb95-0685af2176b2",
		"aaaaabbb-7bfb-395d-eb95-0685af2176b2",
		"aaaabbbb-7bfb-395d-eb95-0685af2176b2",
		"aaabbbbb-7bfb-395d-eb95-0685af2176b2",
		"aabbbbbb-7bfb-395d-eb95-0685af2176b2",
		"abbbbbbb-7bfb-395d-eb95-0685af2176b2",
		"bbbbbbbb-7bfb-395d-eb95-0685af2176b2",
	}
	for i := 0; i < 9; i++ {
		eval := mock.Eval()
		eval.ID = ids[i]
		evals = append(evals, eval)
	}

	err := state.UpsertEvals(1000, evals)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err := state.EvalsByIDPrefix("aaaa")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	gatherEvals := func(iter memdb.ResultIterator) []*structs.Evaluation {
		var evals []*structs.Evaluation
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			evals = append(evals, raw.(*structs.Evaluation))
		}
		return evals
	}

	out := gatherEvals(iter)
	if len(out) != 5 {
		t.Fatalf("bad: expected five evaluations, got: %#v", out)
	}

	sort.Sort(EvalIDSort(evals))

	for index, eval := range out {
		if ids[index] != eval.ID {
			t.Fatalf("bad: got unexpected id: %s", eval.ID)
		}
	}

	iter, err = state.EvalsByIDPrefix("b-a7bfb")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out = gatherEvals(iter)
	if len(out) != 0 {
		t.Fatalf("bad: unexpected zero evaluations, got: %#v", out)
	}

}

func TestStateStore_RestoreEval(t *testing.T) {
	state := testStateStore(t)
	eval := mock.Eval()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "evals"},
		watch.Item{Eval: eval.ID})

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.EvalRestore(eval)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	restore.Commit()

	out, err := state.EvalByID(eval.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(out, eval) {
		t.Fatalf("Bad: %#v %#v", out, eval)
	}

	notify.verify(t)
}

func TestStateStore_UpdateAllocsFromClient(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()
	alloc2 := mock.Alloc()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "allocs"},
		watch.Item{Alloc: alloc.ID},
		watch.Item{AllocEval: alloc.EvalID},
		watch.Item{AllocJob: alloc.JobID},
		watch.Item{AllocNode: alloc.NodeID},
		watch.Item{Alloc: alloc2.ID},
		watch.Item{AllocEval: alloc2.EvalID},
		watch.Item{AllocJob: alloc2.JobID},
		watch.Item{AllocNode: alloc2.NodeID})

	if err := state.UpsertJob(999, alloc.Job); err != nil {
		t.Fatalf("err: %v", err)
	}
	if err := state.UpsertJob(999, alloc2.Job); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc, alloc2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create the delta updates
	ts := map[string]*structs.TaskState{"web": &structs.TaskState{State: structs.TaskStatePending}}
	update := &structs.Allocation{
		ID:           alloc.ID,
		ClientStatus: structs.AllocClientStatusFailed,
		TaskStates:   ts,
		JobID:        alloc.JobID,
		TaskGroup:    alloc.TaskGroup,
	}
	update2 := &structs.Allocation{
		ID:           alloc2.ID,
		ClientStatus: structs.AllocClientStatusRunning,
		TaskStates:   ts,
		JobID:        alloc2.JobID,
		TaskGroup:    alloc2.TaskGroup,
	}

	err = state.UpdateAllocsFromClient(1001, []*structs.Allocation{update, update2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	alloc.CreateIndex = 1000
	alloc.ModifyIndex = 1001
	alloc.TaskStates = ts
	alloc.ClientStatus = structs.AllocClientStatusFailed
	if !reflect.DeepEqual(alloc, out) {
		t.Fatalf("bad: %#v %#v", alloc, out)
	}

	out, err = state.AllocByID(alloc2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	alloc2.ModifyIndex = 1000
	alloc2.ModifyIndex = 1001
	alloc2.ClientStatus = structs.AllocClientStatusRunning
	alloc2.TaskStates = ts
	if !reflect.DeepEqual(alloc2, out) {
		t.Fatalf("bad: %#v %#v", alloc2, out)
	}

	index, err := state.Index("allocs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}

	// Ensure summaries have been updated
	summary, err := state.JobSummaryByID(alloc.JobID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	tgSummary := summary.Summary["web"]
	if tgSummary.Failed != 1 {
		t.Fatalf("expected failed: %v, actual: %v, summary: %#v", 1, tgSummary.Failed, tgSummary)
	}

	summary2, err := state.JobSummaryByID(alloc2.JobID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	tgSummary2 := summary2.Summary["web"]
	if tgSummary2.Running != 1 {
		t.Fatalf("expected running: %v, actual: %v", 1, tgSummary2.Running)
	}

	notify.verify(t)
}

func TestStateStore_UpdateMultipleAllocsFromClient(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()

	if err := state.UpsertJob(999, alloc.Job); err != nil {
		t.Fatalf("err: %v", err)
	}
	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create the delta updates
	ts := map[string]*structs.TaskState{"web": &structs.TaskState{State: structs.TaskStatePending}}
	update := &structs.Allocation{
		ID:           alloc.ID,
		ClientStatus: structs.AllocClientStatusRunning,
		TaskStates:   ts,
		JobID:        alloc.JobID,
		TaskGroup:    alloc.TaskGroup,
	}
	update2 := &structs.Allocation{
		ID:           alloc.ID,
		ClientStatus: structs.AllocClientStatusPending,
		TaskStates:   ts,
		JobID:        alloc.JobID,
		TaskGroup:    alloc.TaskGroup,
	}

	err = state.UpdateAllocsFromClient(1001, []*structs.Allocation{update, update2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	alloc.CreateIndex = 1000
	alloc.ModifyIndex = 1001
	alloc.TaskStates = ts
	alloc.ClientStatus = structs.AllocClientStatusPending
	if !reflect.DeepEqual(alloc, out) {
		t.Fatalf("bad: %#v , actual:%#v", alloc, out)
	}

	summary, err := state.JobSummaryByID(alloc.JobID)
	expectedSummary := &structs.JobSummary{
		JobID: alloc.JobID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Starting: 1,
			},
		},
		CreateIndex: 999,
		ModifyIndex: 1001,
	}
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(summary, expectedSummary) {
		t.Fatalf("expected: %#v, actual: %#v", expectedSummary, summary)
	}
}

func TestStateStore_UpsertAlloc_Alloc(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "allocs"},
		watch.Item{Alloc: alloc.ID},
		watch.Item{AllocEval: alloc.EvalID},
		watch.Item{AllocJob: alloc.JobID},
		watch.Item{AllocNode: alloc.NodeID})

	if err := state.UpsertJob(999, alloc.Job); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(alloc, out) {
		t.Fatalf("bad: %#v %#v", alloc, out)
	}

	index, err := state.Index("allocs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1000 {
		t.Fatalf("bad: %d", index)
	}

	summary, err := state.JobSummaryByID(alloc.JobID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	tgSummary, ok := summary.Summary["web"]
	if !ok {
		t.Fatalf("no summary for task group web")
	}
	if tgSummary.Starting != 1 {
		t.Fatalf("expected queued: %v, actual: %v", 1, tgSummary.Starting)
	}

	notify.verify(t)
}

func TestStateStore_UpdateAlloc_Alloc(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()

	if err := state.UpsertJob(999, alloc.Job); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	summary, err := state.JobSummaryByID(alloc.JobID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	tgSummary := summary.Summary["web"]
	if tgSummary.Starting != 1 {
		t.Fatalf("expected starting: %v, actual: %v", 1, tgSummary.Starting)
	}

	alloc2 := mock.Alloc()
	alloc2.ID = alloc.ID
	alloc2.NodeID = alloc.NodeID + ".new"
	state.UpsertJobSummary(1001, mock.JobSummary(alloc2.JobID))

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "allocs"},
		watch.Item{Alloc: alloc2.ID},
		watch.Item{AllocEval: alloc2.EvalID},
		watch.Item{AllocJob: alloc2.JobID},
		watch.Item{AllocNode: alloc2.NodeID})

	err = state.UpsertAllocs(1002, []*structs.Allocation{alloc2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(alloc2, out) {
		t.Fatalf("bad: %#v %#v", alloc2, out)
	}

	if out.CreateIndex != 1000 {
		t.Fatalf("bad: %#v", out)
	}
	if out.ModifyIndex != 1002 {
		t.Fatalf("bad: %#v", out)
	}

	index, err := state.Index("allocs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1002 {
		t.Fatalf("bad: %d", index)
	}

	// Ensure that summary hasb't changed
	summary, err = state.JobSummaryByID(alloc.JobID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	tgSummary = summary.Summary["web"]
	if tgSummary.Starting != 1 {
		t.Fatalf("expected starting: %v, actual: %v", 1, tgSummary.Starting)
	}

	notify.verify(t)
}

// This test ensures that the state store will mark the clients status as lost
// when set rather than preferring the existing status.
func TestStateStore_UpdateAlloc_Lost(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()
	alloc.ClientStatus = "foo"

	if err := state.UpsertJob(999, alloc.Job); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	alloc2 := new(structs.Allocation)
	*alloc2 = *alloc
	alloc2.ClientStatus = structs.AllocClientStatusLost
	if err := state.UpsertAllocs(1001, []*structs.Allocation{alloc2}); err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocByID(alloc2.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out.ClientStatus != structs.AllocClientStatusLost {
		t.Fatalf("bad: %#v", out)
	}
}

// This test ensures an allocation can be updated when there is no job
// associated with it. This will happen when a job is stopped by an user which
// has non-terminal allocations on clients
func TestStateStore_UpdateAlloc_NoJob(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()

	// Upsert a job
	state.UpsertJobSummary(998, mock.JobSummary(alloc.JobID))
	if err := state.UpsertJob(999, alloc.Job); err != nil {
		t.Fatalf("err: %v", err)
	}

	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := state.DeleteJob(1001, alloc.JobID); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Update the desired state of the allocation to stop
	allocCopy := alloc.Copy()
	allocCopy.DesiredStatus = structs.AllocDesiredStatusStop
	if err := state.UpsertAllocs(1002, []*structs.Allocation{allocCopy}); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Update the client state of the allocation to complete
	allocCopy1 := allocCopy.Copy()
	allocCopy1.ClientStatus = structs.AllocClientStatusComplete
	if err := state.UpdateAllocsFromClient(1003, []*structs.Allocation{allocCopy1}); err != nil {
		t.Fatalf("err: %v", err)
	}

	out, _ := state.AllocByID(alloc.ID)
	// Update the modify index of the alloc before comparing
	allocCopy1.ModifyIndex = 1003
	if !reflect.DeepEqual(out, allocCopy1) {
		t.Fatalf("expected: %#v \n actual: %#v", allocCopy1, out)
	}
}

func TestStateStore_JobSummary(t *testing.T) {
	state := testStateStore(t)

	// Add a job
	job := mock.Job()
	state.UpsertJob(900, job)

	// Get the job back
	outJob, _ := state.JobByID(job.ID)
	if outJob.CreateIndex != 900 {
		t.Fatalf("bad create index: %v", outJob.CreateIndex)
	}
	summary, _ := state.JobSummaryByID(job.ID)
	if summary.CreateIndex != 900 {
		t.Fatalf("bad create index: %v", summary.CreateIndex)
	}

	// Upser an allocation
	alloc := mock.Alloc()
	alloc.JobID = job.ID
	alloc.Job = job
	state.UpsertAllocs(910, []*structs.Allocation{alloc})

	// Update the alloc from client
	alloc1 := alloc.Copy()
	alloc1.ClientStatus = structs.AllocClientStatusPending
	alloc1.DesiredStatus = ""
	state.UpdateAllocsFromClient(920, []*structs.Allocation{alloc})

	alloc3 := alloc.Copy()
	alloc3.ClientStatus = structs.AllocClientStatusRunning
	alloc3.DesiredStatus = ""
	state.UpdateAllocsFromClient(930, []*structs.Allocation{alloc3})

	// Upsert the alloc
	alloc4 := alloc.Copy()
	alloc4.ClientStatus = structs.AllocClientStatusPending
	alloc4.DesiredStatus = structs.AllocDesiredStatusRun
	state.UpsertAllocs(950, []*structs.Allocation{alloc4})

	// Again upsert the alloc
	alloc5 := alloc.Copy()
	alloc5.ClientStatus = structs.AllocClientStatusPending
	alloc5.DesiredStatus = structs.AllocDesiredStatusRun
	state.UpsertAllocs(970, []*structs.Allocation{alloc5})

	expectedSummary := structs.JobSummary{
		JobID: job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Running: 1,
			},
		},
		CreateIndex: 900,
		ModifyIndex: 930,
	}

	summary, _ = state.JobSummaryByID(job.ID)
	if !reflect.DeepEqual(&expectedSummary, summary) {
		t.Fatalf("expected: %#v, actual: %v", expectedSummary, summary)
	}

	// De-register the job.
	state.DeleteJob(980, job.ID)

	// Shouldn't have any effect on the summary
	alloc6 := alloc.Copy()
	alloc6.ClientStatus = structs.AllocClientStatusRunning
	alloc6.DesiredStatus = ""
	state.UpdateAllocsFromClient(990, []*structs.Allocation{alloc6})

	// We shouldn't have any summary at this point
	summary, _ = state.JobSummaryByID(job.ID)
	if summary != nil {
		t.Fatalf("expected nil, actual: %#v", summary)
	}

	// Re-register the same job
	job1 := mock.Job()
	job1.ID = job.ID
	state.UpsertJob(1000, job1)
	outJob2, _ := state.JobByID(job1.ID)
	if outJob2.CreateIndex != 1000 {
		t.Fatalf("bad create index: %v", outJob2.CreateIndex)
	}
	summary, _ = state.JobSummaryByID(job1.ID)
	if summary.CreateIndex != 1000 {
		t.Fatalf("bad create index: %v", summary.CreateIndex)
	}

	// Upsert an allocation
	alloc7 := alloc.Copy()
	alloc7.JobID = outJob.ID
	alloc7.Job = outJob
	alloc7.ClientStatus = structs.AllocClientStatusComplete
	alloc7.DesiredStatus = structs.AllocDesiredStatusRun
	state.UpdateAllocsFromClient(1020, []*structs.Allocation{alloc7})

	expectedSummary = structs.JobSummary{
		JobID: job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{},
		},
		CreateIndex: 1000,
		ModifyIndex: 1000,
	}

	summary, _ = state.JobSummaryByID(job1.ID)
	if !reflect.DeepEqual(&expectedSummary, summary) {
		t.Fatalf("expected: %#v, actual: %#v", expectedSummary, summary)
	}
}

func TestStateStore_ReconcileJobSummary(t *testing.T) {
	state := testStateStore(t)

	// Create an alloc
	alloc := mock.Alloc()

	// Add another task group to the job
	tg2 := alloc.Job.TaskGroups[0].Copy()
	tg2.Name = "db"
	alloc.Job.TaskGroups = append(alloc.Job.TaskGroups, tg2)
	state.UpsertJob(100, alloc.Job)

	// Create one more alloc for the db task group
	alloc2 := mock.Alloc()
	alloc2.TaskGroup = "db"
	alloc2.JobID = alloc.JobID
	alloc2.Job = alloc.Job

	// Upserts the alloc
	state.UpsertAllocs(110, []*structs.Allocation{alloc, alloc2})

	// Change the state of the first alloc to running
	alloc3 := alloc.Copy()
	alloc3.ClientStatus = structs.AllocClientStatusRunning
	state.UpdateAllocsFromClient(120, []*structs.Allocation{alloc3})

	//Add some more allocs to the second tg
	alloc4 := mock.Alloc()
	alloc4.JobID = alloc.JobID
	alloc4.Job = alloc.Job
	alloc4.TaskGroup = "db"
	alloc5 := alloc4.Copy()
	alloc5.ClientStatus = structs.AllocClientStatusRunning

	alloc6 := mock.Alloc()
	alloc6.JobID = alloc.JobID
	alloc6.Job = alloc.Job
	alloc6.TaskGroup = "db"
	alloc7 := alloc6.Copy()
	alloc7.ClientStatus = structs.AllocClientStatusComplete

	alloc8 := mock.Alloc()
	alloc8.JobID = alloc.JobID
	alloc8.Job = alloc.Job
	alloc8.TaskGroup = "db"
	alloc9 := alloc8.Copy()
	alloc9.ClientStatus = structs.AllocClientStatusFailed

	alloc10 := mock.Alloc()
	alloc10.JobID = alloc.JobID
	alloc10.Job = alloc.Job
	alloc10.TaskGroup = "db"
	alloc11 := alloc10.Copy()
	alloc11.ClientStatus = structs.AllocClientStatusLost

	state.UpsertAllocs(130, []*structs.Allocation{alloc4, alloc6, alloc8, alloc10})

	state.UpdateAllocsFromClient(150, []*structs.Allocation{alloc5, alloc7, alloc9, alloc11})

	// DeleteJobSummary is a helper method and doesn't modify the indexes table
	state.DeleteJobSummary(130, alloc.Job.ID)

	state.ReconcileJobSummaries(120)

	summary, _ := state.JobSummaryByID(alloc.Job.ID)
	expectedSummary := structs.JobSummary{
		JobID: alloc.Job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{
				Running: 1,
			},
			"db": structs.TaskGroupSummary{
				Starting: 1,
				Running:  1,
				Failed:   1,
				Complete: 1,
				Lost:     1,
			},
		},
		CreateIndex: 100,
		ModifyIndex: 120,
	}
	if !reflect.DeepEqual(&expectedSummary, summary) {
		t.Fatalf("expected: %v, actual: %v", expectedSummary, summary)
	}
}

func TestStateStore_UpdateAlloc_JobNotPresent(t *testing.T) {
	state := testStateStore(t)

	alloc := mock.Alloc()
	state.UpsertJob(100, alloc.Job)
	state.UpsertAllocs(200, []*structs.Allocation{alloc})

	// Delete the job
	state.DeleteJob(300, alloc.Job.ID)

	// Update the alloc
	alloc1 := alloc.Copy()
	alloc1.ClientStatus = structs.AllocClientStatusRunning

	// Updating allocation should not throw any error
	if err := state.UpdateAllocsFromClient(400, []*structs.Allocation{alloc1}); err != nil {
		t.Fatalf("expect err: %v", err)
	}

	// Re-Register the job
	state.UpsertJob(500, alloc.Job)

	// Update the alloc again
	alloc2 := alloc.Copy()
	alloc2.ClientStatus = structs.AllocClientStatusComplete
	if err := state.UpdateAllocsFromClient(400, []*structs.Allocation{alloc1}); err != nil {
		t.Fatalf("expect err: %v", err)
	}

	// Job Summary of the newly registered job shouldn't account for the
	// allocation update for the older job
	expectedSummary := structs.JobSummary{
		JobID: alloc1.JobID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": structs.TaskGroupSummary{},
		},
		CreateIndex: 500,
		ModifyIndex: 500,
	}
	summary, _ := state.JobSummaryByID(alloc.Job.ID)
	if !reflect.DeepEqual(&expectedSummary, summary) {
		t.Fatalf("expected: %v, actual: %v", expectedSummary, summary)
	}
}

func TestStateStore_EvictAlloc_Alloc(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()

	state.UpsertJobSummary(999, mock.JobSummary(alloc.JobID))
	err := state.UpsertAllocs(1000, []*structs.Allocation{alloc})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	alloc2 := new(structs.Allocation)
	*alloc2 = *alloc
	alloc2.DesiredStatus = structs.AllocDesiredStatusEvict
	err = state.UpsertAllocs(1001, []*structs.Allocation{alloc2})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if out.DesiredStatus != structs.AllocDesiredStatusEvict {
		t.Fatalf("bad: %#v %#v", alloc, out)
	}

	index, err := state.Index("allocs")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if index != 1001 {
		t.Fatalf("bad: %d", index)
	}
}

func TestStateStore_AllocsByNode(t *testing.T) {
	state := testStateStore(t)
	var allocs []*structs.Allocation

	for i := 0; i < 10; i++ {
		alloc := mock.Alloc()
		alloc.NodeID = "foo"
		allocs = append(allocs, alloc)
	}

	for idx, alloc := range allocs {
		state.UpsertJobSummary(uint64(900+idx), mock.JobSummary(alloc.JobID))
	}

	err := state.UpsertAllocs(1000, allocs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocsByNode("foo")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sort.Sort(AllocIDSort(allocs))
	sort.Sort(AllocIDSort(out))

	if !reflect.DeepEqual(allocs, out) {
		t.Fatalf("bad: %#v %#v", allocs, out)
	}
}

func TestStateStore_AllocsByNodeTerminal(t *testing.T) {
	state := testStateStore(t)
	var allocs, term, nonterm []*structs.Allocation

	for i := 0; i < 10; i++ {
		alloc := mock.Alloc()
		alloc.NodeID = "foo"
		if i%2 == 0 {
			alloc.DesiredStatus = structs.AllocDesiredStatusStop
			term = append(term, alloc)
		} else {
			nonterm = append(nonterm, alloc)
		}
		allocs = append(allocs, alloc)
	}

	for idx, alloc := range allocs {
		state.UpsertJobSummary(uint64(900+idx), mock.JobSummary(alloc.JobID))
	}

	err := state.UpsertAllocs(1000, allocs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the terminal allocs
	out, err := state.AllocsByNodeTerminal("foo", true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sort.Sort(AllocIDSort(term))
	sort.Sort(AllocIDSort(out))

	if !reflect.DeepEqual(term, out) {
		t.Fatalf("bad: %#v %#v", term, out)
	}

	// Verify the non-terminal allocs
	out, err = state.AllocsByNodeTerminal("foo", false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sort.Sort(AllocIDSort(nonterm))
	sort.Sort(AllocIDSort(out))

	if !reflect.DeepEqual(nonterm, out) {
		t.Fatalf("bad: %#v %#v", nonterm, out)
	}
}

func TestStateStore_AllocsByJob(t *testing.T) {
	state := testStateStore(t)
	var allocs []*structs.Allocation

	for i := 0; i < 10; i++ {
		alloc := mock.Alloc()
		alloc.JobID = "foo"
		allocs = append(allocs, alloc)
	}

	for i, alloc := range allocs {
		state.UpsertJobSummary(uint64(900+i), mock.JobSummary(alloc.JobID))
	}

	err := state.UpsertAllocs(1000, allocs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out, err := state.AllocsByJob("foo")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	sort.Sort(AllocIDSort(allocs))
	sort.Sort(AllocIDSort(out))

	if !reflect.DeepEqual(allocs, out) {
		t.Fatalf("bad: %#v %#v", allocs, out)
	}
}

func TestStateStore_AllocsByIDPrefix(t *testing.T) {
	state := testStateStore(t)
	var allocs []*structs.Allocation

	ids := []string{
		"aaaaaaaa-7bfb-395d-eb95-0685af2176b2",
		"aaaaaaab-7bfb-395d-eb95-0685af2176b2",
		"aaaaaabb-7bfb-395d-eb95-0685af2176b2",
		"aaaaabbb-7bfb-395d-eb95-0685af2176b2",
		"aaaabbbb-7bfb-395d-eb95-0685af2176b2",
		"aaabbbbb-7bfb-395d-eb95-0685af2176b2",
		"aabbbbbb-7bfb-395d-eb95-0685af2176b2",
		"abbbbbbb-7bfb-395d-eb95-0685af2176b2",
		"bbbbbbbb-7bfb-395d-eb95-0685af2176b2",
	}
	for i := 0; i < 9; i++ {
		alloc := mock.Alloc()
		alloc.ID = ids[i]
		allocs = append(allocs, alloc)
	}

	for i, alloc := range allocs {
		state.UpsertJobSummary(uint64(900+i), mock.JobSummary(alloc.JobID))
	}

	err := state.UpsertAllocs(1000, allocs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err := state.AllocsByIDPrefix("aaaa")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	gatherAllocs := func(iter memdb.ResultIterator) []*structs.Allocation {
		var allocs []*structs.Allocation
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			allocs = append(allocs, raw.(*structs.Allocation))
		}
		return allocs
	}

	out := gatherAllocs(iter)
	if len(out) != 5 {
		t.Fatalf("bad: expected five allocations, got: %#v", out)
	}

	sort.Sort(AllocIDSort(allocs))

	for index, alloc := range out {
		if ids[index] != alloc.ID {
			t.Fatalf("bad: got unexpected id: %s", alloc.ID)
		}
	}

	iter, err = state.AllocsByIDPrefix("b-a7bfb")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	out = gatherAllocs(iter)
	if len(out) != 0 {
		t.Fatalf("bad: unexpected zero allocations, got: %#v", out)
	}
}

func TestStateStore_Allocs(t *testing.T) {
	state := testStateStore(t)
	var allocs []*structs.Allocation

	for i := 0; i < 10; i++ {
		alloc := mock.Alloc()
		allocs = append(allocs, alloc)
	}
	for i, alloc := range allocs {
		state.UpsertJobSummary(uint64(900+i), mock.JobSummary(alloc.JobID))
	}

	err := state.UpsertAllocs(1000, allocs)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	iter, err := state.Allocs()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out []*structs.Allocation
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		out = append(out, raw.(*structs.Allocation))
	}

	sort.Sort(AllocIDSort(allocs))
	sort.Sort(AllocIDSort(out))

	if !reflect.DeepEqual(allocs, out) {
		t.Fatalf("bad: %#v %#v", allocs, out)
	}
}

func TestStateStore_RestoreAlloc(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()

	notify := setupNotifyTest(
		state,
		watch.Item{Table: "allocs"},
		watch.Item{Alloc: alloc.ID},
		watch.Item{AllocEval: alloc.EvalID},
		watch.Item{AllocJob: alloc.JobID},
		watch.Item{AllocNode: alloc.NodeID})

	restore, err := state.Restore()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	err = restore.AllocRestore(alloc)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	restore.Commit()

	out, err := state.AllocByID(alloc.ID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(out, alloc) {
		t.Fatalf("Bad: %#v %#v", out, alloc)
	}

	notify.verify(t)
}

func TestStateStore_SetJobStatus_ForceStatus(t *testing.T) {
	state := testStateStore(t)
	watcher := watch.NewItems()
	txn := state.db.Txn(true)

	// Create and insert a mock job.
	job := mock.Job()
	job.Status = ""
	job.ModifyIndex = 0
	if err := txn.Insert("jobs", job); err != nil {
		t.Fatalf("job insert failed: %v", err)
	}

	exp := "foobar"
	index := uint64(1000)
	if err := state.setJobStatus(index, watcher, txn, job, false, exp); err != nil {
		t.Fatalf("setJobStatus() failed: %v", err)
	}

	i, err := txn.First("jobs", "id", job.ID)
	if err != nil {
		t.Fatalf("job lookup failed: %v", err)
	}
	updated := i.(*structs.Job)

	if updated.Status != exp {
		t.Fatalf("setJobStatus() set %v; expected %v", updated.Status, exp)
	}

	if updated.ModifyIndex != index {
		t.Fatalf("setJobStatus() set %d; expected %d", updated.ModifyIndex, index)
	}
}

func TestStateStore_SetJobStatus_NoOp(t *testing.T) {
	state := testStateStore(t)
	watcher := watch.NewItems()
	txn := state.db.Txn(true)

	// Create and insert a mock job that should be pending.
	job := mock.Job()
	job.Status = structs.JobStatusPending
	job.ModifyIndex = 10
	if err := txn.Insert("jobs", job); err != nil {
		t.Fatalf("job insert failed: %v", err)
	}

	index := uint64(1000)
	if err := state.setJobStatus(index, watcher, txn, job, false, ""); err != nil {
		t.Fatalf("setJobStatus() failed: %v", err)
	}

	i, err := txn.First("jobs", "id", job.ID)
	if err != nil {
		t.Fatalf("job lookup failed: %v", err)
	}
	updated := i.(*structs.Job)

	if updated.ModifyIndex == index {
		t.Fatalf("setJobStatus() should have been a no-op")
	}
}

func TestStateStore_SetJobStatus(t *testing.T) {
	state := testStateStore(t)
	watcher := watch.NewItems()
	txn := state.db.Txn(true)

	// Create and insert a mock job that should be pending but has an incorrect
	// status.
	job := mock.Job()
	job.Status = "foobar"
	job.ModifyIndex = 10
	if err := txn.Insert("jobs", job); err != nil {
		t.Fatalf("job insert failed: %v", err)
	}

	index := uint64(1000)
	if err := state.setJobStatus(index, watcher, txn, job, false, ""); err != nil {
		t.Fatalf("setJobStatus() failed: %v", err)
	}

	i, err := txn.First("jobs", "id", job.ID)
	if err != nil {
		t.Fatalf("job lookup failed: %v", err)
	}
	updated := i.(*structs.Job)

	if updated.Status != structs.JobStatusPending {
		t.Fatalf("setJobStatus() set %v; expected %v", updated.Status, structs.JobStatusPending)
	}

	if updated.ModifyIndex != index {
		t.Fatalf("setJobStatus() set %d; expected %d", updated.ModifyIndex, index)
	}
}

func TestStateStore_GetJobStatus_NoEvalsOrAllocs(t *testing.T) {
	job := mock.Job()
	state := testStateStore(t)
	txn := state.db.Txn(false)
	status, err := state.getJobStatus(txn, job, false)
	if err != nil {
		t.Fatalf("getJobStatus() failed: %v", err)
	}

	if status != structs.JobStatusPending {
		t.Fatalf("getJobStatus() returned %v; expected %v", status, structs.JobStatusPending)
	}
}

func TestStateStore_GetJobStatus_NoEvalsOrAllocs_Periodic(t *testing.T) {
	job := mock.PeriodicJob()
	state := testStateStore(t)
	txn := state.db.Txn(false)
	status, err := state.getJobStatus(txn, job, false)
	if err != nil {
		t.Fatalf("getJobStatus() failed: %v", err)
	}

	if status != structs.JobStatusRunning {
		t.Fatalf("getJobStatus() returned %v; expected %v", status, structs.JobStatusRunning)
	}
}

func TestStateStore_GetJobStatus_NoEvalsOrAllocs_EvalDelete(t *testing.T) {
	job := mock.Job()
	state := testStateStore(t)
	txn := state.db.Txn(false)
	status, err := state.getJobStatus(txn, job, true)
	if err != nil {
		t.Fatalf("getJobStatus() failed: %v", err)
	}

	if status != structs.JobStatusDead {
		t.Fatalf("getJobStatus() returned %v; expected %v", status, structs.JobStatusDead)
	}
}

func TestStateStore_GetJobStatus_DeadEvalsAndAllocs(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	// Create a mock alloc that is dead.
	alloc := mock.Alloc()
	alloc.JobID = job.ID
	alloc.DesiredStatus = structs.AllocDesiredStatusStop
	state.UpsertJobSummary(999, mock.JobSummary(alloc.JobID))
	if err := state.UpsertAllocs(1000, []*structs.Allocation{alloc}); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a mock eval that is complete
	eval := mock.Eval()
	eval.JobID = job.ID
	eval.Status = structs.EvalStatusComplete
	if err := state.UpsertEvals(1001, []*structs.Evaluation{eval}); err != nil {
		t.Fatalf("err: %v", err)
	}

	txn := state.db.Txn(false)
	status, err := state.getJobStatus(txn, job, false)
	if err != nil {
		t.Fatalf("getJobStatus() failed: %v", err)
	}

	if status != structs.JobStatusDead {
		t.Fatalf("getJobStatus() returned %v; expected %v", status, structs.JobStatusDead)
	}
}

func TestStateStore_GetJobStatus_RunningAlloc(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	// Create a mock alloc that is running.
	alloc := mock.Alloc()
	alloc.JobID = job.ID
	alloc.DesiredStatus = structs.AllocDesiredStatusRun
	state.UpsertJobSummary(999, mock.JobSummary(alloc.JobID))
	if err := state.UpsertAllocs(1000, []*structs.Allocation{alloc}); err != nil {
		t.Fatalf("err: %v", err)
	}

	txn := state.db.Txn(false)
	status, err := state.getJobStatus(txn, job, true)
	if err != nil {
		t.Fatalf("getJobStatus() failed: %v", err)
	}

	if status != structs.JobStatusRunning {
		t.Fatalf("getJobStatus() returned %v; expected %v", status, structs.JobStatusRunning)
	}
}

func TestStateStore_SetJobStatus_PendingEval(t *testing.T) {
	state := testStateStore(t)
	job := mock.Job()

	// Create a mock eval that is pending.
	eval := mock.Eval()
	eval.JobID = job.ID
	eval.Status = structs.EvalStatusPending
	if err := state.UpsertEvals(1000, []*structs.Evaluation{eval}); err != nil {
		t.Fatalf("err: %v", err)
	}

	txn := state.db.Txn(false)
	status, err := state.getJobStatus(txn, job, true)
	if err != nil {
		t.Fatalf("getJobStatus() failed: %v", err)
	}

	if status != structs.JobStatusPending {
		t.Fatalf("getJobStatus() returned %v; expected %v", status, structs.JobStatusPending)
	}
}

func TestStateWatch_watch(t *testing.T) {
	sw := newStateWatch()
	notify1 := make(chan struct{}, 1)
	notify2 := make(chan struct{}, 1)
	notify3 := make(chan struct{}, 1)

	// Notifications trigger subscribed channels
	sw.watch(watch.NewItems(watch.Item{Table: "foo"}), notify1)
	sw.watch(watch.NewItems(watch.Item{Table: "bar"}), notify2)
	sw.watch(watch.NewItems(watch.Item{Table: "baz"}), notify3)

	items := watch.NewItems()
	items.Add(watch.Item{Table: "foo"})
	items.Add(watch.Item{Table: "bar"})

	sw.notify(items)
	if len(notify1) != 1 {
		t.Fatalf("should notify")
	}
	if len(notify2) != 1 {
		t.Fatalf("should notify")
	}
	if len(notify3) != 0 {
		t.Fatalf("should not notify")
	}
}

func TestStateWatch_stopWatch(t *testing.T) {
	sw := newStateWatch()
	notify := make(chan struct{})

	// First subscribe
	sw.watch(watch.NewItems(watch.Item{Table: "foo"}), notify)

	// Unsubscribe stop notifications
	sw.stopWatch(watch.NewItems(watch.Item{Table: "foo"}), notify)

	// Check that the group was removed
	if _, ok := sw.items[watch.Item{Table: "foo"}]; ok {
		t.Fatalf("should remove group")
	}

	// Check that we are not notified
	sw.notify(watch.NewItems(watch.Item{Table: "foo"}))
	if len(notify) != 0 {
		t.Fatalf("should not notify")
	}
}

func TestStateJobSummary_UpdateJobCount(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()
	job := alloc.Job
	job.TaskGroups[0].Count = 3
	err := state.UpsertJob(1000, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := state.UpsertAllocs(1001, []*structs.Allocation{alloc}); err != nil {
		t.Fatalf("err: %v", err)
	}
	summary, _ := state.JobSummaryByID(job.ID)
	expectedSummary := structs.JobSummary{
		JobID: job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": {
				Starting: 1,
			},
		},
		CreateIndex: 1000,
		ModifyIndex: 1001,
	}
	if !reflect.DeepEqual(summary, &expectedSummary) {
		t.Fatalf("expected: %v, actual: %v", expectedSummary, summary)
	}

	alloc2 := mock.Alloc()
	alloc2.Job = job
	alloc2.JobID = job.ID

	alloc3 := mock.Alloc()
	alloc3.Job = job
	alloc3.JobID = job.ID

	if err := state.UpsertAllocs(1002, []*structs.Allocation{alloc2, alloc3}); err != nil {
		t.Fatalf("err: %v", err)
	}

	outA, _ := state.AllocByID(alloc3.ID)

	summary, _ = state.JobSummaryByID(job.ID)
	expectedSummary = structs.JobSummary{
		JobID: job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": {
				Starting: 3,
			},
		},
		CreateIndex: job.CreateIndex,
		ModifyIndex: outA.ModifyIndex,
	}
	if !reflect.DeepEqual(summary, &expectedSummary) {
		t.Fatalf("expected summary: %v, actual: %v", expectedSummary, summary)
	}

	alloc4 := mock.Alloc()
	alloc4.ID = alloc2.ID
	alloc4.Job = alloc2.Job
	alloc4.JobID = alloc2.JobID
	alloc4.ClientStatus = structs.AllocClientStatusComplete

	alloc5 := mock.Alloc()
	alloc5.ID = alloc3.ID
	alloc5.Job = alloc3.Job
	alloc5.JobID = alloc3.JobID
	alloc5.ClientStatus = structs.AllocClientStatusComplete

	if err := state.UpdateAllocsFromClient(1004, []*structs.Allocation{alloc4, alloc5}); err != nil {
		t.Fatalf("err: %v", err)
	}
	outA, _ = state.AllocByID(alloc5.ID)
	summary, _ = state.JobSummaryByID(job.ID)
	expectedSummary = structs.JobSummary{
		JobID: job.ID,
		Summary: map[string]structs.TaskGroupSummary{
			"web": {
				Complete: 2,
				Starting: 1,
			},
		},
		CreateIndex: job.CreateIndex,
		ModifyIndex: outA.ModifyIndex,
	}
	if !reflect.DeepEqual(summary, &expectedSummary) {
		t.Fatalf("expected: %v, actual: %v", expectedSummary, summary)
	}
}

func TestJobSummary_UpdateClientStatus(t *testing.T) {
	state := testStateStore(t)
	alloc := mock.Alloc()
	job := alloc.Job
	job.TaskGroups[0].Count = 3

	alloc2 := mock.Alloc()
	alloc2.Job = job
	alloc2.JobID = job.ID

	alloc3 := mock.Alloc()
	alloc3.Job = job
	alloc3.JobID = job.ID

	err := state.UpsertJob(1000, job)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := state.UpsertAllocs(1001, []*structs.Allocation{alloc, alloc2, alloc3}); err != nil {
		t.Fatalf("err: %v", err)
	}
	summary, _ := state.JobSummaryByID(job.ID)
	if summary.Summary["web"].Starting != 3 {
		t.Fatalf("bad job summary: %v", summary)
	}

	alloc4 := mock.Alloc()
	alloc4.ID = alloc2.ID
	alloc4.Job = alloc2.Job
	alloc4.JobID = alloc2.JobID
	alloc4.ClientStatus = structs.AllocClientStatusComplete

	alloc5 := mock.Alloc()
	alloc5.ID = alloc3.ID
	alloc5.Job = alloc3.Job
	alloc5.JobID = alloc3.JobID
	alloc5.ClientStatus = structs.AllocClientStatusFailed

	alloc6 := mock.Alloc()
	alloc6.ID = alloc.ID
	alloc6.Job = alloc.Job
	alloc6.JobID = alloc.JobID
	alloc6.ClientStatus = structs.AllocClientStatusRunning

	if err := state.UpdateAllocsFromClient(1002, []*structs.Allocation{alloc4, alloc5, alloc6}); err != nil {
		t.Fatalf("err: %v", err)
	}
	summary, _ = state.JobSummaryByID(job.ID)
	if summary.Summary["web"].Running != 1 || summary.Summary["web"].Failed != 1 || summary.Summary["web"].Complete != 1 {
		t.Fatalf("bad job summary: %v", summary)
	}

	alloc7 := mock.Alloc()
	alloc7.Job = alloc.Job
	alloc7.JobID = alloc.JobID

	if err := state.UpsertAllocs(1003, []*structs.Allocation{alloc7}); err != nil {
		t.Fatalf("err: %v", err)
	}
	summary, _ = state.JobSummaryByID(job.ID)
	if summary.Summary["web"].Starting != 1 || summary.Summary["web"].Running != 1 || summary.Summary["web"].Failed != 1 || summary.Summary["web"].Complete != 1 {
		t.Fatalf("bad job summary: %v", summary)
	}
}

// setupNotifyTest takes a state store and a set of watch items, then creates
// and subscribes a notification channel for each item.
func setupNotifyTest(state *StateStore, items ...watch.Item) notifyTest {
	var n notifyTest
	for _, item := range items {
		ch := make(chan struct{}, 1)
		state.Watch(watch.NewItems(item), ch)
		n = append(n, &notifyTestCase{item, ch})
	}
	return n
}

// notifyTestCase is used to set up and verify watch triggers.
type notifyTestCase struct {
	item watch.Item
	ch   chan struct{}
}

// notifyTest is a suite of notifyTestCases.
type notifyTest []*notifyTestCase

// verify ensures that each channel received a notification.
func (n notifyTest) verify(t *testing.T) {
	for _, tcase := range n {
		if len(tcase.ch) != 1 {
			t.Fatalf("should notify %#v", tcase.item)
		}
	}
}

// NodeIDSort is used to sort nodes by ID
type NodeIDSort []*structs.Node

func (n NodeIDSort) Len() int {
	return len(n)
}

func (n NodeIDSort) Less(i, j int) bool {
	return n[i].ID < n[j].ID
}

func (n NodeIDSort) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// JobIDis used to sort jobs by id
type JobIDSort []*structs.Job

func (n JobIDSort) Len() int {
	return len(n)
}

func (n JobIDSort) Less(i, j int) bool {
	return n[i].ID < n[j].ID
}

func (n JobIDSort) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// EvalIDis used to sort evals by id
type EvalIDSort []*structs.Evaluation

func (n EvalIDSort) Len() int {
	return len(n)
}

func (n EvalIDSort) Less(i, j int) bool {
	return n[i].ID < n[j].ID
}

func (n EvalIDSort) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// AllocIDsort used to sort allocations by id
type AllocIDSort []*structs.Allocation

func (n AllocIDSort) Len() int {
	return len(n)
}

func (n AllocIDSort) Less(i, j int) bool {
	return n[i].ID < n[j].ID
}

func (n AllocIDSort) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
