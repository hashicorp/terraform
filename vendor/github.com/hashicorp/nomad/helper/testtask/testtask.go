// Package testtask implements a portable set of commands useful as stand-ins
// for user tasks.
package testtask

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/hashicorp/nomad/client/driver/env"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/kardianos/osext"
)

// Path returns the path to the currently running executable.
func Path() string {
	path, err := osext.Executable()
	if err != nil {
		panic(err)
	}
	return path
}

// SetEnv configures the environment of the task so that Run executes a testtask
// script when called from within cmd.
func SetEnv(env *env.TaskEnvironment) {
	env.AppendEnvvars(map[string]string{"TEST_TASK": "execute"})
}

// SetCmdEnv configures the environment of cmd so that Run executes a testtask
// script when called from within cmd.
func SetCmdEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "TEST_TASK=execute")
}

// SetTaskEnv configures the environment of t so that Run executes a testtask
// script when called from within t.
func SetTaskEnv(t *structs.Task) {
	if t.Env == nil {
		t.Env = map[string]string{}
	}
	t.Env["TEST_TASK"] = "execute"
}

// Run interprets os.Args as a testtask script if the current program was
// launched with an environment configured by SetCmdEnv or SetTaskEnv. It
// returns false if the environment was not set by this package.
func Run() bool {
	switch tm := os.Getenv("TEST_TASK"); tm {
	case "":
		return false
	case "execute":
		execute()
		return true
	default:
		fmt.Fprintf(os.Stderr, "unexpected value for TEST_TASK, \"%s\"\n", tm)
		os.Exit(1)
		return true
	}
}

func execute() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "no command provided")
		os.Exit(1)
	}

	args := os.Args[1:]

	// popArg removes the first argument from args and returns it.
	popArg := func() string {
		s := args[0]
		args = args[1:]
		return s
	}

	// execute a sequence of operations from args
	for len(args) > 0 {
		switch cmd := popArg(); cmd {

		case "sleep":
			// sleep <dur>: sleep for a duration indicated by the first
			// argument
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "expected arg for sleep")
				os.Exit(1)
			}
			dur, err := time.ParseDuration(popArg())
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not parse sleep time: %v", err)
				os.Exit(1)
			}
			time.Sleep(dur)

		case "echo":
			// echo <msg>: write the msg followed by a newline to stdout.
			fmt.Println(popArg())

		case "write":
			// write <msg> <file>: write a message to a file. The first
			// argument is the msg. The second argument is the path to the
			// target file.
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "expected two args for write")
				os.Exit(1)
			}
			msg := popArg()
			file := popArg()
			ioutil.WriteFile(file, []byte(msg), 0666)

		default:
			fmt.Fprintln(os.Stderr, "unknown command:", cmd)
			os.Exit(1)
		}
	}
}
