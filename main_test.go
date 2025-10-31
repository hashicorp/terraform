// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/cli"
)

func TestMain_cliArgsFromEnv(t *testing.T) {
	// Set up the state. This test really messes with the environment and
	// global state so we set things up to be restored.

	// Restore original CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set up test command and restore that
	Commands = make(map[string]cli.CommandFactory)
	defer func() {
		Commands = nil
	}()
	testCommandName := "unit-test-cli-args"
	testCommand := &testCommandCLI{}
	Commands[testCommandName] = func() (cli.Command, error) {
		return testCommand, nil
	}

	cases := []struct {
		Name     string
		Args     []string
		EnvValue string
		Expected []string
		Err      bool
	}{
		{
			"no env",
			[]string{testCommandName, "foo", "bar"},
			"",
			[]string{"foo", "bar"},
			false,
		},

		{
			"both env var and CLI",
			[]string{testCommandName, "foo", "bar"},
			"-foo baz",
			[]string{"-foo", "baz", "foo", "bar"},
			false,
		},

		{
			"only env var",
			[]string{testCommandName},
			"-foo bar",
			[]string{"-foo", "bar"},
			false,
		},

		{
			"cli string has blank values",
			[]string{testCommandName, "bar", "", "baz"},
			"-foo bar",
			[]string{"-foo", "bar", "bar", "", "baz"},
			false,
		},

		{
			"cli string has blank values before the command",
			[]string{"", testCommandName, "bar"},
			"-foo bar",
			[]string{"-foo", "bar", "bar"},
			false,
		},

		{
			// this should fail gracefully, this is just testing
			// that we don't panic with our slice arithmetic
			"no command",
			[]string{},
			"-foo bar",
			nil,
			true,
		},

		{
			"single quoted strings",
			[]string{testCommandName, "foo"},
			"-foo 'bar baz'",
			[]string{"-foo", "bar baz", "foo"},
			false,
		},

		{
			"double quoted strings",
			[]string{testCommandName, "foo"},
			`-foo "bar baz"`,
			[]string{"-foo", "bar baz", "foo"},
			false,
		},

		{
			"double quoted single quoted strings",
			[]string{testCommandName, "foo"},
			`-foo "'bar baz'"`,
			[]string{"-foo", "'bar baz'", "foo"},
			false,
		},

		{
			"backticks taken literally",
			// The shellwords library we use to parse the environment variables
			// has the option to automatically execute commands written in
			// backticks. This test is here to make sure we don't accidentally
			// enable that.
			[]string{testCommandName, "foo"},
			"-foo `echo nope`",
			[]string{"-foo", "`echo nope`", "foo"},
			false,
		},

		{
			"no nested environment variable expansion",
			// The shellwords library we use to parse the environment variables
			// has the option to automatically expand sequences that appear
			// to be environment variable interpolations. This test is here to
			// make sure we don't accidentally enable that.
			[]string{testCommandName, "foo"},
			"-foo $OTHER_ENV",
			[]string{"-foo", "$OTHER_ENV", "foo"},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			t.Setenv(EnvCLI, tc.EnvValue)
			t.Setenv("OTHER_ENV", "placeholder")

			// Set up the args
			args := make([]string, len(tc.Args)+1)
			args[0] = oldArgs[0] // process name
			copy(args[1:], tc.Args)

			// Run it!
			os.Args = args
			testCommand.Args = nil
			exit := realMain()
			if (exit != 0) != tc.Err {
				t.Fatalf("bad: %d", exit)
			}
			if tc.Err {
				return
			}

			// Verify
			if !reflect.DeepEqual(testCommand.Args, tc.Expected) {
				t.Fatalf("expected args %#v but got %#v", tc.Expected, testCommand.Args)
			}
		})
	}
}

// This test just has more options than the test above. Use this for
// more control over behavior at the expense of more complex test structures.
func TestMain_cliArgsFromEnvAdvanced(t *testing.T) {
	// Restore original CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set up test command and restore that
	Commands = make(map[string]cli.CommandFactory)
	defer func() {
		Commands = nil
	}()

	cases := []struct {
		Name     string
		Command  string
		EnvVar   string
		Args     []string
		Value    string
		Expected []string
		Err      bool
	}{
		{
			"targeted to another command",
			"command",
			EnvCLI + "_foo",
			[]string{"command", "foo", "bar"},
			"-flag",
			[]string{"foo", "bar"},
			false,
		},

		{
			"targeted to this command",
			"command",
			EnvCLI + "_command",
			[]string{"command", "foo", "bar"},
			"-flag",
			[]string{"-flag", "foo", "bar"},
			false,
		},

		{
			"targeted to a command with a hyphen",
			"command-name",
			EnvCLI + "_command_name",
			[]string{"command-name", "foo", "bar"},
			"-flag",
			[]string{"-flag", "foo", "bar"},
			false,
		},

		{
			"targeted to a command with a space",
			"command name",
			EnvCLI + "_command_name",
			[]string{"command", "name", "foo", "bar"},
			"-flag",
			[]string{"-flag", "foo", "bar"},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			// Set up test command and restore that
			testCommandName := tc.Command
			testCommand := &testCommandCLI{}
			defer func() { delete(Commands, testCommandName) }()
			Commands[testCommandName] = func() (cli.Command, error) {
				return testCommand, nil
			}

			os.Unsetenv(tc.EnvVar)
			defer os.Unsetenv(tc.EnvVar)

			// Set the env var value
			if tc.Value != "" {
				if err := os.Setenv(tc.EnvVar, tc.Value); err != nil {
					t.Fatalf("err: %s", err)
				}
			}

			// Set up the args
			args := make([]string, len(tc.Args)+1)
			args[0] = oldArgs[0] // process name
			copy(args[1:], tc.Args)

			// Run it!
			os.Args = args
			testCommand.Args = nil
			exit := realMain()
			if (exit != 0) != tc.Err {
				t.Fatalf("unexpected exit status %d; want 0", exit)
			}
			if tc.Err {
				return
			}

			// Verify
			if !reflect.DeepEqual(testCommand.Args, tc.Expected) {
				t.Fatalf("bad: %#v", testCommand.Args)
			}
		})
	}
}

