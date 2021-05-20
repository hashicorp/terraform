package views

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
)

// Test the PreApply hook for creating a new resource
func TestUiHookPreApply_create(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)
	h.resources = map[string]uiResourceState{
		"test_instance.foo": {
			Op:    uiResourceCreate,
			Start: time.Now(),
		},
	}

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	priorState := cty.NullVal(cty.Object(map[string]cty.Type{
		"id":  cty.String,
		"bar": cty.List(cty.String),
	}))
	plannedNewState := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
		"bar": cty.ListVal([]cty.Value{
			cty.StringVal("baz"),
		}),
	})

	action, err := h.PreApply(addr, states.CurrentGen, plans.Create, priorState, plannedNewState)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	// stop the background writer
	uiState := h.resources[addr.String()]
	close(uiState.DoneCh)
	<-uiState.done

	expectedOutput := "test_instance.foo: Creating...\n"
	result := done(t)
	output := result.Stdout()
	if output != expectedOutput {
		t.Fatalf("Output didn't match.\nExpected: %q\nGiven: %q", expectedOutput, output)
	}

	expectedErrOutput := ""
	errOutput := result.Stderr()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

// Test the PreApply hook's use of a periodic timer to display "still working"
// log lines
func TestUiHookPreApply_periodicTimer(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)
	h.periodicUiTimer = 1 * time.Second
	h.resources = map[string]uiResourceState{
		"test_instance.foo": {
			Op:    uiResourceModify,
			Start: time.Now(),
		},
	}

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	priorState := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("test"),
		"bar": cty.ListValEmpty(cty.String),
	})
	plannedNewState := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
		"bar": cty.ListVal([]cty.Value{
			cty.StringVal("baz"),
		}),
	})

	action, err := h.PreApply(addr, states.CurrentGen, plans.Update, priorState, plannedNewState)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	time.Sleep(3100 * time.Millisecond)

	// stop the background writer
	uiState := h.resources[addr.String()]
	close(uiState.DoneCh)
	<-uiState.done

	expectedOutput := `test_instance.foo: Modifying... [id=test]
test_instance.foo: Still modifying... [id=test, 1s elapsed]
test_instance.foo: Still modifying... [id=test, 2s elapsed]
test_instance.foo: Still modifying... [id=test, 3s elapsed]
`
	result := done(t)
	output := result.Stdout()
	if output != expectedOutput {
		t.Fatalf("Output didn't match.\nExpected: %q\nGiven: %q", expectedOutput, output)
	}

	expectedErrOutput := ""
	errOutput := result.Stderr()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

// Test the PreApply hook's destroy path, including passing a deposed key as
// the gen argument.
func TestUiHookPreApply_destroy(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)
	h.resources = map[string]uiResourceState{
		"test_instance.foo": {
			Op:    uiResourceDestroy,
			Start: time.Now(),
		},
	}

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	priorState := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("abc123"),
		"verbs": cty.ListVal([]cty.Value{
			cty.StringVal("boop"),
		}),
	})
	plannedNewState := cty.NullVal(cty.Object(map[string]cty.Type{
		"id":    cty.String,
		"verbs": cty.List(cty.String),
	}))

	key := states.NewDeposedKey()
	action, err := h.PreApply(addr, key, plans.Delete, priorState, plannedNewState)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}

	// stop the background writer
	uiState := h.resources[addr.String()]
	close(uiState.DoneCh)
	<-uiState.done

	result := done(t)
	expectedOutput := fmt.Sprintf("test_instance.foo (deposed object %s): Destroying... [id=abc123]\n", key)
	output := result.Stdout()
	if output != expectedOutput {
		t.Fatalf("Output didn't match.\nExpected: %q\nGiven: %q", expectedOutput, output)
	}

	expectedErrOutput := ""
	errOutput := result.Stderr()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

// Verify that colorize is called on format strings, not user input, by adding
// valid color codes as resource names and IDs.
func TestUiHookPostApply_colorInterpolation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: false})
	h := NewUiHook(view)
	h.resources = map[string]uiResourceState{
		"test_instance.foo[\"[red]\"]": {
			Op:    uiResourceCreate,
			Start: time.Now(),
		},
	}

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.StringKey("[red]")).Absolute(addrs.RootModuleInstance)

	newState := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("[blue]"),
	})

	action, err := h.PostApply(addr, states.CurrentGen, newState, nil)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	reset := "\x1b[0m"
	bold := "\x1b[1m"
	wantPrefix := reset + bold + `test_instance.foo["[red]"]: Creation complete after`
	wantSuffix := "[id=[blue]]" + reset + "\n"
	output := result.Stdout()

	if !strings.HasPrefix(output, wantPrefix) {
		t.Fatalf("wrong output prefix\n got: %#v\nwant: %#v", output, wantPrefix)
	}

	if !strings.HasSuffix(output, wantSuffix) {
		t.Fatalf("wrong output suffix\n got: %#v\nwant: %#v", output, wantSuffix)
	}

	expectedErrOutput := ""
	errOutput := result.Stderr()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

