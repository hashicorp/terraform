package nomad

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/testutil"
)

func testBlockedEvals(t *testing.T) (*BlockedEvals, *EvalBroker) {
	broker := testBroker(t, 0)
	broker.SetEnabled(true)
	blocked := NewBlockedEvals(broker)
	blocked.SetEnabled(true)
	return blocked, broker
}

func TestBlockedEvals_Block_Disabled(t *testing.T) {
	blocked, _ := testBlockedEvals(t)
	blocked.SetEnabled(false)

	// Create an escaped eval and add it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.EscapedComputedClass = true
	blocked.Block(e)

	// Verify block did nothing
	bStats := blocked.Stats()
	if bStats.TotalBlocked != 0 || bStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", bStats)
	}
}

func TestBlockedEvals_Block_SameJob(t *testing.T) {
	blocked, _ := testBlockedEvals(t)

	// Create two blocked evals and add them to the blocked tracker.
	e := mock.Eval()
	e2 := mock.Eval()
	e2.JobID = e.JobID
	blocked.Block(e)
	blocked.Block(e2)

	// Verify block did track both
	bStats := blocked.Stats()
	if bStats.TotalBlocked != 1 || bStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", bStats)
	}
}

func TestBlockedEvals_Block_PriorUnblocks(t *testing.T) {
	blocked, _ := testBlockedEvals(t)

	// Do unblocks prior to blocking
	blocked.Unblock("v1:123", 1000)
	blocked.Unblock("v1:123", 1001)

	// Create two blocked evals and add them to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.ClassEligibility = map[string]bool{"v1:123": false, "v1:456": false}
	e.SnapshotIndex = 999
	blocked.Block(e)

	// Verify block did track both
	bStats := blocked.Stats()
	if bStats.TotalBlocked != 1 || bStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", bStats)
	}
}

func TestBlockedEvals_GetDuplicates(t *testing.T) {
	blocked, _ := testBlockedEvals(t)

	// Create duplicate blocked evals and add them to the blocked tracker.
	e := mock.Eval()
	e2 := mock.Eval()
	e2.JobID = e.JobID
	e3 := mock.Eval()
	e3.JobID = e.JobID
	blocked.Block(e)
	blocked.Block(e2)

	// Verify block did track both
	bStats := blocked.Stats()
	if bStats.TotalBlocked != 1 || bStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", bStats)
	}

	// Get the duplicates.
	out := blocked.GetDuplicates(0)
	if len(out) != 1 || !reflect.DeepEqual(out[0], e2) {
		t.Fatalf("bad: %#v %#v", out, e2)
	}

	// Call block again after a small sleep.
	go func() {
		time.Sleep(500 * time.Millisecond)
		blocked.Block(e3)
	}()

	// Get the duplicates.
	out = blocked.GetDuplicates(1 * time.Second)
	if len(out) != 1 || !reflect.DeepEqual(out[0], e3) {
		t.Fatalf("bad: %#v %#v", out, e2)
	}
}

