package communicator

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/terraform/terraform"
)

func TestCommunicator_new(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type": "telnet",
			},
		},
	}
	if _, err := New(r); err == nil {
		t.Fatalf("expected error with telnet")
	}

	r.Ephemeral.ConnInfo["type"] = "ssh"
	if _, err := New(r); err != nil {
		t.Fatalf("err: %v", err)
	}

	r.Ephemeral.ConnInfo["type"] = "winrm"
	if _, err := New(r); err != nil {
		t.Fatalf("err: %v", err)
	}
}
func TestRetryFunc(t *testing.T) {
	origMax := maxBackoffDelay
	maxBackoffDelay = time.Second
	origStart := initialBackoffDelay
	initialBackoffDelay = 10 * time.Millisecond

	defer func() {
		maxBackoffDelay = origMax
		initialBackoffDelay = origStart
	}()

	// succeed on the third try
	errs := []error{io.EOF, &net.OpError{Err: errors.New("ERROR")}, nil}
	count := 0

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Retry(ctx, func() error {
		if count >= len(errs) {
			return errors.New("failed to stop after nil error")
		}

		err := errs[count]
		count++

		return err
	})

	if count != 3 {
		t.Fatal("retry func should have been called 3 times")
	}

	if err != nil {
		t.Fatal(err)
	}
}

func TestRetryFuncBackoff(t *testing.T) {
	origMax := maxBackoffDelay
	maxBackoffDelay = time.Second
	origStart := initialBackoffDelay
	initialBackoffDelay = 100 * time.Millisecond

	defer func() {
		maxBackoffDelay = origMax
		initialBackoffDelay = origStart
	}()

	count := 0

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	Retry(ctx, func() error {
		count++
		return io.EOF
	})

	if count > 4 {
		t.Fatalf("retry func failed to backoff. called %d times", count)
	}
}