// verify that we output valid autocomplete results
func TestMain_autoComplete(t *testing.T) {
	// Restore original CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Restore stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	// Set up test command and restore that
	Commands = make(map[string]cli.CommandFactory)
	defer func() {
		Commands = nil
	}()
	Commands["version"] = func() (cli.Command, error) {
		return &testCommandCLI{}, nil
	}

	// Run command that should get autocomplete suggestion "version"
	os.Setenv("COMP_LINE", "terraform versio")
	defer os.Unsetenv("COMP_LINE")
	os.Args = []string{"terraform", "terraform", "versio"}
	exit := realMain()
	if exit != 0 {
		t.Fatalf("unexpected exit status %d; want 0", exit)
	}

	// Check autocomplete suggestion
	expectedAutocomplete := "version"
	b := make([]byte, 25)
	n, err := r.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	output := string(b[0:n])
	output = strings.ReplaceAll(output, "\n", "")
	if output != expectedAutocomplete {
		t.Fatalf("expected autocompletion to return %q, but got %q", expectedAutocomplete, output)
	}

	// Run command that should NOT get an autocomplete suggestion
	r, w, _ = os.Pipe()
	os.Stdout = w

	os.Setenv("COMP_LINE", "terraform zzz")
	defer os.Unsetenv("COMP_LINE")
	os.Args = []string{"terraform", "terraform", "zzz"}
	exit = realMain()
	if exit != 0 {
		t.Fatalf("unexpected exit status %d; want 0", exit)
	}

	// Avoid infinite blocking in this case, where no autocomplete suggestions are returned
	r.SetReadDeadline(time.Now().Add(time.Duration(1 * time.Second)))

	// Check autocomplete suggestion
	b = make([]byte, 25)
	n, err = r.Read(b)
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected autocompletion to return 0 bytes, but got %d: %q", n, b[0:n])
	}
}

// Test the autocompleting the workspace name during `terraform workspace select`
func TestMain_autoComplete_workspaceName(t *testing.T) {
	// Restore original CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Restore stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	t.Run("can autocomplete from 'd' to 'default'", func(t *testing.T) {
		// Auto complete in a working directory with no config;
		// an implied local backend is used, which will report the default backend exists.
		td := t.TempDir()
		t.Chdir(td)

		// Set up trigger for command-line autocompletion feature
		// We expect "d" to autocomplete to "default"
		expectedAutocomplete := "default"
		os.Setenv("COMP_LINE", "terraform workspace select d")
		defer os.Unsetenv("COMP_LINE")

		// Run it!
		os.Args = []string{"terraform", "terraform", "workspace", "select", "d"}
		exit := realMain()
		if exit != 0 {
			t.Fatalf("unexpected exit status %d; want 0", exit)
		}

		b := make([]byte, 25)
		n, err := r.Read(b)
		if err != nil {
			t.Fatal(err)
		}
		output := string(b[0:n])
		output = strings.ReplaceAll(output, "\n", "")

		if string(output) != expectedAutocomplete {
			t.Fatalf("expected autocompletion to return %q, but got %q", expectedAutocomplete, output)
		}
	})

	t.Run("can handle autocomplete returning no results", func(t *testing.T) {
		// Auto complete in a working directory with no config;
		// an implied local backend is used, which will report the default backend exists.
		td := t.TempDir()
		t.Chdir(td)

		// Set up trigger for command-line autocompletion feature
		// We expect "z" to not autocomplete to anything, as only 'default' exists.
		os.Setenv("COMP_LINE", "terraform workspace select z")
		defer os.Unsetenv("COMP_LINE")

		// Run it!
		os.Args = []string{"terraform", "terraform", "workspace", "select", "z"}
		exit := realMain()
		if exit != 0 {
			t.Fatalf("unexpected exit status %d; want 0", exit)
		}

		// Avoid infinite blocking in this test case, where no autocomplete suggestions are returned
		r.SetReadDeadline(time.Now().Add(time.Duration(1 * time.Second)))

		b := make([]byte, 25)
		n, err := r.Read(b)
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("expected autocompletion to return 0 bytes, but got %d: %q", n, b[0:n])
		}
	})
}

type testCommandCLI struct {
	Args []string
}

func (c *testCommandCLI) Run(args []string) int {
	c.Args = args
	return 0
}

func (c *testCommandCLI) Synopsis() string { return "" }
func (c *testCommandCLI) Help() string     { return "" }

func TestWarnOutput(t *testing.T) {
	mock := cli.NewMockUi()
	wrapped := &ui{mock}
	wrapped.Warn("WARNING")

	stderr := mock.ErrorWriter.String()
	stdout := mock.OutputWriter.String()

	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}

	if stdout != "WARNING\n" {
		t.Fatalf("unexpected stdout: %q\n", stdout)
	}
}
