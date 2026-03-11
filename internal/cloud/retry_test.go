// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fatalError struct{}

var fe = errors.New("this was a fatal error")

func (f fatalError) FatalError() error {
	return fe
}

func (f fatalError) Error() string {
	return f.FatalError().Error()
}

func Test_RetryBackoff_canceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	err := RetryBackoff(ctx, func() error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected canceled error, got %q", err)
	}
}

func Test_RetryBackoff_deadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))

	defer cancel()

	err := RetryBackoff(ctx, func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected timeout error, got %q", err)
	}
}

func Test_RetryBackoff_happy(t *testing.T) {
	t.Parallel()

	err := RetryBackoff(context.Background(), func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected nil err, got %q", err)
	}
}

func Test_RetryBackoff_fatal(t *testing.T) {
	t.Parallel()

	err := RetryBackoff(context.Background(), func() error {
		return fatalError{}
	})

	if !errors.Is(fe, err) {
		t.Errorf("expected fatal error, got %q", err)
	}
}

func Test_RetryBackoff_non_fatal(t *testing.T) {
	t.Parallel()

	var retriedCount = 0

	err := RetryBackoff(context.Background(), func() error {
		retriedCount += 1
		if retriedCount == 2 {
			return nil
		}
		return errors.New("retryable error")
	})

	if err != nil {
		t.Errorf("expected no error, got %q", err)
	}

	if retriedCount != 2 {
		t.Errorf("expected 2 retries, got %d", retriedCount)
	}
}
