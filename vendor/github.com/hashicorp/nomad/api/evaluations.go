package api

import (
	"sort"
	"time"
)

// Evaluations is used to query the evaluation endpoints.
type Evaluations struct {
	client *Client
}

// Evaluations returns a new handle on the evaluations.
func (c *Client) Evaluations() *Evaluations {
	return &Evaluations{client: c}
}

// List is used to dump all of the evaluations.
func (e *Evaluations) List(q *QueryOptions) ([]*Evaluation, *QueryMeta, error) {
	var resp []*Evaluation
	qm, err := e.client.query("/v1/evaluations", &resp, q)
	if err != nil {
		return nil, nil, err
	}
	sort.Sort(EvalIndexSort(resp))
	return resp, qm, nil
}

func (e *Evaluations) PrefixList(prefix string) ([]*Evaluation, *QueryMeta, error) {
	return e.List(&QueryOptions{Prefix: prefix})
}

// Info is used to query a single evaluation by its ID.
func (e *Evaluations) Info(evalID string, q *QueryOptions) (*Evaluation, *QueryMeta, error) {
	var resp Evaluation
	qm, err := e.client.query("/v1/evaluation/"+evalID, &resp, q)
	if err != nil {
		return nil, nil, err
	}
	return &resp, qm, nil
}

// Allocations is used to retrieve a set of allocations given
// an evaluation ID.
func (e *Evaluations) Allocations(evalID string, q *QueryOptions) ([]*AllocationListStub, *QueryMeta, error) {
	var resp []*AllocationListStub
	qm, err := e.client.query("/v1/evaluation/"+evalID+"/allocations", &resp, q)
	if err != nil {
		return nil, nil, err
	}
	sort.Sort(AllocIndexSort(resp))
	return resp, qm, nil
}

// Evaluation is used to serialize an evaluation.
type Evaluation struct {
	ID                string
	Priority          int
	Type              string
	TriggeredBy       string
	JobID             string
	JobModifyIndex    uint64
	NodeID            string
	NodeModifyIndex   uint64
	Status            string
	StatusDescription string
	Wait              time.Duration
	NextEval          string
	PreviousEval      string
	BlockedEval       string
	FailedTGAllocs    map[string]*AllocationMetric
	CreateIndex       uint64
	ModifyIndex       uint64
}

// EvalIndexSort is a wrapper to sort evaluations by CreateIndex.
// We reverse the test so that we get the highest index first.
type EvalIndexSort []*Evaluation

func (e EvalIndexSort) Len() int {
	return len(e)
}

func (e EvalIndexSort) Less(i, j int) bool {
	return e[i].CreateIndex > e[j].CreateIndex
}

func (e EvalIndexSort) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
