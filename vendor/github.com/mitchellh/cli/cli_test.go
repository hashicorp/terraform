package cli

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestCLIIsHelp(t *testing.T) {
	testCases := []struct {
		args   []string
		isHelp bool
	}{
		{[]string{"-h"}, true},
		{[]string{"-help"}, true},
		{[]string{"--help"}, true},
		{[]string{"-h", "foo"}, true},
		{[]string{"foo", "bar"}, false},
		{[]string{"-v", "bar"}, false},
		{[]string{"foo", "-h"}, false},
		{[]string{"foo", "-help"}, false},
		{[]string{"foo", "--help"}, false},
	}

	for _, testCase := range testCases {
		cli := &CLI{Args: testCase.args}
		result := cli.IsHelp()

		if result != testCase.isHelp {
			t.Errorf("Expected '%#v'. Args: %#v", testCase.isHelp, testCase.args)
		}
	}
}

func TestCLIIsVersion(t *testing.T) {
	testCases := []struct {
		args      []string
		isVersion bool
	}{
		{[]string{"-v"}, true},
		{[]string{"-version"}, true},
		{[]string{"--version"}, true},
		{[]string{"-v", "foo"}, true},
		{[]string{"foo", "bar"}, false},
		{[]string{"-h", "bar"}, false},
		{[]string{"foo", "-v"}, false},
		{[]string{"foo", "-version"}, false},
		{[]string{"foo", "--version"}, false},
	}

	for _, testCase := range testCases {
		cli := &CLI{Args: testCase.args}
		result := cli.IsVersion()

		if result != testCase.isVersion {
			t.Errorf("Expected '%#v'. Args: %#v", testCase.isVersion, testCase.args)
		}
	}
}

func TestCLIRun(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"foo", "-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != command.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !command.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(command.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", command.RunArgs)
	}
}

func TestCLIRun_blank(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"", "foo", "-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return command, nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != command.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !command.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(command.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", command.RunArgs)
	}
}