// Test that the PostApply hook renders a total time.
func TestUiHookPostApply_emptyState(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)
	h.resources = map[string]uiResourceState{
		"data.google_compute_zones.available": {
			Op:    uiResourceDestroy,
			Start: time.Now(),
		},
	}

	addr := addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "google_compute_zones",
		Name: "available",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	newState := cty.NullVal(cty.Object(map[string]cty.Type{
		"id":    cty.String,
		"names": cty.List(cty.String),
	}))

	action, err := h.PostApply(addr, states.CurrentGen, newState, nil)
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	expectedRegexp := "^data.google_compute_zones.available: Destruction complete after -?[a-z0-9µ.]+\n$"
	output := result.Stdout()
	if matched, _ := regexp.MatchString(expectedRegexp, output); !matched {
		t.Fatalf("Output didn't match regexp.\nExpected: %q\nGiven: %q", expectedRegexp, output)
	}

	expectedErrOutput := ""
	errOutput := result.Stderr()
	if errOutput != expectedErrOutput {
		t.Fatalf("Error output didn't match.\nExpected: %q\nGiven: %q", expectedErrOutput, errOutput)
	}
}

func TestPreProvisionInstanceStep(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	action, err := h.PreProvisionInstanceStep(addr, "local-exec")
	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	if got, want := result.Stdout(), "test_instance.foo: Provisioning with 'local-exec'...\n"; got != want {
		t.Fatalf("unexpected output\n got: %q\nwant: %q", got, want)
	}
}

// Test ProvisionOutput, including lots of edge cases for the output
// whitespace/line ending logic.
func TestProvisionOutput(t *testing.T) {
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	testCases := map[string]struct {
		provisioner string
		input       string
		wantOutput  string
	}{
		"single line": {
			"local-exec",
			"foo\n",
			"test_instance.foo (local-exec): foo\n",
		},
		"multiple lines": {
			"x",
			`foo
bar
baz
`,
			`test_instance.foo (x): foo
test_instance.foo (x): bar
test_instance.foo (x): baz
`,
		},
		"trailing whitespace": {
			"x",
			"foo                  \nbar\n",
			"test_instance.foo (x): foo\ntest_instance.foo (x): bar\n",
		},
		"blank lines": {
			"x",
			"foo\n\nbar\n\n\nbaz\n",
			`test_instance.foo (x): foo
test_instance.foo (x): bar
test_instance.foo (x): baz
`,
		},
		"no final newline": {
			"x",
			`foo
bar`,
			`test_instance.foo (x): foo
test_instance.foo (x): bar
`,
		},
		"CR, no LF": {
			"MacOS 9?",
			"foo\rbar\r",
			`test_instance.foo (MacOS 9?): foo
test_instance.foo (MacOS 9?): bar
`,
		},
		"CRLF": {
			"winrm",
			"foo\r\nbar\r\n",
			`test_instance.foo (winrm): foo
test_instance.foo (winrm): bar
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			h := NewUiHook(view)

			h.ProvisionOutput(addr, tc.provisioner, tc.input)
			result := done(t)

			if got := result.Stdout(); got != tc.wantOutput {
				t.Fatalf("unexpected output\n got: %q\nwant: %q", got, tc.wantOutput)
			}
		})
	}
}

// Test the PreRefresh hook in the normal path where the resource exists with
// an ID key and value in the state.
func TestPreRefresh(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	priorState := cty.ObjectVal(map[string]cty.Value{
		"id":  cty.StringVal("test"),
		"bar": cty.ListValEmpty(cty.String),
	})

	action, err := h.PreRefresh(addr, states.CurrentGen, priorState)

	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	if got, want := result.Stdout(), "test_instance.foo: Refreshing state... [id=test]\n"; got != want {
		t.Fatalf("unexpected output\n got: %q\nwant: %q", got, want)
	}
}

// Test that PreRefresh still works if no ID key and value can be determined
// from state.
func TestPreRefresh_noID(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	priorState := cty.ObjectVal(map[string]cty.Value{
		"bar": cty.ListValEmpty(cty.String),
	})

	action, err := h.PreRefresh(addr, states.CurrentGen, priorState)

	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	if got, want := result.Stdout(), "test_instance.foo: Refreshing state...\n"; got != want {
		t.Fatalf("unexpected output\n got: %q\nwant: %q", got, want)
	}
}

// Test the very simple PreImportState hook.
func TestPreImportState(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	action, err := h.PreImportState(addr, "test")

	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	if got, want := result.Stdout(), "test_instance.foo: Importing from ID \"test\"...\n"; got != want {
		t.Fatalf("unexpected output\n got: %q\nwant: %q", got, want)
	}
}

// Test the PostImportState UI hook. Again, this hook behaviour seems odd to
// me (see below), so please don't consider these tests as justification for
// keeping this behaviour.
func TestPostImportState(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	h := NewUiHook(view)

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	// The "Prepared [...] for import" lines display the type name of each of
	// the imported resources passed to the hook. I'm not sure how it's
	// possible for an import to result in a different resource type name than
	// the target address, but the hook works like this so we're covering it.
	imported := []providers.ImportedResource{
		{
			TypeName: "test_some_instance",
			State: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("test"),
			}),
		},
		{
			TypeName: "test_other_instance",
			State: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("test"),
			}),
		},
	}

	action, err := h.PostImportState(addr, imported)

	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
	result := done(t)

	want := `test_instance.foo: Import prepared!
  Prepared test_some_instance for import
  Prepared test_other_instance for import
`
	if got := result.Stdout(); got != want {
		t.Fatalf("unexpected output\n got: %q\nwant: %q", got, want)
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
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あ...さ",
			MaxLen:   3,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あ...さ",
			MaxLen:   5,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あい...さ",
			MaxLen:   6,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あい...こさ",
			MaxLen:   7,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あいう...こさ",
			MaxLen:   8,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あいう...けこさ",
			MaxLen:   9,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あいうえ...けこさ",
			MaxLen:   10,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あいうえおかきくけこさ",
			MaxLen:   11,
		},
		{
			Input:    "あいうえおかきくけこさ",
			Expected: "あいうえおかきくけこさ",
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
