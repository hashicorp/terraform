package backend

import (
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

// CLI is an optional interface that can be implemented to be initialized
// with information from the Terraform CLI. If this is implemented, this
// initialization function will be called with data to help interact better
// with a CLI.
//
// This interface was created to improve backend interaction with the
// official Terraform CLI while making it optional for API users to have
// to provide full CLI interaction to every backend.
//
// If you're implementing a Backend, it is acceptable to require CLI
// initialization. In this case, your backend should be coded to error
// on other methods (such as State, Operation) if CLI initialization was not
// done with all required fields.
type CLI interface {
	Backend

	// CLIInit is called once with options. The options passed to this
	// function may not be modified after calling this since they can be
	// read/written at any time by the Backend implementation.
	//
	// This may be called before or after Configure is called, so if settings
	// here affect configurable settings, care should be taken to handle
	// whether they should be overwritten or not.
	CLIInit(*CLIOpts) error
}

// CLIOpts are the options passed into CLIInit for the CLI interface.
//
// These options represent the functionality the CLI exposes and often
// maps to meta-flags available on every CLI (such as -input).
//
// When implementing a backend, it isn't expected that every option applies.
// Your backend should be documented clearly to explain to end users what
// options have an affect and what won't. In some cases, it may even make sense
// to error in your backend when an option is set so that users don't make
// a critically incorrect assumption about behavior.
type CLIOpts struct {
	// CLI and Colorize control the CLI output. If CLI is nil then no CLI
	// output will be done. If CLIColor is nil then no coloring will be done.
	CLI      cli.Ui
	CLIColor *colorstring.Colorize

	// StatePath is the local path where state is read from.
	//
	// StateOutPath is the local path where the state will be written.
	// If this is empty, it will default to StatePath.
	//
	// StateBackupPath is the local path where a backup file will be written.
	// If this is empty, no backup will be taken.
	StatePath       string
	StateOutPath    string
	StateBackupPath string

	// ContextOpts are the base context options to set when initializing a
	// Terraform context. Many of these will be overridden or merged by
	// Operation. See Operation for more details.
	ContextOpts *terraform.ContextOpts

	// Input will ask for necessary input prior to performing any operations.
	//
	// Validation will perform validation prior to running an operation. The
	// variable naming doesn't match the style of others since we have a func
	// Validate.
	Input      bool
	Validation bool

	// RunningInAutomation indicates that commands are being run by an
	// automated system rather than directly at a command prompt.
	//
	// This is a hint not to produce messages that expect that a user can
	// run a follow-up command, perhaps because Terraform is running in
	// some sort of workflow automation tool that abstracts away the
	// exact commands that are being run.
	RunningInAutomation bool
}
