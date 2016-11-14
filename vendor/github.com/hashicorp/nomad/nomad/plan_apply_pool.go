package nomad

import (
	"github.com/hashicorp/nomad/nomad/state"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	// workerPoolBufferSize is the size of the buffers used to push
	// request to the workers and to collect the responses. It should
	// be large enough just to keep things busy
	workerPoolBufferSize = 64
)

// EvaluatePool is used to have a pool of workers that are evaluating
// if a plan is valid. It can be used to parallelize the evaluation
// of a plan.
type EvaluatePool struct {
	workers    int
	workerStop []chan struct{}
	req        chan evaluateRequest
	res        chan evaluateResult
}

type evaluateRequest struct {
	snap   *state.StateSnapshot
	plan   *structs.Plan
	nodeID string
}

type evaluateResult struct {
	nodeID string
	fit    bool
	err    error
}

// NewEvaluatePool returns a pool of the given size.
func NewEvaluatePool(workers, bufSize int) *EvaluatePool {
	p := &EvaluatePool{
		workers:    workers,
		workerStop: make([]chan struct{}, workers),
		req:        make(chan evaluateRequest, bufSize),
		res:        make(chan evaluateResult, bufSize),
	}
	for i := 0; i < workers; i++ {
		stopCh := make(chan struct{})
		p.workerStop[i] = stopCh
		go p.run(stopCh)
	}
	return p
}

// Size returns the current size
func (p *EvaluatePool) Size() int {
	return p.workers
}

// SetSize is used to resize the worker pool
func (p *EvaluatePool) SetSize(size int) {
	// Protect against a negative size
	if size < 0 {
		size = 0
	}

	// Handle an upwards resize
	if size >= p.workers {
		for i := p.workers; i < size; i++ {
			stopCh := make(chan struct{})
			p.workerStop = append(p.workerStop, stopCh)
			go p.run(stopCh)
		}
		p.workers = size
		return
	}

	// Handle a downwards resize
	for i := p.workers; i > size; i-- {
		close(p.workerStop[i-1])
		p.workerStop[i-1] = nil
	}
	p.workerStop = p.workerStop[:size]
	p.workers = size
}

// RequestCh is used to push requests
func (p *EvaluatePool) RequestCh() chan<- evaluateRequest {
	return p.req
}

// ResultCh is used to read the results as they are ready
func (p *EvaluatePool) ResultCh() <-chan evaluateResult {
	return p.res
}

// Shutdown is used to shutdown the pool
func (p *EvaluatePool) Shutdown() {
	p.SetSize(0)
}

// run is a long running go routine per worker
func (p *EvaluatePool) run(stopCh chan struct{}) {
	for {
		select {
		case req := <-p.req:
			fit, err := evaluateNodePlan(req.snap, req.plan, req.nodeID)
			p.res <- evaluateResult{req.nodeID, fit, err}

		case <-stopCh:
			return
		}
	}
}
