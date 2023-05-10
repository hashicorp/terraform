// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package logging

import (
	"fmt"
	"strings"
	"testing"
)

func TestPanicRecorder(t *testing.T) {
	rec := panics.registerPlugin("test")

	output := []string{
		"panic: test",
		"  stack info",
	}

	for _, line := range output {
		rec(line)
	}

	expected := fmt.Sprintf(pluginPanicOutput, "test", strings.Join(output, "\n"))

	res := PluginPanics()
	if len(res) == 0 {
		t.Fatal("no output")
	}

	if res[0] != expected {
		t.Fatalf("expected: %q\ngot: %q", expected, res[0])
	}
}

func TestPanicLimit(t *testing.T) {
	rec := panics.registerPlugin("test")

	rec("panic: test")

	for i := 0; i < 200; i++ {
		rec(fmt.Sprintf("LINE: %d", i))
	}

	res := PluginPanics()
	// take the extra content into account
	max := strings.Count(pluginPanicOutput, "\n") + panics.maxLines
	for _, out := range res {
		found := strings.Count(out, "\n")
		if found > max {
			t.Fatalf("expected no more than %d lines, got: %d", max, found)
		}
	}
}
