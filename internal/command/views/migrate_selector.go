// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"golang.org/x/term"

	"github.com/hashicorp/terraform/internal/command/migrate"
	"github.com/hashicorp/terraform/internal/terminal"
)

// MigrationChoice represents one selectable migration in the TUI.
type MigrationChoice struct {
	Migration migrate.Migration
	Detail    string // e.g. "4 files, 9 changes"
	Selected  bool
}

// SelectMigrations presents an interactive TUI for choosing which migrations
// to run. Returns the selected migration IDs, or nil if the user cancelled.
func SelectMigrations(streams *terminal.Streams, choices []MigrationChoice) []string {
	if len(choices) == 0 {
		return nil
	}

	fd := int(streams.Stdin.File.Fd())

	// Put terminal in raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: non-interactive
		return nil
	}
	defer term.Restore(fd, oldState)

	cursor := 0
	out := streams.Stdout.File

	render := func() {
		// Move cursor to start and clear
		fmt.Fprint(out, "\033[?25l") // hide cursor

		for i, c := range choices {
			// Move to line
			if i > 0 {
				fmt.Fprint(out, "\r\n")
			}
			fmt.Fprint(out, "\033[2K") // clear line

			checkbox := "[ ]"
			if c.Selected {
				checkbox = "[x]"
			}

			prefix := "  "
			if i == cursor {
				prefix = "> "
			}

			line := fmt.Sprintf("%s%s %s  %s", prefix, checkbox, c.Migration.ID(), c.Detail)
			if i == cursor {
				fmt.Fprint(out, "\033[1m"+line+"\033[0m") // bold
			} else {
				fmt.Fprint(out, line)
			}
		}

		// Print help line
		fmt.Fprint(out, "\r\n\033[2K")
		fmt.Fprint(out, "\033[2m↑/↓ move • space select • a all • enter confirm • q cancel\033[0m")

		// Move cursor back to top
		fmt.Fprintf(out, "\033[%dA", len(choices))
		fmt.Fprint(out, "\r")
	}

	// Initial render with header
	fmt.Fprint(out, "Select migrations to run:\r\n\r\n")
	render()

	buf := make([]byte, 3)
	for {
		n, err := streams.Stdin.File.Read(buf)
		if err != nil || n == 0 {
			break
		}

		switch {
		case n == 1 && (buf[0] == 'q' || buf[0] == 'Q' || buf[0] == 27):
			// q or raw Escape: cancel
			fmt.Fprintf(out, "\033[%dB", len(choices)) // move past list
			fmt.Fprint(out, "\r\n\033[?25h")           // show cursor
			return nil

		case n == 1 && buf[0] == ' ':
			// Space: toggle selection
			choices[cursor].Selected = !choices[cursor].Selected
			render()

		case n == 1 && (buf[0] == 'a' || buf[0] == 'A'):
			// a: toggle all
			allSelected := true
			for _, c := range choices {
				if !c.Selected {
					allSelected = false
					break
				}
			}
			for i := range choices {
				choices[i].Selected = !allSelected
			}
			render()

		case n == 1 && (buf[0] == 13 || buf[0] == 10):
			// Enter: confirm
			fmt.Fprintf(out, "\033[%dB", len(choices)) // move past list
			fmt.Fprint(out, "\r\n\033[?25h")           // show cursor

			var selected []string
			for _, c := range choices {
				if c.Selected {
					selected = append(selected, c.Migration.ID())
				}
			}
			return selected

		case n == 1 && buf[0] == 'k':
			// k: up (vim)
			if cursor > 0 {
				cursor--
			}
			render()

		case n == 1 && buf[0] == 'j':
			// j: down (vim)
			if cursor < len(choices)-1 {
				cursor++
			}
			render()

		case n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 65:
			// Arrow up
			if cursor > 0 {
				cursor--
			}
			render()

		case n == 3 && buf[0] == 27 && buf[1] == 91 && buf[2] == 66:
			// Arrow down
			if cursor < len(choices)-1 {
				cursor++
			}
			render()
		}
	}

	fmt.Fprint(out, "\033[?25h") // show cursor
	return nil
}
