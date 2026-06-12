// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestGraph_planPhase_mermaid(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("graph"), td)
	t.Chdir(td)

	ui := new(cli.MockUi)
	streams, closeStreams := terminal.StreamsForTesting(t)
	c := &GraphCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(applyFixtureProvider()),
			Ui:               ui,
			Streams:          streams,
		},
	}

	args := []string{"-type=plan", "-format=mermaid"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := closeStreams(t)
	if !strings.Contains(output.Stdout(), "flowchart") && !strings.Contains(output.Stdout(), "-->") {
		t.Fatalf("doesn't look like mermaid graph:\n%s\n\nstderr:\n%s", output.Stdout(), output.Stderr())
	}
}
