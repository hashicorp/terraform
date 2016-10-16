package nomad

import (
	"testing"
	"time"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
)

func testPlanQueue(t *testing.T) *PlanQueue {
	pq, err := NewPlanQueue()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return pq
}

func TestPlanQueue_Enqueue_Dequeue(t *testing.T) {
	pq := testPlanQueue(t)
	if pq.Enabled() {
		t.Fatalf("should not be enabled")
	}
	pq.SetEnabled(true)
	if !pq.Enabled() {
		t.Fatalf("should be enabled")
	}

	plan := mock.Plan()
	future, err := pq.Enqueue(plan)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	stats := pq.Stats()
	if stats.Depth != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	resCh := make(chan *structs.PlanResult, 1)
	go func() {
		res, err := future.Wait()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		resCh <- res
	}()

	pending, err := pq.Dequeue(time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	stats = pq.Stats()
	if stats.Depth != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	if pending == nil || pending.plan != plan {
		t.Fatalf("bad: %#v", pending)
	}

	result := mock.PlanResult()
	pending.respond(result, nil)

	select {
	case r := <-resCh:
		if r != result {
			t.Fatalf("Bad: %#v", r)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestPlanQueue_Enqueue_Disable(t *testing.T) {
	pq := testPlanQueue(t)

	// Enqueue
	plan := mock.Plan()
	pq.SetEnabled(true)
	future, err := pq.Enqueue(plan)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Flush via SetEnabled
	pq.SetEnabled(false)

	// Check the stats
	stats := pq.Stats()
	if stats.Depth != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Future should be canceled
	res, err := future.Wait()
	if err != planQueueFlushed {
		t.Fatalf("err: %v", err)
	}
	if res != nil {
		t.Fatalf("bad: %#v", res)
	}
}

func TestPlanQueue_Dequeue_Timeout(t *testing.T) {
	pq := testPlanQueue(t)
	pq.SetEnabled(true)

	start := time.Now()
	out, err := pq.Dequeue(5 * time.Millisecond)
	end := time.Now()

	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != nil {
		t.Fatalf("unexpected: %#v", out)
	}

	if diff := end.Sub(start); diff < 5*time.Millisecond {
		t.Fatalf("bad: %#v", diff)
	}
}

// Ensure higher priority dequeued first
func TestPlanQueue_Dequeue_Priority(t *testing.T) {
	pq := testPlanQueue(t)
	pq.SetEnabled(true)

	plan1 := mock.Plan()
	plan1.Priority = 10
	pq.Enqueue(plan1)

	plan2 := mock.Plan()
	plan2.Priority = 30
	pq.Enqueue(plan2)

	plan3 := mock.Plan()
	plan3.Priority = 20
	pq.Enqueue(plan3)

	out1, _ := pq.Dequeue(time.Second)
	if out1.plan != plan2 {
		t.Fatalf("bad: %#v", out1)
	}

	out2, _ := pq.Dequeue(time.Second)
	if out2.plan != plan3 {
		t.Fatalf("bad: %#v", out2)
	}

	out3, _ := pq.Dequeue(time.Second)
	if out3.plan != plan1 {
		t.Fatalf("bad: %#v", out3)
	}
}

// Ensure FIFO at fixed priority
func TestPlanQueue_Dequeue_FIFO(t *testing.T) {
	pq := testPlanQueue(t)
	pq.SetEnabled(true)

	plans := make([]*structs.Plan, 100)
	for i := 0; i < len(plans); i++ {
		if i%5 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		plans[i] = mock.Plan()
		pq.Enqueue(plans[i])
	}

	var prev *pendingPlan
	for i := range plans {
		out, err := pq.Dequeue(time.Second)
		if err != nil {
			t.Fatalf("failed to dequeue plan %d: %v", i, err)
		}
		if prev != nil && out.enqueueTime.Before(prev.enqueueTime) {
			t.Fatalf("out of order dequeue at %d, prev=%v, got=%v", i, prev.enqueueTime, out.enqueueTime)
		}
		prev = out
	}
}
