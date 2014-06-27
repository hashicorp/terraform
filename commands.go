package main

import (
	"os"

	"github.com/hashicorp/terraform/command"
	"github.com/mitchellh/cli"
)

// Commands is the mapping of all the available Terraform commands.
var Commands map[string]cli.CommandFactory

// Ui is the cli.Ui used for communicating to the outside world.
var Ui cli.Ui

const ErrorPrefix = "e:"
const OutputPrefix = "o:"

func init() {
	Ui = &cli.PrefixedUi{
		AskPrefix:    OutputPrefix,
		OutputPrefix: OutputPrefix,
		InfoPrefix:   OutputPrefix,
		ErrorPrefix:  ErrorPrefix,
		Ui:           &cli.BasicUi{Writer: os.Stdout},
	}

	Commands = map[string]cli.CommandFactory{
		"apply": func() (cli.Command, error) {
			return &command.ApplyCommand{
				TFConfig: &TFConfig,
				Ui:       Ui,
			}, nil
		},

		"plan": func() (cli.Command, error) {
			return &command.PlanCommand{
				TFConfig: &TFConfig,
				Ui:       Ui,
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Revision:          GitCommit,
				Version:           Version,
				VersionPrerelease: VersionPrerelease,
				Ui:                Ui,
			}, nil
		},
	}
}
