// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestMoveResultsCollectorRecordOldAddrCollapsesChains(t *testing.T) {
	collector := newMoveResultsCollector()

	addrA := mustParseAbsResourceInstanceStrForTest(t, "test_object.a")
	addrB := mustParseAbsResourceInstanceStrForTest(t, "test_object.b")
	addrC := mustParseAbsResourceInstanceStrForTest(t, "test_object.c")

	collector.RecordOldAddr(addrA, addrB)
	collector.RecordOldAddr(addrB, addrC)

	got := collector.Results()
	if got.Changes.Len() != 1 {
		t.Fatalf("expected 1 change, got %d", got.Changes.Len())
	}

	change, ok := got.Changes.GetOk(addrC)
	if !ok {
		t.Fatalf("expected final address %s to be recorded", addrC)
	}
	if !change.From.Equal(addrA) {
		t.Fatalf("wrong collapsed source address: got %s, want %s", change.From, addrA)
	}
	if !change.To.Equal(addrC) {
		t.Fatalf("wrong collapsed destination address: got %s, want %s", change.To, addrC)
	}
}

func TestMoveResultsCollectorConcurrentRecordOldAddr(t *testing.T) {
	collector := newMoveResultsCollector()

	var wg sync.WaitGroup
	const n = 32
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			from := mustParseAbsResourceInstanceStrForTest(t, fmt.Sprintf("test_object.from[%d]", i))
			to := mustParseAbsResourceInstanceStrForTest(t, fmt.Sprintf("test_object.to[%d]", i))
			collector.RecordOldAddr(from, to)
		}(i)
	}
	wg.Wait()

	got := collector.Results()
	if got.Changes.Len() != n {
		t.Fatalf("expected %d changes, got %d", n, got.Changes.Len())
	}
}

func mustParseAbsResourceInstanceStrForTest(t *testing.T, str string) addrs.AbsResourceInstance {
	t.Helper()

	addr, diags := addrs.ParseAbsResourceInstanceStr(str)
	if diags.HasErrors() {
		t.Fatalf("invalid resource instance address %q: %s", str, diags.Err())
	}
	return addr
}
