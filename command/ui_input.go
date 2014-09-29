package command

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

var defaultInputReader io.Reader
var defaultInputWriter io.Writer

// UIInput is an implementation of terraform.UIInput that asks the CLI
// for input stdin.
type UIInput struct {
	// Colorize will color the output.
	Colorize *colorstring.Colorize

	// Reader and Writer for IO. If these aren't set, they will default to
	// Stdout and Stderr respectively.
	Reader io.Reader
	Writer io.Writer

	interrupted bool
	l           sync.Mutex
	once        sync.Once
}

func (i *UIInput) Input(opts *terraform.InputOpts) (string, error) {
	i.once.Do(i.init)

	r := i.Reader
	w := i.Writer
	if r == nil {
		r = defaultInputReader
	}
	if w == nil {
		w = defaultInputWriter
	}
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

	// Build the output format for asking
	var buf bytes.Buffer
	buf.WriteString("[reset]")
	buf.WriteString(fmt.Sprintf("[bold]%s[reset]\n", opts.Query))
	if opts.Description != "" {
		s := bufio.NewScanner(strings.NewReader(opts.Description))
		for s.Scan() {
			buf.WriteString(fmt.Sprintf("  %s\n", s.Text()))
		}
		buf.WriteString("\n")
	}
	if opts.Default != "" {
		buf.WriteString("  [bold]Default:[reset] ")
		buf.WriteString(opts.Default)
		buf.WriteString("\n")
	}
	buf.WriteString("  [bold]Enter a value:[reset] ")

	// Ask the user for their input
	if _, err := fmt.Fprint(w, i.Colorize.Color(buf.String())); err != nil {
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
		fmt.Fprint(w, "\n")

		if line == "" {
			line = opts.Default
		}

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

func (i *UIInput) init() {
	if i.Colorize == nil {
		i.Colorize = &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
		}
	}
}
