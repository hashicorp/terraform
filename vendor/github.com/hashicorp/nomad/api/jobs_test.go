package api

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/testutil"
)

func TestJobs_Register(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Listing jobs before registering returns nothing
	resp, qm, err := jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(resp); n != 0 {
		t.Fatalf("expected 0 jobs, got: %d", n)
	}

	// Create a job and attempt to register it
	job := testJob()
	eval, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if eval == "" {
		t.Fatalf("missing eval id")
	}
	assertWriteMeta(t, wm)

	// Query the jobs back out again
	resp, qm, err = jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check that we got the expected response
	if len(resp) != 1 || resp[0].ID != job.ID {
		t.Fatalf("bad: %#v", resp[0])
	}
}

func TestJobs_EnforceRegister(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Listing jobs before registering returns nothing
	resp, qm, err := jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(resp); n != 0 {
		t.Fatalf("expected 0 jobs, got: %d", n)
	}

	// Create a job and attempt to register it with an incorrect index.
	job := testJob()
	eval, wm, err := jobs.EnforceRegister(job, 10, nil)
	if err == nil || !strings.Contains(err.Error(), RegisterEnforceIndexErrPrefix) {
		t.Fatalf("expected enforcement error: %v", err)
	}

	// Register
	eval, wm, err = jobs.EnforceRegister(job, 0, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if eval == "" {
		t.Fatalf("missing eval id")
	}
	assertWriteMeta(t, wm)

	// Query the jobs back out again
	resp, qm, err = jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check that we got the expected response
	if len(resp) != 1 {
		t.Fatalf("bad length: %d", len(resp))
	}

	if resp[0].ID != job.ID {
		t.Fatalf("bad: %#v", resp[0])
	}
	curIndex := resp[0].JobModifyIndex

	// Fail at incorrect index
	eval, wm, err = jobs.EnforceRegister(job, 123456, nil)
	if err == nil || !strings.Contains(err.Error(), RegisterEnforceIndexErrPrefix) {
		t.Fatalf("expected enforcement error: %v", err)
	}

	// Works at correct index
	eval, wm, err = jobs.EnforceRegister(job, curIndex, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if eval == "" {
		t.Fatalf("missing eval id")
	}
	assertWriteMeta(t, wm)
}

func TestJobs_Info(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Trying to retrieve a job by ID before it exists
	// returns an error
	_, _, err := jobs.Info("job1", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %#v", err)
	}

	// Register the job
	job := testJob()
	_, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Query the job again and ensure it exists
	result, qm, err := jobs.Info("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check that the result is what we expect
	if result == nil || result.ID != job.ID {
		t.Fatalf("expect: %#v, got: %#v", job, result)
	}
}

func TestJobs_PrefixList(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Listing when nothing exists returns empty
	results, qm, err := jobs.PrefixList("dummy")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(results); n != 0 {
		t.Fatalf("expected 0 jobs, got: %d", n)
	}

	// Register the job
	job := testJob()
	_, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Query the job again and ensure it exists
	// Listing when nothing exists returns empty
	results, qm, err = jobs.PrefixList(job.ID[:1])
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check if we have the right list
	if len(results) != 1 || results[0].ID != job.ID {
		t.Fatalf("bad: %#v", results)
	}
}

func TestJobs_List(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Listing when nothing exists returns empty
	results, qm, err := jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(results); n != 0 {
		t.Fatalf("expected 0 jobs, got: %d", n)
	}

	// Register the job
	job := testJob()
	_, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Query the job again and ensure it exists
	// Listing when nothing exists returns empty
	results, qm, err = jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check if we have the right list
	if len(results) != 1 || results[0].ID != job.ID {
		t.Fatalf("bad: %#v", results)
	}
}

func TestJobs_Allocations(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Looking up by a non-existent job returns nothing
	allocs, qm, err := jobs.Allocations("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(allocs); n != 0 {
		t.Fatalf("expected 0 allocs, got: %d", n)
	}

	// TODO: do something here to create some allocations for
	// an existing job, lookup again.
}

func TestJobs_Evaluations(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Looking up by a non-existent job ID returns nothing
	evals, qm, err := jobs.Evaluations("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(evals); n != 0 {
		t.Fatalf("expected 0 evals, got: %d", n)
	}

	// Insert a job. This also creates an evaluation so we should
	// be able to query that out after.
	job := testJob()
	evalID, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Look up the evaluations again.
	evals, qm, err = jobs.Evaluations("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check that we got the evals back, evals are in order most recent to least recent
	// so the last eval is the original registered eval
	idx := len(evals) - 1
	if n := len(evals); n == 0 || evals[idx].ID != evalID {
		t.Fatalf("expected >= 1 eval (%s), got: %#v", evalID, evals[idx])
	}
}

func TestJobs_Deregister(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Register a new job
	job := testJob()
	_, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Attempting delete on non-existing job returns an error
	if _, _, err = jobs.Deregister("nope", nil); err != nil {
		t.Fatalf("unexpected error deregistering job: %v", err)

	}

	// Deleting an existing job works
	evalID, wm3, err := jobs.Deregister("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm3)
	if evalID == "" {
		t.Fatalf("missing eval ID")
	}

	// Check that the job is really gone
	result, qm, err := jobs.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)
	if n := len(result); n != 0 {
		t.Fatalf("expected 0 jobs, got: %d", n)
	}
}

func TestJobs_ForceEvaluate(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Force-eval on a non-existent job fails
	_, _, err := jobs.ForceEvaluate("job1", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %#v", err)
	}

	// Create a new job
	_, wm, err := jobs.Register(testJob(), nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Try force-eval again
	evalID, wm, err := jobs.ForceEvaluate("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Retrieve the evals and see if we get a matching one
	evals, qm, err := jobs.Evaluations("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)
	for _, eval := range evals {
		if eval.ID == evalID {
			return
		}
	}
	t.Fatalf("evaluation %q missing", evalID)
}

func TestJobs_PeriodicForce(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Force-eval on a non-existent job fails
	_, _, err := jobs.PeriodicForce("job1", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %#v", err)
	}

	// Create a new job
	job := testPeriodicJob()
	_, _, err = jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testutil.WaitForResult(func() (bool, error) {
		out, _, err := jobs.Info(job.ID, nil)
		if err != nil || out == nil || out.ID != job.ID {
			return false, err
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})

	// Try force again
	evalID, wm, err := jobs.PeriodicForce(job.ID, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	if evalID == "" {
		t.Fatalf("empty evalID")
	}

	// Retrieve the eval
	evals := c.Evaluations()
	eval, qm, err := evals.Info(evalID, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)
	if eval.ID == evalID {
		return
	}
	t.Fatalf("evaluation %q missing", evalID)
}

func TestJobs_Plan(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Create a job and attempt to register it
	job := testJob()
	eval, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if eval == "" {
		t.Fatalf("missing eval id")
	}
	assertWriteMeta(t, wm)

	// Check that passing a nil job fails
	if _, _, err := jobs.Plan(nil, true, nil); err == nil {
		t.Fatalf("expect an error when job isn't provided")
	}

	// Make a plan request
	planResp, wm, err := jobs.Plan(job, true, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if planResp == nil {
		t.Fatalf("nil response")
	}

	if planResp.JobModifyIndex == 0 {
		t.Fatalf("bad JobModifyIndex value: %#v", planResp)
	}
	if planResp.Diff == nil {
		t.Fatalf("got nil diff: %#v", planResp)
	}
	if planResp.Annotations == nil {
		t.Fatalf("got nil annotations: %#v", planResp)
	}
	// Can make this assertion because there are no clients.
	if len(planResp.CreatedEvals) == 0 {
		t.Fatalf("got no CreatedEvals: %#v", planResp)
	}

	// Make a plan request w/o the diff
	planResp, wm, err = jobs.Plan(job, false, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	if planResp == nil {
		t.Fatalf("nil response")
	}

	if planResp.JobModifyIndex == 0 {
		t.Fatalf("bad JobModifyIndex value: %d", planResp.JobModifyIndex)
	}
	if planResp.Diff != nil {
		t.Fatalf("got non-nil diff: %#v", planResp)
	}
	if planResp.Annotations == nil {
		t.Fatalf("got nil annotations: %#v", planResp)
	}
	// Can make this assertion because there are no clients.
	if len(planResp.CreatedEvals) == 0 {
		t.Fatalf("got no CreatedEvals: %#v", planResp)
	}
}

func TestJobs_JobSummary(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	jobs := c.Jobs()

	// Trying to retrieve a job summary before the job exists
	// returns an error
	_, _, err := jobs.Summary("job1", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %#v", err)
	}

	// Register the job
	job := testJob()
	taskName := job.TaskGroups[0].Name
	_, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Query the job summary again and ensure it exists
	result, qm, err := jobs.Summary("job1", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check that the result is what we expect
	if job.ID != result.JobID {
		t.Fatalf("err: expected job id of %s saw %s", job.ID, result.JobID)
	}
	if _, ok := result.Summary[taskName]; !ok {
		t.Fatalf("err: unable to find %s key in job summary", taskName)
	}
}

func TestJobs_NewBatchJob(t *testing.T) {
	job := NewBatchJob("job1", "myjob", "region1", 5)
	expect := &Job{
		Region:   "region1",
		ID:       "job1",
		Name:     "myjob",
		Type:     JobTypeBatch,
		Priority: 5,
	}
	if !reflect.DeepEqual(job, expect) {
		t.Fatalf("expect: %#v, got: %#v", expect, job)
	}
}

func TestJobs_NewServiceJob(t *testing.T) {
	job := NewServiceJob("job1", "myjob", "region1", 5)
	expect := &Job{
		Region:   "region1",
		ID:       "job1",
		Name:     "myjob",
		Type:     JobTypeService,
		Priority: 5,
	}
	if !reflect.DeepEqual(job, expect) {
		t.Fatalf("expect: %#v, got: %#v", expect, job)
	}
}

func TestJobs_SetMeta(t *testing.T) {
	job := &Job{Meta: nil}

	// Initializes a nil map
	out := job.SetMeta("foo", "bar")
	if job.Meta == nil {
		t.Fatalf("should initialize metadata")
	}

	// Check that the job was returned
	if job != out {
		t.Fatalf("expect: %#v, got: %#v", job, out)
	}

	// Setting another pair is additive
	job.SetMeta("baz", "zip")
	expect := map[string]string{"foo": "bar", "baz": "zip"}
	if !reflect.DeepEqual(job.Meta, expect) {
		t.Fatalf("expect: %#v, got: %#v", expect, job.Meta)
	}
}

func TestJobs_Constrain(t *testing.T) {
	job := &Job{Constraints: nil}

	// Create and add a constraint
	out := job.Constrain(NewConstraint("kernel.name", "=", "darwin"))
	if n := len(job.Constraints); n != 1 {
		t.Fatalf("expected 1 constraint, got: %d", n)
	}

	// Check that the job was returned
	if job != out {
		t.Fatalf("expect: %#v, got: %#v", job, out)
	}

	// Adding another constraint preserves the original
	job.Constrain(NewConstraint("memory.totalbytes", ">=", "128000000"))
	expect := []*Constraint{
		&Constraint{
			LTarget: "kernel.name",
			RTarget: "darwin",
			Operand: "=",
		},
		&Constraint{
			LTarget: "memory.totalbytes",
			RTarget: "128000000",
			Operand: ">=",
		},
	}
	if !reflect.DeepEqual(job.Constraints, expect) {
		t.Fatalf("expect: %#v, got: %#v", expect, job.Constraints)
	}
}

func TestJobs_Sort(t *testing.T) {
	jobs := []*JobListStub{
		&JobListStub{ID: "job2"},
		&JobListStub{ID: "job0"},
		&JobListStub{ID: "job1"},
	}
	sort.Sort(JobIDSort(jobs))

	expect := []*JobListStub{
		&JobListStub{ID: "job0"},
		&JobListStub{ID: "job1"},
		&JobListStub{ID: "job2"},
	}
	if !reflect.DeepEqual(jobs, expect) {
		t.Fatalf("\n\n%#v\n\n%#v", jobs, expect)
	}
}
