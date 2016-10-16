package nomad

import (
	"testing"
	"time"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
)

var (
	defaultSched = []string{
		structs.JobTypeService,
		structs.JobTypeBatch,
	}
)

func testBroker(t *testing.T, timeout time.Duration) *EvalBroker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	b, err := NewEvalBroker(timeout, 3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return b
}

func TestEvalBroker_Enqueue_Dequeue_Nack_Ack(t *testing.T) {
	b := testBroker(t, 0)

	// Enqueue, but broker is disabled!
	eval := mock.Eval()
	b.Enqueue(eval)

	// Verify nothing was done
	stats := b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	if b.Enabled() {
		t.Fatalf("should not be enabled")
	}

	// Enable the broker, and enqueue
	b.SetEnabled(true)
	b.Enqueue(eval)

	// Double enqueue is a no-op
	b.Enqueue(eval)

	if !b.Enabled() {
		t.Fatalf("should be enabled")
	}

	// Verify enqueue is done
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Ready != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue should work
	out, token, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad : %#v", out)
	}

	tokenOut, ok := b.Outstanding(out.ID)
	if !ok {
		t.Fatalf("should be outstanding")
	}
	if tokenOut != token {
		t.Fatalf("Bad: %#v %#v", token, tokenOut)
	}

	// OutstandingReset should verify the token
	err = b.OutstandingReset("nope", "foo")
	if err != ErrNotOutstanding {
		t.Fatalf("err: %v", err)
	}
	err = b.OutstandingReset(out.ID, "foo")
	if err != ErrTokenMismatch {
		t.Fatalf("err: %v", err)
	}
	err = b.OutstandingReset(out.ID, tokenOut)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Ready != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Unacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Nack with wrong token should fail
	err = b.Nack(eval.ID, "foobarbaz")
	if err == nil {
		t.Fatalf("should fail to nack")
	}

	// Nack back into the queue
	err = b.Nack(eval.ID, token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if _, ok := b.Outstanding(out.ID); ok {
		t.Fatalf("should not be outstanding")
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Ready != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Unacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue should work again
	out2, token2, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out2 != eval {
		t.Fatalf("bad : %#v", out2)
	}
	if token2 == token {
		t.Fatalf("should get a new token")
	}

	tokenOut2, ok := b.Outstanding(out.ID)
	if !ok {
		t.Fatalf("should be outstanding")
	}
	if tokenOut2 != token2 {
		t.Fatalf("Bad: %#v %#v", token2, tokenOut2)
	}

	// Ack with wrong token
	err = b.Ack(eval.ID, "zip")
	if err == nil {
		t.Fatalf("should fail to ack")
	}

	// Ack finally
	err = b.Ack(eval.ID, token2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if _, ok := b.Outstanding(out.ID); ok {
		t.Fatalf("should not be outstanding")
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Ready != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[eval.Type].Unacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
}

func TestEvalBroker_Serialize_DuplicateJobID(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	eval := mock.Eval()
	b.Enqueue(eval)

	eval2 := mock.Eval()
	eval2.JobID = eval.JobID
	eval2.CreateIndex = eval.CreateIndex + 1
	b.Enqueue(eval2)

	eval3 := mock.Eval()
	eval3.JobID = eval.JobID
	eval3.CreateIndex = eval.CreateIndex + 2
	b.Enqueue(eval3)

	stats := b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 2 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue should work
	out, token, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad : %#v", out)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 2 {
		t.Fatalf("bad: %#v", stats)
	}

	// Ack out
	err = b.Ack(eval.ID, token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue should work
	out, token, err = b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval2 {
		t.Fatalf("bad : %#v", out)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Ack out
	err = b.Ack(eval2.ID, token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue should work
	out, token, err = b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval3 {
		t.Fatalf("bad : %#v", out)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Ack out
	err = b.Ack(eval3.ID, token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalBlocked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
}

func TestEvalBroker_Enqueue_Disable(t *testing.T) {
	b := testBroker(t, 0)

	// Enqueue
	eval := mock.Eval()
	b.SetEnabled(true)
	b.Enqueue(eval)

	// Flush via SetEnabled
	b.SetEnabled(false)

	// Check the stats
	stats := b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if _, ok := stats.ByScheduler[eval.Type]; ok {
		t.Fatalf("bad: %#v", stats)
	}
}

func TestEvalBroker_Dequeue_Timeout(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	start := time.Now()
	out, _, err := b.Dequeue(defaultSched, 5*time.Millisecond)
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

func TestEvalBroker_Dequeue_Empty_Timeout(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)
	doneCh := make(chan struct{}, 1)

	go func() {
		out, _, err := b.Dequeue(defaultSched, 0)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out == nil {
			t.Fatal("Expect an eval")
		}
		doneCh <- struct{}{}
	}()

	// Sleep for a little bit
	select {
	case <-time.After(5 * time.Millisecond):
	case <-doneCh:
		t.Fatalf("Dequeue(0) should block")
	}

	// Enqueue to unblock the dequeue.
	eval := mock.Eval()
	b.Enqueue(eval)

	select {
	case <-doneCh:
		return
	case <-time.After(5 * time.Millisecond):
		t.Fatal("timeout: Dequeue(0) should return after enqueue")
	}
}

// Ensure higher priority dequeued first
func TestEvalBroker_Dequeue_Priority(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	eval1 := mock.Eval()
	eval1.Priority = 10
	b.Enqueue(eval1)

	eval2 := mock.Eval()
	eval2.Priority = 30
	b.Enqueue(eval2)

	eval3 := mock.Eval()
	eval3.Priority = 20
	b.Enqueue(eval3)

	out1, _, _ := b.Dequeue(defaultSched, time.Second)
	if out1 != eval2 {
		t.Fatalf("bad: %#v", out1)
	}

	out2, _, _ := b.Dequeue(defaultSched, time.Second)
	if out2 != eval3 {
		t.Fatalf("bad: %#v", out2)
	}

	out3, _, _ := b.Dequeue(defaultSched, time.Second)
	if out3 != eval1 {
		t.Fatalf("bad: %#v", out3)
	}
}

// Ensure FIFO at fixed priority
func TestEvalBroker_Dequeue_FIFO(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)
	NUM := 100

	for i := 0; i < NUM; i++ {
		eval1 := mock.Eval()
		eval1.CreateIndex = uint64(i)
		eval1.ModifyIndex = uint64(i)
		b.Enqueue(eval1)
	}

	for i := 0; i < NUM; i++ {
		out1, _, _ := b.Dequeue(defaultSched, time.Second)
		if out1.CreateIndex != uint64(i) {
			t.Fatalf("bad: %d %#v", i, out1)
		}
	}
}

// Ensure fairness between schedulers
func TestEvalBroker_Dequeue_Fairness(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)
	NUM := 100

	for i := 0; i < NUM; i++ {
		eval1 := mock.Eval()
		if i < (NUM / 2) {
			eval1.Type = structs.JobTypeService
		} else {
			eval1.Type = structs.JobTypeBatch
		}
		b.Enqueue(eval1)
	}

	counter := 0
	for i := 0; i < NUM; i++ {
		out1, _, _ := b.Dequeue(defaultSched, time.Second)

		switch out1.Type {
		case structs.JobTypeService:
			if counter < 0 {
				counter = 0
			}
			counter += 1
		case structs.JobTypeBatch:
			if counter > 0 {
				counter = 0
			}
			counter -= 1
		}

		// This will fail randomly at times. It is very hard to
		// test deterministically that its acting randomly.
		if counter >= 25 || counter <= -25 {
			t.Fatalf("unlikely sequence: %d", counter)
		}
	}
}

// Ensure we get unblocked
func TestEvalBroker_Dequeue_Blocked(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	// Start with a blocked dequeue
	outCh := make(chan *structs.Evaluation, 1)
	go func() {
		start := time.Now()
		out, _, err := b.Dequeue(defaultSched, time.Second)
		end := time.Now()
		outCh <- out
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if d := end.Sub(start); d < 5*time.Millisecond {
			t.Fatalf("bad: %v", d)
		}
	}()

	// Wait for a bit
	time.Sleep(5 * time.Millisecond)

	// Enqueue
	eval := mock.Eval()
	b.Enqueue(eval)

	// Ensure dequeue
	select {
	case out := <-outCh:
		if out != eval {
			t.Fatalf("bad: %v", out)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

// Ensure we nack in a timely manner
func TestEvalBroker_Nack_Timeout(t *testing.T) {
	b := testBroker(t, 5*time.Millisecond)
	b.SetEnabled(true)

	// Enqueue
	eval := mock.Eval()
	b.Enqueue(eval)

	// Dequeue
	out, _, err := b.Dequeue(defaultSched, time.Second)
	start := time.Now()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad: %v", out)
	}

	// Dequeue, should block on Nack timer
	out, _, err = b.Dequeue(defaultSched, time.Second)
	end := time.Now()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad: %v", out)
	}

	// Check the nack timer
	if diff := end.Sub(start); diff < 5*time.Millisecond {
		t.Fatalf("bad: %#v", diff)
	}
}

// Ensure we nack in a timely manner
func TestEvalBroker_Nack_TimeoutReset(t *testing.T) {
	b := testBroker(t, 5*time.Millisecond)
	b.SetEnabled(true)

	// Enqueue
	eval := mock.Eval()
	b.Enqueue(eval)

	// Dequeue
	out, token, err := b.Dequeue(defaultSched, time.Second)
	start := time.Now()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad: %v", out)
	}

	// Reset in 2 milliseconds
	time.Sleep(2 * time.Millisecond)
	if err := b.OutstandingReset(out.ID, token); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Dequeue, should block on Nack timer
	out, _, err = b.Dequeue(defaultSched, time.Second)
	end := time.Now()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad: %v", out)
	}

	// Check the nack timer
	if diff := end.Sub(start); diff < 7*time.Millisecond {
		t.Fatalf("bad: %#v", diff)
	}
}

func TestEvalBroker_PauseResumeNackTimeout(t *testing.T) {
	b := testBroker(t, 5*time.Millisecond)
	b.SetEnabled(true)

	// Enqueue
	eval := mock.Eval()
	b.Enqueue(eval)

	// Dequeue
	out, token, err := b.Dequeue(defaultSched, time.Second)
	start := time.Now()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad: %v", out)
	}

	// Pause in 2 milliseconds
	time.Sleep(2 * time.Millisecond)
	if err := b.PauseNackTimeout(out.ID, token); err != nil {
		t.Fatalf("err: %v", err)
	}

	go func() {
		time.Sleep(2 * time.Millisecond)
		if err := b.ResumeNackTimeout(out.ID, token); err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	// Dequeue, should block until the timer is resumed
	out, _, err = b.Dequeue(defaultSched, time.Second)
	end := time.Now()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad: %v", out)
	}

	// Check the nack timer
	if diff := end.Sub(start); diff < 9*time.Millisecond {
		t.Fatalf("bad: %#v", diff)
	}
}

func TestEvalBroker_DeliveryLimit(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	eval := mock.Eval()
	b.Enqueue(eval)

	for i := 0; i < 3; i++ {
		// Dequeue should work
		out, token, err := b.Dequeue(defaultSched, time.Second)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out != eval {
			t.Fatalf("bad : %#v", out)
		}

		// Nack with wrong token should fail
		err = b.Nack(eval.ID, token)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	// Check the stats
	stats := b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[failedQueue].Ready != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[failedQueue].Unacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue from failed queue
	out, token, err := b.Dequeue([]string{failedQueue}, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad : %#v", out)
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[failedQueue].Ready != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[failedQueue].Unacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Ack finally
	err = b.Ack(out.ID, token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if _, ok := b.Outstanding(out.ID); ok {
		t.Fatalf("should not be outstanding")
	}

	// Check the stats
	stats = b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.ByScheduler[failedQueue].Ready != 0 {
		t.Fatalf("bad: %#v", stats.ByScheduler[failedQueue])
	}
	if stats.ByScheduler[failedQueue].Unacked != 0 {
		t.Fatalf("bad: %#v", stats.ByScheduler[failedQueue])
	}
}

func TestEvalBroker_AckAtDeliveryLimit(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	eval := mock.Eval()
	b.Enqueue(eval)

	for i := 0; i < 3; i++ {
		// Dequeue should work
		out, token, err := b.Dequeue(defaultSched, time.Second)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if out != eval {
			t.Fatalf("bad : %#v", out)
		}

		if i == 2 {
			b.Ack(eval.ID, token)
		} else {
			// Nack with wrong token should fail
			err = b.Nack(eval.ID, token)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
		}
	}

	// Check the stats
	stats := b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if _, ok := stats.ByScheduler[failedQueue]; ok {
		t.Fatalf("bad: %#v", stats)
	}
}

// Ensure fairness between schedulers
func TestEvalBroker_Wait(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	// Create an eval that should wait
	eval := mock.Eval()
	eval.Wait = 10 * time.Millisecond
	b.Enqueue(eval)

	// Verify waiting
	stats := b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalWaiting != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Let the wait elapse
	time.Sleep(15 * time.Millisecond)

	// Verify ready
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalWaiting != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Dequeue should work
	out, _, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad : %#v", out)
	}
}

// Ensure that priority is taken into account when enqueueing many evaluations.
func TestEvalBroker_EnqueueAll_Dequeue_Fair(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	// Start with a blocked dequeue
	outCh := make(chan *structs.Evaluation, 1)
	go func() {
		start := time.Now()
		out, _, err := b.Dequeue(defaultSched, time.Second)
		end := time.Now()
		outCh <- out
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if d := end.Sub(start); d < 5*time.Millisecond {
			t.Fatalf("bad: %v", d)
		}
	}()

	// Wait for a bit
	time.Sleep(5 * time.Millisecond)

	// Enqueue
	evals := make(map[*structs.Evaluation]string, 8)
	expectedPriority := 90
	for i := 10; i <= expectedPriority; i += 10 {
		eval := mock.Eval()
		eval.Priority = i
		evals[eval] = ""

	}
	b.EnqueueAll(evals)

	// Ensure dequeue
	select {
	case out := <-outCh:
		if out.Priority != expectedPriority {
			t.Fatalf("bad: %v", out)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}

func TestEvalBroker_EnqueueAll_Requeue_Ack(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	// Create the evaluation, enqueue and dequeue
	eval := mock.Eval()
	b.Enqueue(eval)

	out, token, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad : %#v", out)
	}

	// Requeue the same evaluation.
	b.EnqueueAll(map[*structs.Evaluation]string{eval: token})

	// The stats should show one unacked
	stats := b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Ack the evaluation.
	if err := b.Ack(eval.ID, token); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check stats again as this should cause the re-enqueued one to transition
	// into the ready state
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}

	// Another dequeue should be successful
	out2, token2, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out2 != eval {
		t.Fatalf("bad : %#v", out)
	}
	if token == token2 {
		t.Fatalf("bad : %s and %s", token, token2)
	}
}

func TestEvalBroker_EnqueueAll_Requeue_Nack(t *testing.T) {
	b := testBroker(t, 0)
	b.SetEnabled(true)

	// Create the evaluation, enqueue and dequeue
	eval := mock.Eval()
	b.Enqueue(eval)

	out, token, err := b.Dequeue(defaultSched, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != eval {
		t.Fatalf("bad : %#v", out)
	}

	// Requeue the same evaluation.
	b.EnqueueAll(map[*structs.Evaluation]string{eval: token})

	// The stats should show one unacked
	stats := b.Stats()
	if stats.TotalReady != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", stats)
	}

	// Nack the evaluation.
	if err := b.Nack(eval.ID, token); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check stats again as this should cause the re-enqueued one to be dropped
	stats = b.Stats()
	if stats.TotalReady != 1 {
		t.Fatalf("bad: %#v", stats)
	}
	if stats.TotalUnacked != 0 {
		t.Fatalf("bad: %#v", stats)
	}
	if len(b.requeue) != 0 {
		t.Fatalf("bad: %#v", b.requeue)
	}
}
