// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/terminal"
)

func TestNewView_canProducePurpleColor(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewView(streams)

	if v == nil {
		t.Fatal("expected NewView to return a non-nil view")
	}

	if _, ok := v.colorize.Colors["purple"]; !ok {
		t.Fatal("expected colorizer colors to include purple color")
	}

	message := `
                                          [purple]-                                [reset]
                                          [purple]-----                           -[reset]
                                          [purple]---------                      --[reset]
                                          [purple]---------  -                -----[reset]
                                           [purple]---------  ------        -------[reset]
                                             [purple]-------  ---------  ----------[reset]
                                                [purple]----  ---------- ----------[reset]
                                                  [purple]--  ---------- ----------[reset]
   [bold]Welcome to HCP Terraform![reset]                       [purple]-  ---------- -------[reset]
                                                      [purple]---  ----- ---[reset]
   [bold]Documentation:[reset] terraform.io/docs/cloud             [purple]--------   -[reset]
                                                      [purple]----------[reset]
                                                      [purple]----------[reset]
                                                       [purple]---------[reset]
                                                           [purple]-----[reset]
                                                               [purple]-[reset]`

	v.streams.Print(
		v.colorize.Color(message),
	)
	output := done(t)

	t.Logf("output:\n%s", output.Stdout())

	if strings.Contains(output.Stdout(), "[purple]") {
		t.Fatal("expected output to not contain raw [purple] tags")
	}
}
