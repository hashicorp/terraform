package funcs

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestCmdlineUnix(t *testing.T) {
	tests := []struct {
		Args []cty.Value
		Want cty.Value
	}{
		{
			[]cty.Value{
				cty.StringVal("whoami"),
			},
			cty.StringVal(`'whoami'`),
		},
		{
			[]cty.Value{
				cty.StringVal("bleep bloop"),
			},
			cty.StringVal(`'bleep bloop'`),
		},
		{
			[]cty.Value{
				cty.StringVal("cat"),
				cty.StringVal("foo.txt"),
			},
			cty.StringVal(`'cat' foo.txt`),
		},
		{
			[]cty.Value{
				cty.StringVal("echo"),
				cty.StringVal("Hello World"),
			},
			cty.StringVal(`'echo' 'Hello World'`),
		},
		{
			[]cty.Value{
				cty.StringVal("echo"),
				cty.StringVal("I'm Terraform"),
			},
			cty.StringVal(`'echo' 'I'\''m Terraform'`),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("CmdlineUnix(%#v)", test.Args), func(t *testing.T) {
			got, err := CmdlineUnixFunc.Call(test.Args)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestCmdlineWindows(t *testing.T) {
	// The underling shquot library always quotes the first argument
	// (the program name) because that ensures consistent processing
	// both for programs that use the CommandLineToArgvW function and
	// for the Visual C++ runtime startup code. That then gets escaped
	// as ^" to robustly pass through cmd.exe. These steps together
	// produce a result that is not what a human author would typically
	// write, but this approach avoids various edge-cases in the
	// command line processing rules.

	tests := []struct {
		Args []cty.Value
		Want cty.Value
	}{
		{
			[]cty.Value{
				cty.StringVal("ver"),
			},
			cty.StringVal(`^"ver^"`),
		},
		{
			[]cty.Value{
				cty.StringVal("bleep bloop"),
			},
			cty.StringVal(`^"bleep bloop^"`),
		},
		{
			[]cty.Value{
				cty.StringVal("type"),
				cty.StringVal("foo.txt"),
			},
			cty.StringVal(`^"type^" foo.txt`),
		},
		{
			[]cty.Value{
				cty.StringVal("echo"),
				cty.StringVal("Hello World"),
			},
			cty.StringVal(`^"echo^" ^"Hello World^"`),
		},
		{
			[]cty.Value{
				cty.StringVal("echo"),
				cty.StringVal(`I said "Hello"!`),
			},
			cty.StringVal(`^"echo^" ^"I said \^"Hello\^"^!^"`),
		},
		{
			[]cty.Value{
				cty.StringVal("echo"),
				cty.StringVal(`^`),
			},
			cty.StringVal(`^"echo^" ^^`),
		},
		{
			[]cty.Value{
				cty.StringVal(`with"quote`),
			},
			// The escaping rules for the command line don't offer any
			// reliable way to escape double quotes, so the shquot
			// library just strips them out. This is not a big problem in
			// practice because a quote character can never appear in a
			// valid command name anyway.
			cty.StringVal(`^"withquote^"`),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("CmdlineWindows(%#v)", test.Args), func(t *testing.T) {
			got, err := CmdlineWindowsFunc.Call(test.Args)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
