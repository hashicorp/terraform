package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// Meta are the meta-options that are available on all or most commands.
type Meta struct {
	Color       bool
	ContextOpts *terraform.ContextOpts
	Ui          cli.Ui
}

// Colorize returns the colorization structure for a command.
func (m *Meta) Colorize() *colorstring.Colorize {
	return &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: !m.Color,
		Reset:   true,
	}
}

// Context returns a Terraform Context taking into account the context
// options used to initialize this meta configuration.
func (m *Meta) Context(path, statePath string) (*terraform.Context, error) {
	opts := m.contextOpts()

	// First try to just read the plan directly from the path given.
	f, err := os.Open(path)
	if err == nil {
		plan, err := terraform.ReadPlan(f)
		f.Close()
		if err == nil {
			return plan.Context(opts), nil
		}
	}

	if statePath != "" {
		if _, err := os.Stat(statePath); err != nil {
			return nil, fmt.Errorf(
				"There was an error reading the state file. The path\n"+
					"and error are shown below. If you're trying to build a\n"+
					"brand new infrastructure, explicitly pass the '-init'\n"+
					"flag to Terraform to tell it it is okay to build new\n"+
					"state.\n\n"+
					"Path: %s\n"+
					"Error: %s",
				statePath,
				err)
		}
	}

	// Load up the state
	var state *terraform.State
	if statePath != "" {
		f, err := os.Open(statePath)
		if err == nil {
			state, err = terraform.ReadState(f)
			f.Close()
		}

		if err != nil {
			return nil, fmt.Errorf("Error loading state: %s", err)
		}
	}

	config, err := config.LoadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %s", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Error validating config: %s", err)
	}

	opts.Config = config
	opts.State = state
	ctx := terraform.NewContext(opts)

	if _, err := ctx.Plan(nil); err != nil {
		return nil, fmt.Errorf("Error running plan: %s", err)
	}

	return ctx, nil

}

// contextOpts returns the options to use to initialize a Terraform
// context with the settings from this Meta.
func (m *Meta) contextOpts() *terraform.ContextOpts {
	var opts terraform.ContextOpts = *m.ContextOpts
	opts.Hooks = make([]terraform.Hook, len(m.ContextOpts.Hooks)+1)
	opts.Hooks[0] = m.uiHook()
	copy(opts.Hooks[1:], m.ContextOpts.Hooks)
	return &opts
}

// process will process the meta-parameters out of the arguments. This
// will potentially modify the args in-place. It will return the resulting
// slice.
func (m *Meta) process(args []string) []string {
	m.Color = true

	for i, v := range args {
		if v == "-no-color" {
			m.Color = false
			return append(args[:i], args[i+1:]...)
		}
	}

	return args
}

// uiHook returns the UiHook to use with the context.
func (m *Meta) uiHook() *UiHook {
	return &UiHook{
		Colorize: m.Colorize(),
		Ui:       m.Ui,
	}
}
