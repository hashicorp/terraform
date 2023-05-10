// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/mitchellh/cli"
)

// helpFunc is a cli.HelpFunc that can be used to output the help CLI instructions for Terraform.
func helpFunc(commands map[string]cli.CommandFactory) string {
	// Determine the maximum key length, and classify based on type
	var otherCommands []string
	maxKeyLen := 0

	for key := range commands {
		if _, ok := HiddenCommands[key]; ok {
			// We don't consider hidden commands when deciding the
			// maximum command length.
			continue
		}

		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}

		isOther := true
		for _, candidate := range PrimaryCommands {
			if candidate == key {
				isOther = false
				break
			}
		}
		if isOther {
			otherCommands = append(otherCommands, key)
		}
	}
	sort.Strings(otherCommands)

	// The output produced by this is included in the docs at
	// website/source/docs/cli/commands/index.html.markdown; if you
	// change this then consider updating that to match.
	helpText := fmt.Sprintf(`
Usage: terraform [global options] <subcommand> [args]

The available commands for execution are listed below.
The primary workflow commands are given first, followed by
less common or more advanced commands.

Main commands:
%s
All other commands:
%s
Global options (use these before the subcommand, if any):
  -chdir=DIR    Switch to a different working directory before executing the
                given subcommand.
  -help         Show this help output, or the help for a specified subcommand.
  -version      An alias for the "version" subcommand.
`, listCommands(commands, PrimaryCommands, maxKeyLen), listCommands(commands, otherCommands, maxKeyLen))

	return strings.TrimSpace(helpText)
}

// listCommands just lists the commands in the map with the
// given maximum key length.
func listCommands(allCommands map[string]cli.CommandFactory, order []string, maxKeyLen int) string {
	var buf bytes.Buffer

	for _, key := range order {
		commandFunc, ok := allCommands[key]
		if !ok {
			// This suggests an inconsistency in the command table definitions
			// in commands.go .
			panic("command not found: " + key)
		}

		command, err := commandFunc()
		if err != nil {
			// This would be really weird since there's no good reason for
			// any of our command factories to fail.
			log.Printf("[ERR] cli: Command '%s' failed to load: %s",
				key, err)
			continue
		}

		key = fmt.Sprintf("%s%s", key, strings.Repeat(" ", maxKeyLen-len(key)))
		buf.WriteString(fmt.Sprintf("  %s  %s\n", key, command.Synopsis()))
	}

	return buf.String()
}
