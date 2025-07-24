// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/cli"
)

func TestLockCommand_noArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &LockCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	// Test that the command shows help when given arguments
	code := c.Run([]string{"some-arg"})
	if code != cli.RunResultHelp {
		t.Fatalf("expected help exit code, got: %d", code)
	}

	output := ui.ErrorWriter.String()
	if !strings.Contains(output, "force-lock command does not accept any arguments") {
		t.Fatalf("expected error about arguments, got: %s", output)
	}
}

func TestLockCommand_help(t *testing.T) {
	c := &LockCommand{}
	help := c.Help()

	if !strings.Contains(help, "force-lock") {
		t.Fatalf("expected help to contain 'force-lock', got: %s", help)
	}

	if !strings.Contains(help, "Manually lock the state") {
		t.Fatalf("expected help to contain description, got: %s", help)
	}
}

func TestLockCommand_synopsis(t *testing.T) {
	c := &LockCommand{}
	synopsis := c.Synopsis()

	if synopsis == "" {
		t.Fatal("expected non-empty synopsis")
	}

	if !strings.Contains(synopsis, "lock") {
		t.Fatalf("expected synopsis to contain 'lock', got: %s", synopsis)
	}
}