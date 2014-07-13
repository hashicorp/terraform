package main

import (
	"os"
	"os/signal"

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
				ShutdownCh:  makeShutdownCh(),
				ContextOpts: &ContextOpts,
				Ui:          Ui,
			}, nil
		},

		"graph": func() (cli.Command, error) {
			return &command.GraphCommand{
				ContextOpts: &ContextOpts,
				Ui:          Ui,
			}, nil
		},

		"plan": func() (cli.Command, error) {
			return &command.PlanCommand{
				ContextOpts: &ContextOpts,
				Ui:          Ui,
			}, nil
		},

		"refresh": func() (cli.Command, error) {
			return &command.RefreshCommand{
				ContextOpts: &ContextOpts,
				Ui:          Ui,
			}, nil
		},

		"show": func() (cli.Command, error) {
			return &command.ShowCommand{
				ContextOpts: &ContextOpts,
				Ui:          Ui,
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

// makeShutdownCh creates an interrupt listener and returns a channel.
// A message will be sent on the channel for every interrupt received.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}
