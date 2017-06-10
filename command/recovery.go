package command

import (
	"fmt"
	"github.com/hashicorp/terraform/state"
)

type RecoveryCommand struct {
	Meta
	FailFast       bool
	SuppressErrors bool
}

func (c *RecoveryCommand) Run(args []string) int {

	args = c.Meta.process(args, true)
	cmdFlags := c.Meta.flagSet("recover")

	cmdFlags.BoolVar(&c.FailFast, "fail-fast", false, "Stop execution after the first failed attempt.")
	cmdFlags.BoolVar(&c.SuppressErrors, "suppress-errors", false, "Suppress error (return code 0) if any internal problem detected.")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	b, err := c.Backend(nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return c.returnError()
	}

	// Get the state
	stateStore, err := b.State(c.Env())
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return c.returnError()
	}

	if recoveryLogReader, ok := stateStore.(state.RecoveryLogReader); ok {
		instances, err := recoveryLogReader.ReadRecoveryLog()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Recovery error: %s", err))
			return c.returnError()
		}
		if len(instances) > 0 {
			if c.startRecoveryProcess(instances) > 0 {
				return c.returnError()
			}
			c.removeRecoveryLog(stateStore)
		} else {
			c.Ui.Info(fmt.Sprintf("Nothing to recover. Recovery log is empty."))
		}
	}
	c.Ui.Info("Recovery complete.\n")
	return 0
}

func (c *RecoveryCommand) removeRecoveryLog(stateStore state.State) {
	if recoveryLogWriter, ok := stateStore.(state.RecoveryLogWriter); ok {
		c.Ui.Info(fmt.Sprintf("Remove recovery log...\n"))
		recoveryLogWriter.DeleteRecoveryLog()
	} else {
		c.Ui.Warn(fmt.Sprintf("State does not support 'RecoveryLogWriter' functional.\n"))
	}
}

func (c *RecoveryCommand) startRecoveryProcess(instances map[string]state.Instance) int {
	returnCode := 0
	for key, instance := range instances {
		c.Ui.Info(fmt.Sprintf("Processing instance record %s\nID: %s Address: %s\n", key, instance.Id, instance.Address))
		importCommand := &ImportCommand{
			Meta: c.Meta,
		}
		if importCommand.Run([]string{instance.Address, instance.Id}) > 0 {
			c.Ui.Info("Import failure")
			if c.FailFast {
				c.Ui.Info("Fast stopped...")
				return 1
			}
			returnCode = 1
		} else {
			c.Ui.Info(fmt.Sprintf("Instance ID: %s successfully imported.\n", instance.Id))
			refreshCommand := RefreshCommand{
				Meta: c.Meta,
			}
			if refreshCommand.Run([]string{}) > 0 {
				c.Ui.Info("Error with perform 'refresh' command.")
			}
		}
	}
	return returnCode
}

func (c *RecoveryCommand) returnError() int {
	if c.SuppressErrors {
		c.Ui.Info("Any problem detected. But the error was suppressed.")
		return 0
	}
	return 1
}

func (c *RecoveryCommand) Help() string {
	helpString := `
Usage: terraform recover [options]

  Find recovery log in remote bucket. Currently only AWS S3 supported.

  This will find recovery log tried to import all specified resources from recovery log.
  Note. After this operation you may need to run apply some times to continue deployment.
  See also terraform import --help

Options:

  -fail-fast=false            Stop execution after the first failed import.

  -suppress-errors=false      Suppress error (return code 0) if any internal problem detected.
  `

	return helpString
}

func (c *RecoveryCommand) Synopsis() string {
	return "Find recovery log in remote bucket."
}