func TestBlockedEvals_UnblockEscaped(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Create an escaped eval and add it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.EscapedComputedClass = true
	blocked.Block(e)

	// Verify block caused the eval to be tracked
	bStats := blocked.Stats()
	if bStats.TotalBlocked != 1 || bStats.TotalEscaped != 1 {
		t.Fatalf("bad: %#v", bStats)
	}

	blocked.Unblock("v1:123", 1000)

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock caused an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		// Verify Unblock updates the stats
		bStats := blocked.Stats()
		if bStats.TotalBlocked != 0 || bStats.TotalEscaped != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestBlockedEvals_UnblockEligible(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Create a blocked eval that is eligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.ClassEligibility = map[string]bool{"v1:123": true}
	blocked.Block(e)

	// Verify block caused the eval to be tracked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 1 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	blocked.Unblock("v1:123", 1000)

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock caused an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		// Verify Unblock updates the stats
		bStats := blocked.Stats()
		if bStats.TotalBlocked != 0 || bStats.TotalEscaped != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestBlockedEvals_UnblockIneligible(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Create a blocked eval that is ineligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.ClassEligibility = map[string]bool{"v1:123": false}
	blocked.Block(e)

	// Verify block caused the eval to be tracked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 1 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	// Should do nothing
	blocked.Unblock("v1:123", 1000)

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock didn't cause an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 0 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		bStats := blocked.Stats()
		if bStats.TotalBlocked != 1 || bStats.TotalEscaped != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestBlockedEvals_UnblockUnknown(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Create a blocked eval that is ineligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.ClassEligibility = map[string]bool{"v1:123": true, "v1:456": false}
	blocked.Block(e)

	// Verify block caused the eval to be tracked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 1 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	// Should unblock because the eval hasn't seen this node class.
	blocked.Unblock("v1:789", 1000)

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock causes an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		// Verify Unblock updates the stats
		bStats := blocked.Stats()
		if bStats.TotalBlocked != 0 || bStats.TotalEscaped != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestBlockedEvals_Reblock(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Create an evaluation, Enqueue/Dequeue it to get a token
	e := mock.Eval()
	e.SnapshotIndex = 500
	e.Status = structs.EvalStatusBlocked
	e.ClassEligibility = map[string]bool{"v1:123": true, "v1:456": false}
	broker.Enqueue(e)

	_, token, err := broker.Dequeue([]string{e.Type}, time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Reblock the evaluation
	blocked.Reblock(e, token)

	// Verify block caused the eval to be tracked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 1 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	// Should unblock because the eval
	blocked.Unblock("v1:123", 1000)

	brokerStats := broker.Stats()
	if brokerStats.TotalReady != 0 && brokerStats.TotalUnacked != 1 {
		t.Fatalf("bad: %#v", brokerStats)
	}

	// Ack the evaluation which should cause the reblocked eval to transistion
	// to ready
	if err := broker.Ack(e.ID, token); err != nil {
		t.Fatalf("err: %v", err)
	}

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock causes an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		// Verify Unblock updates the stats
		bStats := blocked.Stats()
		if bStats.TotalBlocked != 0 || bStats.TotalEscaped != 0 {
			return false, fmt.Errorf("bad: %#v", bStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

// Test the block case in which the eval should be immediately unblocked since
// it is escaped and old
func TestBlockedEvals_Block_ImmediateUnblock_Escaped(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Do an unblock prior to blocking
	blocked.Unblock("v1:123", 1000)

	// Create a blocked eval that is eligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.EscapedComputedClass = true
	e.SnapshotIndex = 900
	blocked.Block(e)

	// Verify block caused the eval to be immediately unblocked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 0 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock caused an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

// Test the block case in which the eval should be immediately unblocked since
// there is an unblock on an unseen class that occurred while it was in the
// scheduler
func TestBlockedEvals_Block_ImmediateUnblock_UnseenClass_After(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Do an unblock prior to blocking
	blocked.Unblock("v1:123", 1000)

	// Create a blocked eval that is eligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.EscapedComputedClass = false
	e.SnapshotIndex = 900
	blocked.Block(e)

	// Verify block caused the eval to be immediately unblocked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 0 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock caused an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

// Test the block case in which the eval should not immediately unblock since
// there is an unblock on an unseen class that occurred before it was in the
// scheduler
func TestBlockedEvals_Block_ImmediateUnblock_UnseenClass_Before(t *testing.T) {
	blocked, _ := testBlockedEvals(t)

	// Do an unblock prior to blocking
	blocked.Unblock("v1:123", 500)

	// Create a blocked eval that is eligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.EscapedComputedClass = false
	e.SnapshotIndex = 900
	blocked.Block(e)

	// Verify block caused the eval to be immediately unblocked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 1 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}
}

// Test the block case in which the eval should be immediately unblocked since
// it a class it is eligible for has been unblocked
func TestBlockedEvals_Block_ImmediateUnblock_SeenClass(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Do an unblock prior to blocking
	blocked.Unblock("v1:123", 1000)

	// Create a blocked eval that is eligible on a specific node class and add
	// it to the blocked tracker.
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.ClassEligibility = map[string]bool{"v1:123": true, "v1:456": false}
	e.SnapshotIndex = 900
	blocked.Block(e)

	// Verify block caused the eval to be immediately unblocked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 0 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock caused an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 1 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}

		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})
}

func TestBlockedEvals_UnblockFailed(t *testing.T) {
	blocked, broker := testBlockedEvals(t)

	// Create blocked evals that are due to failures
	e := mock.Eval()
	e.Status = structs.EvalStatusBlocked
	e.TriggeredBy = structs.EvalTriggerMaxPlans
	e.EscapedComputedClass = true
	blocked.Block(e)

	e2 := mock.Eval()
	e2.Status = structs.EvalStatusBlocked
	e2.TriggeredBy = structs.EvalTriggerMaxPlans
	e2.ClassEligibility = map[string]bool{"v1:123": true, "v1:456": false}
	blocked.Block(e2)

	// Trigger an unblock fail
	blocked.UnblockFailed()

	// Verify UnblockFailed caused the eval to be immediately unblocked
	blockedStats := blocked.Stats()
	if blockedStats.TotalBlocked != 0 && blockedStats.TotalEscaped != 0 {
		t.Fatalf("bad: %#v", blockedStats)
	}

	testutil.WaitForResult(func() (bool, error) {
		// Verify Unblock caused an enqueue
		brokerStats := broker.Stats()
		if brokerStats.TotalReady != 2 {
			return false, fmt.Errorf("bad: %#v", brokerStats)
		}
		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})

	// Reblock an eval for the same job and check that it gets tracked.
	blocked.Block(e)
	blockedStats = blocked.Stats()
	if blockedStats.TotalBlocked != 1 && blockedStats.TotalEscaped != 1 {
		t.Fatalf("bad: %#v", blockedStats)
	}
}
