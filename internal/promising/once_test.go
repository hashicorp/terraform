// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package promising_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform/internal/promising"
)

func TestOnce(t *testing.T) {
	type FakeResult struct {
		msg string
	}

	var o promising.Once[*FakeResult]

	ctx := context.Background()
	results := make([]*FakeResult, 5)
	var callCount atomic.Int64
	for i := range results {
		// The "Once" mechanism expects to be run inside a task so that
		// it can create promises and detect self-dependency problems.
		result, err := promising.MainTask(ctx, func(ctx context.Context) (*FakeResult, error) {
			return o.Do(ctx, func(ctx context.Context) (*FakeResult, error) {
				callCount.Add(1)
				return &FakeResult{
					msg: "hello",
				}, nil
			})
		})
		if err != nil {
			t.Fatal(err)
		}
		results[i] = result
	}

	if got, want := callCount.Load(), int64(1); got != want {
		t.Errorf("incorrect call count %d; want %d", got, want)
	}

	gotPtr := results[0]
	if gotPtr == nil {
		t.Fatal("first result is nil; want non-nil pointer")
	}
	if got, want := gotPtr.msg, "hello"; got != want {
		t.Fatalf("wrong message %q; want %q", got, want)
	}

	// Because of the coalescing effect of Once, all of the results should
	// point to the same FakeResult object.
	for i, result := range results {
		if result != gotPtr {
			t.Errorf("result %d does not match result 0; all results should be identical", i)
		}
	}
}
