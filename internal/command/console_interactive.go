// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build !solaris
// +build !solaris

// The readline library we use doesn't currently support solaris so
// we just build tag it off.

package command

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/hashicorp/cli"

	"github.com/hashicorp/terraform/internal/repl"
)

func (c *ConsoleCommand) modeInteractive(session *repl.Session, ui cli.Ui) int {
	// Configure input
	l, err := readline.NewEx(&readline.Config{
		Prompt:            "> ",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		Stdin:             os.Stdin,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error initializing console: %s",
			err))
		return 1
	}
	defer l.Close()

	// TODO: Currently we're handling multi-line input largely _in spite of_
	// the readline library, because it doesn't support that. This means that
	// in particular the history treats each line as a separate history entry,
	// and doesn't allow editing of previous lines after the user's already
	// pressed enter.
	//
	// Hopefully we can do better than this one day, but having some basic
	// support for multi-line input is at least better than none at all:
	// this is mainly helpful when pasting in expressions from elsewhere that
	// already have newline characters in them, to avoid pre-editing it.

	lines := make([]string, 0, 4)
	for {
		// Read a line
		if len(lines) == 0 {
			l.SetPrompt("> ")
		} else {
			l.SetPrompt(": ")
		}
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(lines) == 0 && line == "" {
				break
			} else if line != "" {
				continue
			} else {
				// Reset the entry buffer to start a new expression
				lines = lines[:0]
				ui.Output("(multi-line entry canceled)")
				continue
			}
		} else if err == io.EOF {
			break
		}
		lines = append(lines, line)
		// The following implements a heuristic for deciding if it seems likely
		// that the user was intending to continue entering more expression
		// characters on a subsequent line. This should get the right answer
		// for any valid expression, but might get confused by invalid input.
		// The user can always hit enter one more time (entering a blank line)
		// to break out of a multi-line sequence and force interpretation of
		// what was already entered.
		if repl.ExpressionEntryCouldContinue(lines) {
			continue
		}

		input := strings.Join(lines, "\n") + "\n"
		lines = lines[:0] // reset for next iteration
		out, exit, diags := session.Handle(input)
		if diags.HasErrors() {
			c.showDiagnostics(diags)
		}
		if exit {
			break
		}

		ui.Output(out)
	}

	return 0
}
