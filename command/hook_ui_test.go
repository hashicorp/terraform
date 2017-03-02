package command

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

func TestUiHookPostApply_emptyState(t *testing.T) {
	colorize := &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
		Reset:   true,
	}

	ir := bytes.NewReader([]byte{})
	errBuf := bytes.NewBuffer([]byte{})
	outBuf := bytes.NewBuffer([]byte{})
	ui := cli.MockUi{
		InputReader:  ir,
		ErrorWriter:  errBuf,
		OutputWriter: outBuf,
	}
	h := &UiHook{
		Colorize: colorize,
		Ui:       &ui,
	}
	h.init()
	h.resources = map[string]uiResourceState{
		"data.google_compute_zones.available": uiResourceState{
			Op:    uiResourceDestroy,
			Start: time.Now(),
		},
	}

	mock := &terraform.MockInstanceInfo{
		terraform.InstanceInfo{
			Id:         "data.google_compute_zones.available",
			ModulePath: []string{"root"},
			Type:       "google_compute_zones",
		},
	}
	n := mock.WithUniqueExtra("destroy")
	action, err := h.PostApply(n, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	expectedOutput := ""
	output := outBuf.String()
	if output != expectedOutput {
		t.Fatalf("Output didn't match.\nExpected: %q\nGiven: %q", expectedOutput, output)
	}

	expectedErrOutput := ""
	errOutput := errBuf.String()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

func TestTruncateId(t *testing.T) {
	testCases := []struct {
		Input    string
		Expected string
		MaxLen   int
	}{
		{
			Input:    "Hello world",
			Expected: "H...d",
			MaxLen:   3,
		},
		{
			Input:    "Hello world",
			Expected: "H...d",
			MaxLen:   5,
		},
		{
			Input:    "Hello world",
			Expected: "He...d",
			MaxLen:   6,
		},
		{
			Input:    "Hello world",
			Expected: "He...ld",
			MaxLen:   7,
		},
		{
			Input:    "Hello world",
			Expected: "Hel...ld",
			MaxLen:   8,
		},
		{
			Input:    "Hello world",
			Expected: "Hel...rld",
			MaxLen:   9,
		},
		{
			Input:    "Hello world",
			Expected: "Hell...rld",
			MaxLen:   10,
		},
		{
			Input:    "Hello world",
			Expected: "Hello world",
			MaxLen:   11,
		},
		{
			Input:    "Hello world",
			Expected: "Hello world",
			MaxLen:   12,
		},
	}
	for i, tc := range testCases {
		testName := fmt.Sprintf("%d", i)
		t.Run(testName, func(t *testing.T) {
			out := truncateId(tc.Input, tc.MaxLen)
			if out != tc.Expected {
				t.Fatalf("Expected %q to be shortened to %d as %q (given: %q)",
					tc.Input, tc.MaxLen, tc.Expected, out)
			}
		})
	}
}
