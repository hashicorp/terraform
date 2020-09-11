# complete

[![Build Status](https://travis-ci.org/posener/complete.svg?branch=master)](https://travis-ci.org/posener/complete)
[![codecov](https://codecov.io/gh/posener/complete/branch/master/graph/badge.svg)](https://codecov.io/gh/posener/complete)
[![golangci](https://golangci.com/badges/github.com/posener/complete.svg)](https://golangci.com/r/github.com/posener/complete)
[![GoDoc](https://godoc.org/github.com/posener/complete?status.svg)](http://godoc.org/github.com/posener/complete)
[![goreadme](https://goreadme.herokuapp.com/badge/posener/complete.svg)](https://goreadme.herokuapp.com)

Package complete provides a tool for bash writing bash completion in go, and bash completion for the go command line.

Writing bash completion scripts is a hard work. This package provides an easy way
to create bash completion scripts for any command, and also an easy way to install/uninstall
the completion of the command.

#### Go Command Bash Completion

In [./cmd/gocomplete](./cmd/gocomplete) there is an example for bash completion for the `go` command line.

This is an example that uses the `complete` package on the `go` command - the `complete` package
can also be used to implement any completions, see #usage.

#### Install

1. Type in your shell:

```go
go get -u github.com/posener/complete/gocomplete
gocomplete -install
```

2. Restart your shell

Uninstall by `gocomplete -uninstall`

#### Features

- Complete `go` command, including sub commands and all flags.
- Complete packages names or `.go` files when necessary.
- Complete test names after `-run` flag.

#### Complete package

Supported shells:

- [x] bash
- [x] zsh
- [x] fish

#### Usage

Assuming you have program called `run` and you want to have bash completion
for it, meaning, if you type `run` then space, then press the `Tab` key,
the shell will suggest relevant complete options.

In that case, we will create a program called `runcomplete`, a go program,
with a `func main()` and so, that will make the completion of the `run`
program. Once the `runcomplete` will be in a binary form, we could
`runcomplete -install` and that will add to our shell all the bash completion
options for `run`.

So here it is:

```go
import "github.com/posener/complete"

func main() {

	// create a Command object, that represents the command we want
	// to complete.
	run := complete.Command{

		// Sub defines a list of sub commands of the program,
		// this is recursive, since every command is of type command also.
		Sub: complete.Commands{

			// add a build sub command
			"build": complete.Command {

				// define flags of the build sub command
				Flags: complete.Flags{
					// build sub command has a flag '-cpus', which
					// expects number of cpus after it. in that case
					// anything could complete this flag.
					"-cpus": complete.PredictAnything,
				},
			},
		},

		// define flags of the 'run' main command
		Flags: complete.Flags{
			// a flag -o, which expects a file ending with .out after
			// it, the tab completion will auto complete for files matching
			// the given pattern.
			"-o": complete.PredictFiles("*.out"),
		},

		// define global flags of the 'run' main command
		// those will show up also when a sub command was entered in the
		// command line
		GlobalFlags: complete.Flags{

			// a flag '-h' which does not expects anything after it
			"-h": complete.PredictNothing,
		},
	}

	// run the command completion, as part of the main() function.
	// this triggers the autocompletion when needed.
	// name must be exactly as the binary that we want to complete.
	complete.New("run", run).Run()
}
```

#### Self completing program

In case that the program that we want to complete is written in go we
can make it self completing.
Here is an example: [./example/self/main.go](./example/self/main.go) .

## Sub Packages

* [cmd](./cmd): Package cmd used for command line options for the complete tool

* [gocomplete](./gocomplete): Package main is complete tool for the go command line

* [match](./match): Package match contains matchers that decide if to apply completion.


---

Created by [goreadme](https://github.com/apps/goreadme)
