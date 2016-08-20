package nomad

import (
	"testing"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
)

func TestEvaluatePool(t *testing.T) {
	state := testStateStore(t)
	node := mock.Node()
	state.UpsertNode(1000, node)
	snap, _ := state.Snapshot()

	alloc := mock.Alloc()
	plan := &structs.Plan{
		NodeAllocation: map[string][]*structs.Allocation{
			node.ID: []*structs.Allocation{alloc},
		},
	}

	pool := NewEvaluatePool(1, 4)
	defer pool.Shutdown()

	// Push a request
	req := pool.RequestCh()
	req <- evaluateRequest{snap, plan, node.ID}

	// Get the response
	res := <-pool.ResultCh()

	// Verify response
	if res.err != nil {
		t.Fatalf("err: %v", res.err)
	}
	if !res.fit {
		t.Fatalf("bad")
	}
}

func TestEvaluatePool_Resize(t *testing.T) {
	pool := NewEvaluatePool(1, 4)
	defer pool.Shutdown()
	if n := pool.Size(); n != 1 {
		t.Fatalf("bad: %d", n)
	}

	// Scale up
	pool.SetSize(4)
	if n := pool.Size(); n != 4 {
		t.Fatalf("bad: %d", n)
	}

	// Scale down
	pool.SetSize(2)
	if n := pool.Size(); n != 2 {
		t.Fatalf("bad: %d", n)
	}
}
