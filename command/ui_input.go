package command

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/hashicorp/terraform/terraform"
)

// UIInput is an implementation of terraform.UIInput that asks the CLI
// for input stdin.
type UIInput struct {
	// Reader and Writer for IO. If these aren't set, they will default to
	// Stdout and Stderr respectively.
	Reader io.Reader
	Writer io.Writer

	interrupted bool
	l           sync.Mutex
}

func (i *UIInput) Input(opts *terraform.InputOpts) (string, error) {
	r := i.Reader
	w := i.Writer
	if r == nil {
		r = os.Stdin
	}
	if w == nil {
		w = os.Stdout
	}

	// Make sure we only ask for input once at a time. Terraform
	// should enforce this, but it doesn't hurt to verify.
	i.l.Lock()
	defer i.l.Unlock()

	// If we're interrupted, then don't ask for input
	if i.interrupted {
		return "", errors.New("interrupted")
	}

	// Listen for interrupts so we can cancel the input ask
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	// Ask the user for their input
	if _, err := fmt.Fprint(w, opts.Query); err != nil {
		return "", err
	}

	// Listen for the input in a goroutine. This will allow us to
	// interrupt this if we are interrupted (SIGINT)
	result := make(chan string, 1)
	go func() {
		var line string
		if _, err := fmt.Fscanln(r, &line); err != nil {
			log.Printf("[ERR] UIInput scan err: %s", err)
		}

		result <- line
	}()

	select {
	case line := <-result:
		return line, nil
	case <-sigCh:
		// Print a newline so that any further output starts properly
		// on a new line.
		fmt.Fprintln(w)

		// Mark that we were interrupted so future Ask calls fail.
		i.interrupted = true

		return "", errors.New("interrupted")
	}
}
