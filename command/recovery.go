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

	args, err := c.Meta.process(args, true)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to execute 'recover': %s", err))
	}

	cmdFlags := c.Meta.flagSet("recover")

	cmdFlags.BoolVar(&c.FailFast, "stop-if-import-filed", false, "Stop recovery execution after the first import failed attempt.")
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
	stateStore, err := b.State(c.Workspace())
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
	} else {
		fmt.Printf("State does not support 'RecoveryLogReader' functional.\n" +
			"Please make sure what you set AWS remote backend as backend. Currently, only this backend is supported.\n")
	}

	c.Ui.Info("Recovery complete.\n")
	return 0
}

func (c *RecoveryCommand) removeRecoveryLog(stateStore state.State) {
	if recoveryLogWriter, ok := stateStore.(state.RecoveryLogWriter); ok {
		c.Ui.Info(fmt.Sprintf("Remove recovery log...\n"))
		recoveryLogWriter.DeleteRecoveryLog()
	} else {
		fmt.Printf("State does not support 'RecoveryLogWriter' functional.\n" +
			"Please make sure what you set AWS remote backend as backend. Currently, only this backend is supported.\n")
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
			c.Ui.Info("Import failure\n")
			if c.FailFast {
				c.Ui.Info("Flag 'stop-if-import-filed' was set as 'true'. Stopping recovery.\n")
				c.Ui.Info("You can try to solve this problem manually using recovery log and lost resource log.\n")
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
		c.Ui.Info("A problem was detected. But the error was suppressed. See execution log above.")
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

  -stop-if-import-filed=false   Stop recovery execution after the first import failed attempt.

  -suppress-errors=false        Suppress error (return code 0) if any internal problem detected.
  `

	return helpString
}

func (c *RecoveryCommand) Synopsis() string {
	return "Find recovery log in remote bucket and try to recover resources."
}