func TestCLIRun_default(t *testing.T) {
	commandBar := new(MockCommand)
	commandBar.RunResult = 42

	cli := &CLI{
		Args: []string{"-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"": func() (Command, error) {
				return commandBar, nil
			},
			"foo": func() (Command, error) {
				return new(MockCommand), nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != commandBar.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !commandBar.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(commandBar.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", commandBar.RunArgs)
	}
}

func TestCLIRun_nested(t *testing.T) {
	command := new(MockCommand)
	cli := &CLI{
		Args: []string{"foo", "bar", "-bar", "-baz"},
		Commands: map[string]CommandFactory{
			"foo": func() (Command, error) {
				return new(MockCommand), nil
			},
			"foo bar": func() (Command, error) {
				return command, nil
			},
		},
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != command.RunResult {
		t.Fatalf("bad: %d", exitCode)
	}

	if !command.RunCalled {
		t.Fatalf("run should be called")
	}

	if !reflect.DeepEqual(command.RunArgs, []string{"-bar", "-baz"}) {
		t.Fatalf("bad args: %#v", command.RunArgs)
	}
}

func TestCLIRun_nestedMissingParent(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := &CLI{
		Args: []string{"foo"},
		Commands: map[string]CommandFactory{
			"foo bar": func() (Command, error) {
				return &MockCommand{SynopsisText: "hi!"}, nil
			},
		},
		HelpWriter: buf,
	}

	exitCode, err := cli.Run()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if exitCode != 1 {
		t.Fatalf("bad exit code: %d", exitCode)
	}

	if buf.String() != testCommandNestedMissingParent {
		t.Fatalf("bad: %#v", buf.String())
	}
}

func TestCLIRun_printHelp(t *testing.T) {
	testCases := [][]string{
		{},
		{"-h"},
		{"i-dont-exist"},
		{"-bad-flag", "foo"},
	}

	for _, testCase := range testCases {
		buf := new(bytes.Buffer)
		helpText := "foo"

		cli := &CLI{
			Args: testCase,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return new(MockCommand), nil
				},
			},
			HelpFunc: func(map[string]CommandFactory) string {
				return helpText
			},
			HelpWriter: buf,
		}

		code, err := cli.Run()
		if err != nil {
			t.Errorf("Args: %#v. Error: %s", testCase, err)
			continue
		}

		if code != 1 {
			t.Errorf("Args: %#v. Code: %d", testCase, code)
			continue
		}

		if !strings.Contains(buf.String(), helpText) {
			t.Errorf("Args: %#v. Text: %v", testCase, buf.String())
		}
	}
}

func TestCLIRun_printCommandHelp(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo"},
		{"-h", "foo"},
	}

	for _, args := range testCases {
		command := &MockCommand{
			HelpText: "donuts",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return command, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 1 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != (command.HelpText + "\n") {
			t.Fatalf("bad: %#v", buf.String())
		}
	}
}

func TestCLIRun_printCommandHelpSubcommands(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo"},
		{"-h", "foo"},
	}

	for _, args := range testCases {
		command := &MockCommand{
			HelpText: "donuts",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return command, nil
				},
				"foo bar": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
				"foo longer": func() (Command, error) {
					return &MockCommand{SynopsisText: "hi!"}, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 1 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != testCommandHelpSubcommandsOutput {
			t.Fatalf("bad: %#v", buf.String())
		}
	}
}

func TestCLIRun_printCommandHelpTemplate(t *testing.T) {
	testCases := [][]string{
		{"--help", "foo"},
		{"-h", "foo"},
	}

	for _, args := range testCases {
		command := &MockCommandHelpTemplate{
			MockCommand: MockCommand{
				HelpText: "donuts",
			},

			HelpTemplateText: "hello {{.Help}}",
		}

		buf := new(bytes.Buffer)
		cli := &CLI{
			Args: args,
			Commands: map[string]CommandFactory{
				"foo": func() (Command, error) {
					return command, nil
				},
			},
			HelpWriter: buf,
		}

		exitCode, err := cli.Run()
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if exitCode != 1 {
			t.Fatalf("bad exit code: %d", exitCode)
		}

		if buf.String() != "hello "+command.HelpText+"\n" {
			t.Fatalf("bad: %#v", buf.String())
		}
	}
}

func TestCLISubcommand(t *testing.T) {
	testCases := []struct {
		args       []string
		subcommand string
	}{
		{[]string{"bar"}, "bar"},
		{[]string{"foo", "-h"}, "foo"},
		{[]string{"-h", "bar"}, "bar"},
		{[]string{"foo", "bar", "-h"}, "foo"},
	}

	for _, testCase := range testCases {
		cli := &CLI{Args: testCase.args}
		result := cli.Subcommand()

		if result != testCase.subcommand {
			t.Errorf("Expected %#v, got %#v. Args: %#v",
				testCase.subcommand, result, testCase.args)
		}
	}
}

func TestCLISubcommand_nested(t *testing.T) {
	testCases := []struct {
		args       []string
		subcommand string
	}{
		{[]string{"bar"}, "bar"},
		{[]string{"foo", "-h"}, "foo"},
		{[]string{"-h", "bar"}, "bar"},
		{[]string{"foo", "bar", "-h"}, "foo bar"},
		{[]string{"foo", "bar", "baz", "-h"}, "foo bar"},
		{[]string{"foo", "bar", "-h", "baz"}, "foo bar"},
	}

	for _, testCase := range testCases {
		cli := &CLI{
			Args: testCase.args,
			Commands: map[string]CommandFactory{
				"foo bar": func() (Command, error) {
					return new(MockCommand), nil
				},
			},
		}
		result := cli.Subcommand()

		if result != testCase.subcommand {
			t.Errorf("Expected %#v, got %#v. Args: %#v",
				testCase.subcommand, result, testCase.args)
		}
	}
}

const testCommandNestedMissingParent = `This command is accessed by using one of the subcommands below.

Subcommands:

    bar    hi!

`

const testCommandHelpSubcommandsOutput = `donuts

Subcommands:

    bar       hi!
    longer    hi!

`
