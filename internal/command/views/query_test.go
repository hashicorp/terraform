// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/version"
)

func TestNewQuery_Version(t *testing.T) {
	t.Run("json view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		v := NewQuery(arguments.ViewJSON, NewView(streams))

		v.Version()

		want := []map[string]interface{}{
			{
				"@level":    "info",
				"@message":  fmt.Sprintf("Terraform %s", version.String()),
				"@module":   "terraform.ui",
				"type":      "version",
				"terraform": version.String(),
				"ui":        JSON_UI_VERSION,
			},
		}
		testJSONViewOutputEquals(t, done(t).Stdout(), want)
	})

	t.Run("human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		v := NewQuery(arguments.ViewHuman, NewView(streams))

		v.Version()

		got := done(t).Stdout()
		want := ""
		if got != want {
			t.Fatalf("expected output %q, got %q", want, got)
		}
	})
}
