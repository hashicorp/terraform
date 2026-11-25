// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/terraform"
)

func TestUIInput_impl(t *testing.T) {
	t.Parallel()
	var _ terraform.UIInput = new(uIInput)
	var _ InputRequester = new(uIInput)
	var _ InputRequesterForTest = new(uIInput)
}

func TestNewUIInput_Input(t *testing.T) {
	t.Parallel()
	i := NewUIInput(
		UIInputOptions{},
	)
	j, ok := i.(*uIInput)
	if !ok {
		t.Fatalf("test should be testing a *uIInput")
	}
	j.Reader = bytes.NewBufferString("foo\n")

	v, err := i.Input(context.Background(), &terraform.InputOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != "foo" {
		t.Fatalf("unexpected input: %s", v)
	}
}

// When writing an integration test, the test can define data supplied in response
// to input prompts as a map of query id's to answers.
func TestNewUIInputForTest_testInputResponseMap(t *testing.T) {
	t.Parallel()

	testInputResponseMap := map[string]string{"prompt-number-1": "foo"}
	i := NewUIInputForTests(
		t,
		UIInputOptions{},
		testInputResponseMap,
	)

	v, err := i.Input(context.Background(), &terraform.InputOpts{
		Id: "prompt-number-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != "foo" {
		t.Fatalf("unexpected input: %s", v)
	}
}

func TestUIInputInput_canceled(t *testing.T) {
	t.Parallel()
	r, w := io.Pipe()
	i := NewUIInput(
		UIInputOptions{},
	)
	j, ok := i.(*uIInput)
	if !ok {
		t.Fatalf("test should be testing a *uIInput")
	}
	j.Reader = r

	// Make a context that can be canceled.
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Cancel the context after 2 seconds.
		time.Sleep(2 * time.Second)
		cancel()
	}()

	// Get input until the context is canceled.
	v, err := i.Input(ctx, &terraform.InputOpts{})
	if err != context.Canceled {
		t.Fatalf("expected a context.Canceled error, got: %v", err)
	}

	// As the context was canceled v should be empty.
	if v != "" {
		t.Fatalf("unexpected input: %s", v)
	}

	// As the context was canceled we should still be listening.
	listening := atomic.LoadInt32(&j.listening)
	if listening != 1 {
		t.Fatalf("expected listening to be 1, got: %d", listening)
	}

	go func() {
		// Fake input is given after 1 second.
		time.Sleep(time.Second)
		fmt.Fprint(w, "foo\n")
		w.Close()
	}()

	v, err = i.Input(context.Background(), &terraform.InputOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != "foo" {
		t.Fatalf("unexpected input: %s", v)
	}
}

func TestUIInputInput_spaces(t *testing.T) {
	t.Parallel()
	i := NewUIInput(UIInputOptions{})
	j, ok := i.(*uIInput)
	if !ok {
		t.Fatalf("test should be testing a *uIInput")
	}
	j.Reader = bytes.NewBufferString("foo bar\n")

	v, err := j.Input(context.Background(), &terraform.InputOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != "foo bar" {
		t.Fatalf("unexpected input: %s", v)
	}
}

func TestUIInputInput_Error(t *testing.T) {
	t.Parallel()

	i := NewUIInput(UIInputOptions{})

	v, err := i.Input(context.Background(), &terraform.InputOpts{})
	if err == nil {
		t.Fatalf("Error is not 'nil'")
	}

	if err.Error() != "EOF" {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != "" {
		t.Fatalf("input must be empty")
	}
}
