package local

import (
	"log"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/format"
)

// backend.CLI impl.
func (b *Local) CLIInit(opts *backend.CLIOpts) error {
	b.CLI = opts.CLI
	b.CLIColor = opts.CLIColor
	b.Streams = opts.Streams
	b.ShowDiagnostics = opts.ShowDiagnostics
	b.ContextOpts = opts.ContextOpts
	b.OpInput = opts.Input
	b.OpValidation = opts.Validation
	b.RunningInAutomation = opts.RunningInAutomation

	// configure any new cli options
	if opts.StatePath != "" {
		log.Printf("[TRACE] backend/local: CLI option -state is overriding state path to %s", opts.StatePath)
		b.OverrideStatePath = opts.StatePath
	}

	if opts.StateOutPath != "" {
		log.Printf("[TRACE] backend/local: CLI option -state-out is overriding state output path to %s", opts.StateOutPath)
		b.OverrideStateOutPath = opts.StateOutPath
	}

	if opts.StateBackupPath != "" {
		log.Printf("[TRACE] backend/local: CLI option -backup is overriding state backup path to %s", opts.StateBackupPath)
		b.OverrideStateBackupPath = opts.StateBackupPath
	}

	return nil
}

// outputColumns returns the number of text character cells any non-error
// output should be wrapped to.
//
// This is the number of columns to use if you are calling b.CLI.Output or
// b.CLI.Info.
func (b *Local) outputColumns() int {
	if b.Streams == nil {
		// We can potentially get here in tests, if they don't populate the
		// CLIOpts fully.
		return 78 // placeholder just so we don't panic
	}
	return b.Streams.Stdout.Columns()
}

// errorColumns returns the number of text character cells any error
// output should be wrapped to.
//
// This is the number of columns to use if you are calling b.CLI.Error or
// b.CLI.Warn.
//
//lint:ignore U1000 TODO
func (b *Local) errorColumns() int {
	if b.Streams == nil {
		// We can potentially get here in tests, if they don't populate the
		// CLIOpts fully.
		return 78 // placeholder just so we don't panic
	}
	return b.Streams.Stderr.Columns()
}

// outputHorizRule will call b.CLI.Output with enough horizontal line
// characters to fill an entire row of output.
//
// This function does nothing if the backend doesn't have a CLI attached.
//
// If UI color is enabled, the rule will get a dark grey coloring to try to
// visually de-emphasize it.
func (b *Local) outputHorizRule() {
	if b.CLI == nil {
		return
	}
	b.CLI.Output(format.HorizontalRule(b.CLIColor, b.outputColumns()))
}
