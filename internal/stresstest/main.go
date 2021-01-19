package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
)

func main() {
	// For this tool, most output is logging on stderr, though some specific
	// commands might also produce result output on stdout. We typically
	// report failures through logging, so nothing else ends up on Stderr.
	log.SetPrefix("")
	log.SetOutput(os.Stderr)

	cli := &cli.CLI{
		Name:       "stresstest",
		Args:       os.Args[1:],
		Commands:   commandFactories(),
		HelpFunc:   cli.BasicHelpFunc("stresstest"),
		HelpWriter: os.Stderr,
	}

	exitCode, err := cli.Run()
	if err != nil {
		log.Fatalf("Error executing CLI: %s", err)
	}
	os.Exit(exitCode)
}

func commandFactories() map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"graph": func() (cli.Command, error) {
			return &graphCommand{}, nil
		},
		"graph soak": func() (cli.Command, error) {
			return &graphSoakCommand{}, nil
		},
		"graph export-series": func() (cli.Command, error) {
			return &graphExportSeriesCommand{}, nil
		},
		"terraform": func() (cli.Command, error) {
			return &terraformCommand{}, nil
		},
	}
}
