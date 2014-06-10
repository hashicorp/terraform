package main

import (
	"os"

	"github.com/hashicorp/terraform/command"
	"github.com/mitchellh/cli"
)

// Commands is the mapping of all the available Terraform commands.
var Commands map[string]cli.CommandFactory

const ErrorPrefix = "e:"
const OutputPrefix = "o:"

func init() {
	ui := &cli.PrefixedUi{
		AskPrefix:    OutputPrefix,
		OutputPrefix: OutputPrefix,
		InfoPrefix:   OutputPrefix,
		ErrorPrefix:  ErrorPrefix,
		Ui:           &cli.BasicUi{Writer: os.Stdout},
	}

	Commands = map[string]cli.CommandFactory{
		"apply": func() (cli.Command, error) {
			return &command.ApplyCommand{
				Ui: ui,
			}, nil
		},

		"diff": func() (cli.Command, error) {
			return &command.DiffCommand{
				TFConfig: &TFConfig,
				Ui:       ui,
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Revision:          GitCommit,
				Version:           Version,
				VersionPrerelease: VersionPrerelease,
				Ui:                ui,
			}, nil
		},
	}
}
