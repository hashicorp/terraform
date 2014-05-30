package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ActiveState/tail"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/panicwrap"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	var wrapConfig panicwrap.WrapConfig

	if !panicwrap.Wrapped(&wrapConfig) {
		// Determine where logs should go in general (requested by the user)
		logWriter, err := logOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't setup log output: %s", err)
			return 1
		}

		// We always send logs to a temporary file that we use in case
		// there is a panic. Otherwise, we delete it.
		logTempFile, err := ioutil.TempFile("", "terraform-log")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't setup logging tempfile: %s", err)
			return 1
		}
		logTempFile.Close()
		defer os.Remove(logTempFile.Name())

		// Tell the logger to log to this file
		os.Setenv(EnvLog, "1")
		os.Setenv(EnvLogFile, logTempFile.Name())

		if logWriter != nil {
			// Start tailing the file beforehand to get the data
			t, err := tail.TailFile(logTempFile.Name(), tail.Config{
				Follow:    true,
				Logger:    tail.DiscardingLogger,
				MustExist: true,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't setup logging tempfile: %s", err)
				return 1
			}
			go func() {
				for line := range t.Lines {
					logWriter.Write([]byte(line.Text + "\n"))
				}
			}()
		}

		// Create the configuration for panicwrap and wrap our executable
		wrapConfig.Handler = panicHandler(logTempFile)
		exitStatus, err := panicwrap.Wrap(&wrapConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't start Terraform: %s", err)
			return 1
		}

		// If >= 0, we're the parent, so just exit
		if exitStatus >= 0 {
			return exitStatus
		}

		// We're the child, so just close the tempfile we made in order to
		// save file handles since the tempfile is only used by the parent.
		logTempFile.Close()
	}

	// Call the real main
	return wrappedMain()
}

func wrappedMain() int {
	logOutput, err := logOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting Terraform: %s", err)
		return 1
	}
	if logOutput != nil {
		log.SetOutput(logOutput)
	}

	// Get the command line args. We shortcut "--version" and "-v" to
	// just show the version.
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "-v" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		}
	}

	cli := &cli.CLI{
		Args:     args,
		Commands: Commands,
		HelpFunc: cli.BasicHelpFunc("terraform"),
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}
