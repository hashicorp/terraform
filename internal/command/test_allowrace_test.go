//go:build !race
// +build !race

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/cli"
	testing_command "github.com/hashicorp/terraform/internal/command/testing"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

// The test contains a data race due to the disabling of the provider lock.
// The provider lock was disabled, so that we can measure the true duration of the
// test operation. Without disabling the provider lock, runs may block each other
// when working with the provider, which does not happen by default in the real-world.
func TestTest_ParallelJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "parallel")), td)
	defer testChdir(t, td)()

	provider := testing_command.NewProvider(&testing_command.ResourceStore{
		Data:   make(map[string]cty.Value),
		Nolock: true,
	})
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{Meta: meta}
	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 9, code, output.All())
	}

	c := &TestCommand{Meta: meta}
	c.Run([]string{"-json", "-no-color"})
	output := done(t).All()

	if !strings.Contains(output, "40 passed, 0 failed") {
		t.Errorf("output didn't produce the right output:\n\n%s", output)
	}

	// Split the log into lines
	lines := strings.Split(output, "\n")

	// Find the start of the teardown and complete timestamps
	// The difference is the approximate duration of the test teardown operation.
	// This test is running in parallel, so we expect the teardown to also run in parallel.
	// We sleep for 3 seconds in the test teardown to simulate a long-running destroy.
	// There are 6 unique state keys in the parallel test, so we expect the teardown to take less than 3*6 (18) seconds.
	var startTimestamp, completeTimestamp string
	for _, line := range lines {
		if strings.Contains(line, `{"path":"parallel.tftest.hcl","progress":"teardown"`) {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				if ts, ok := obj["@timestamp"].(string); ok {
					startTimestamp = ts
				}
			}
		} else if strings.Contains(line, `{"path":"parallel.tftest.hcl","progress":"complete"`) {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				if ts, ok := obj["@timestamp"].(string); ok {
					completeTimestamp = ts
				}
			}
		}
	}

	if startTimestamp == "" || completeTimestamp == "" {
		t.Fatalf("could not find start or complete timestamp in log output")
	}

	startTime, err := time.Parse(time.RFC3339Nano, startTimestamp)
	if err != nil {
		t.Fatalf("failed to parse start timestamp: %v", err)
	}
	completeTime, err := time.Parse(time.RFC3339Nano, completeTimestamp)
	if err != nil {
		t.Fatalf("failed to parse complete timestamp: %v", err)
	}
	dur := completeTime.Sub(startTime)
	if dur > 10*time.Second {
		t.Fatalf("parallel.tftest.hcl duration took too long: %0.2f seconds", dur.Seconds())
	}
}
