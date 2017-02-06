package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/state"
)

// UnlockCommand is a cli.Command implementation that manually unlocks
// the state.
type UnlockCommand struct {
	Meta
}

func (c *UnlockCommand) Run(args []string) int {
	args = c.Meta.process(args, false)

	cmdFlags := c.Meta.flagSet("unlock")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// assume everything is initialized. The user can manually init if this is
	// required.
	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		ConfigPath: configPath,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	st, err := b.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	s, ok := st.(state.Locker)
	if !ok {
		c.Ui.Error("Current state does not support locking")
		return 1
	}

	if err := s.Unlock(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to unlock state: %s", err))
		return 1
	}

	return 0
}

func (c *UnlockCommand) Help() string {
	helpText := `
Usage: terraform unlock [DIR]

  Manually unlock the state for the defined configuration.

  This will not modify your infrastructure. This command removes the lock on the
  state for the current configuration. The behavior of this lock is dependent
  on the backend being used. Local state files cannot be unlocked by another
  process.
`
	return strings.TrimSpace(helpText)
}

func (c *UnlockCommand) Synopsis() string {
	return "Manually unlock the terraform state"
}
