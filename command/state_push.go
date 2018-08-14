package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

// StatePushCommand is a Command implementation that shows a single resource.
type StatePushCommand struct {
	Meta
	StateMeta
}

func (c *StatePushCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	var flagForce bool
	cmdFlags := c.Meta.flagSet("state push")
	cmdFlags.BoolVar(&flagForce, "force", false, "")
	if err := cmdFlags.Parse(args); err != nil {
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	if len(args) != 1 {
		c.Ui.Error("Exactly one argument expected: path to state to push")
		return 1
	}

	c.Ui.Error("state push not yet updated for new state types")
	return 1

	/*
		// Determine our reader for the input state. This is the filepath
		// or stdin if "-" is given.
		var r io.Reader = os.Stdin
		if args[0] != "-" {
			f, err := os.Open(args[0])
			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}

			// Note: we don't need to defer a Close here because we do a close
			// automatically below directly after the read.

			r = f
		}

		// Read the state
		sourceState, err := terraform.ReadState(r)
		if c, ok := r.(io.Closer); ok {
			// Close the reader if possible right now since we're done with it.
			c.Close()
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error reading source state %q: %s", args[0], err))
			return 1
		}

		// Load the backend
		b, backendDiags := c.Backend(nil)
		if backendDiags.HasErrors() {
			c.showDiagnostics(backendDiags)
			return 1
		}

		// Get the state
		env := c.Workspace()
		state, err := b.StateMgr(env)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load destination state: %s", err))
			return 1
		}
		if err := state.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load destination state: %s", err))
			return 1
		}

		dstState := state.State()

		// If we're not forcing, then perform safety checks
			if !flagForce && !dstState.Empty() {
				if !dstState.SameLineage(sourceState) {
					c.Ui.Error(strings.TrimSpace(errStatePushLineage))
					return 1
				}

				age, err := dstState.CompareAges(sourceState)
				if err != nil {
					c.Ui.Error(err.Error())
					return 1
				}
				if age == terraform.StateAgeReceiverNewer {
					c.Ui.Error(strings.TrimSpace(errStatePushSerialNewer))
					return 1
				}
			}

			// Overwrite it
			if err := state.WriteState(sourceState); err != nil {
				c.Ui.Error(fmt.Sprintf("Failed to write state: %s", err))
				return 1
			}
			if err := state.PersistState(); err != nil {
				c.Ui.Error(fmt.Sprintf("Failed to write state: %s", err))
				return 1
			}
	*/

	return 0
}

func (c *StatePushCommand) Help() string {
	helpText := `
Usage: terraform state push [options] PATH

  Update remote state from a local state file at PATH.

  This command "pushes" a local state and overwrites remote state
  with a local state file. The command will protect you against writing
  an older serial or a different state file lineage unless you specify the
  "-force" flag.

  This command works with local state (it will overwrite the local
  state), but is less useful for this use case.

  If PATH is "-", then this command will read the state to push from stdin.
  Data from stdin is not streamed to the backend: it is loaded completely
  (until pipe close), verified, and then pushed.

Options:

  -force              Write the state even if lineages don't match or the
                      remote serial is higher.

`
	return strings.TrimSpace(helpText)
}

func (c *StatePushCommand) Synopsis() string {
	return "Update remote state from a local state file"
}

const errStatePushLineage = `
The lineages do not match! The state will not be pushed.

The "lineage" is a unique identifier given to a state on creation. It helps
protect Terraform from overwriting a seemingly unrelated state file since it
represents potentially losing real state.

Please verify you're pushing the correct state. If you're sure you are, you
can force the behavior with the "-force" flag.
`

const errStatePushSerialNewer = `
The destination state has a higher serial number! The state will not be pushed.

A higher serial could indicate that there is data in the destination state
that was not present when the source state was created. As a protection measure,
Terraform will not automatically overwrite this state.

Please verify you're pushing the correct state. If you're sure you are, you
can force the behavior with the "-force" flag.
`
