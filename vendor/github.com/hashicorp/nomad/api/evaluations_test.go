package api

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestEvaluations_List(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	e := c.Evaluations()

	// Listing when nothing exists returns empty
	result, qm, err := e.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(result); n != 0 {
		t.Fatalf("expected 0 evaluations, got: %d", n)
	}

	// Register a job. This will create an evaluation.
	jobs := c.Jobs()
	job := testJob()
	evalID, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Check the evaluations again
	result, qm, err = e.List(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// if the eval fails fast there can be more than 1
	// but they are in order of most recent first, so look at the last one
	idx := len(result) - 1
	if len(result) == 0 || result[idx].ID != evalID {
		t.Fatalf("expected eval (%s), got: %#v", evalID, result[idx])
	}
}

func TestEvaluations_PrefixList(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	e := c.Evaluations()

	// Listing when nothing exists returns empty
	result, qm, err := e.PrefixList("abcdef")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(result); n != 0 {
		t.Fatalf("expected 0 evaluations, got: %d", n)
	}

	// Register a job. This will create an evaluation.
	jobs := c.Jobs()
	job := testJob()
	evalID, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Check the evaluations again
	result, qm, err = e.PrefixList(evalID[:4])
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check if we have the right list
	if len(result) != 1 || result[0].ID != evalID {
		t.Fatalf("bad: %#v", result)
	}
}

func TestEvaluations_Info(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	e := c.Evaluations()

	// Querying a non-existent evaluation returns error
	_, _, err := e.Info("8E231CF4-CA48-43FF-B694-5801E69E22FA", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %s", err)
	}

	// Register a job. Creates a new evaluation.
	jobs := c.Jobs()
	job := testJob()
	evalID, wm, err := jobs.Register(job, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertWriteMeta(t, wm)

	// Try looking up by the new eval ID
	result, qm, err := e.Info(evalID, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	assertQueryMeta(t, qm)

	// Check that we got the right result
	if result == nil || result.ID != evalID {
		t.Fatalf("expected eval %q, got: %#v", evalID, result)
	}
}

func TestEvaluations_Allocations(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	e := c.Evaluations()

	// Returns empty if no allocations
	allocs, qm, err := e.Allocations("8E231CF4-CA48-43FF-B694-5801E69E22FA", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if qm.LastIndex != 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if n := len(allocs); n != 0 {
		t.Fatalf("expected 0 allocs, got: %d", n)
	}
}

func TestEvaluations_Sort(t *testing.T) {
	evals := []*Evaluation{
		&Evaluation{CreateIndex: 2},
		&Evaluation{CreateIndex: 1},
		&Evaluation{CreateIndex: 5},
	}
	sort.Sort(EvalIndexSort(evals))

	expect := []*Evaluation{
		&Evaluation{CreateIndex: 5},
		&Evaluation{CreateIndex: 2},
		&Evaluation{CreateIndex: 1},
	}
	if !reflect.DeepEqual(evals, expect) {
		t.Fatalf("\n\n%#v\n\n%#v", evals, expect)
	}
}
