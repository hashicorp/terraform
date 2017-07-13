package command

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

func TestUiHookPreApply_periodicTimer(t *testing.T) {
	ui := cli.NewMockUi()
	h := &UiHook{
		Colorize: &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
			Reset:   true,
		},
		Ui:              ui,
		PeriodicUiTimer: 1 * time.Second,
	}
	h.init()
	h.resources = map[string]uiResourceState{
		"data.aws_availability_zones.available": uiResourceState{
			Op:    uiResourceDestroy,
			Start: time.Now(),
		},
	}

	n := &terraform.InstanceInfo{
		Id:         "data.aws_availability_zones.available",
		ModulePath: []string{"root"},
		Type:       "aws_availability_zones",
	}

	s := &terraform.InstanceState{
		ID: "2017-03-05 10:56:59.298784526 +0000 UTC",
		Attributes: map[string]string{
			"id":      "2017-03-05 10:56:59.298784526 +0000 UTC",
			"names.#": "4",
			"names.0": "us-east-1a",
			"names.1": "us-east-1b",
			"names.2": "us-east-1c",
			"names.3": "us-east-1d",
		},
	}
	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	action, err := h.PreApply(n, s, d)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	time.Sleep(3100 * time.Millisecond)

	// stop the background writer
	uiState := h.resources[n.HumanId()]
	close(uiState.DoneCh)
	<-uiState.done

	expectedOutput := `data.aws_availability_zones.available: Destroying... (ID: 2017-03-05 10:56:59.298784526 +0000 UTC)
data.aws_availability_zones.available: Still destroying... (ID: 2017-03-05 10:56:59.298784526 +0000 UTC, 1s elapsed)
data.aws_availability_zones.available: Still destroying... (ID: 2017-03-05 10:56:59.298784526 +0000 UTC, 2s elapsed)
data.aws_availability_zones.available: Still destroying... (ID: 2017-03-05 10:56:59.298784526 +0000 UTC, 3s elapsed)
`
	output := ui.OutputWriter.String()
	if output != expectedOutput {
		t.Fatalf("Output didn't match.\nExpected: %q\nGiven: %q", expectedOutput, output)
	}

	expectedErrOutput := ""
	errOutput := ui.ErrorWriter.String()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

func TestUiHookPreApply_destroy(t *testing.T) {
	ui := cli.NewMockUi()
	h := &UiHook{
		Colorize: &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
			Reset:   true,
		},
		Ui: ui,
	}
	h.init()
	h.resources = map[string]uiResourceState{
		"data.aws_availability_zones.available": uiResourceState{
			Op:    uiResourceDestroy,
			Start: time.Now(),
		},
	}

	n := &terraform.InstanceInfo{
		Id:         "data.aws_availability_zones.available",
		ModulePath: []string{"root"},
		Type:       "aws_availability_zones",
	}

	s := &terraform.InstanceState{
		ID: "2017-03-05 10:56:59.298784526 +0000 UTC",
		Attributes: map[string]string{
			"id":      "2017-03-05 10:56:59.298784526 +0000 UTC",
			"names.#": "4",
			"names.0": "us-east-1a",
			"names.1": "us-east-1b",
			"names.2": "us-east-1c",
			"names.3": "us-east-1d",
		},
	}
	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	action, err := h.PreApply(n, s, d)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	expectedOutput := "data.aws_availability_zones.available: Destroying... (ID: 2017-03-05 10:56:59.298784526 +0000 UTC)\n"
	output := ui.OutputWriter.String()
	if output != expectedOutput {
		t.Fatalf("Output didn't match.\nExpected: %q\nGiven: %q", expectedOutput, output)
	}

	expectedErrOutput := ""
	errOutput := ui.ErrorWriter.String()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

func TestUiHookPostApply_emptyState(t *testing.T) {
	ui := cli.NewMockUi()
	h := &UiHook{
		Colorize: &colorstring.Colorize{
			Colors:  colorstring.DefaultColors,
			Disable: true,
			Reset:   true,
		},
		Ui: ui,
	}
	h.init()
	h.resources = map[string]uiResourceState{
		"data.google_compute_zones.available": uiResourceState{
			Op:    uiResourceDestroy,
			Start: time.Now(),
		},
	}

	n := &terraform.InstanceInfo{
		Id:         "data.google_compute_zones.available",
		ModulePath: []string{"root"},
		Type:       "google_compute_zones",
	}
	action, err := h.PostApply(n, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	expectedRegexp := "^data.google_compute_zones.available: Destruction complete after -?[a-z0-9.]+\n$"
	output := ui.OutputWriter.String()
	if matched, _ := regexp.MatchString(expectedRegexp, output); !matched {
		t.Fatalf("Output didn't match regexp.\nExpected: %q\nGiven: %q", expectedRegexp, output)
	}

	expectedErrOutput := ""
	errOutput := ui.ErrorWriter.String()
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
