package command

import (
	"github.com/posener/complete"
)

// This file contains some re-usable predictors for auto-complete. The
// command-specific autocomplete configurations live within each command's
// own source file, as AutocompleteArgs and AutocompleteFlags methods on each
// Command implementation.

// For completing the value of boolean flags like -foo false
var completePredictBoolean = complete.PredictSet("true", "false")

// We don't currently have a real predictor for module sources, but
// we'll probably add one later.
var completePredictModuleSource = complete.PredictAnything

type completePredictSequence []complete.Predictor

func (s completePredictSequence) Predict(a complete.Args) []string {
	// Only one level of command is stripped off the prefix of a.Completed
	// here, so nested subcommands like "workspace new" will need to provide
	// dummy entries (e.g. complete.PredictNothing) as placeholders for
	// all but the first subcommand. For example, "workspace new" needs
	// one placeholder for the argument "new".
	idx := len(a.Completed)
	if idx >= len(s) {
		return nil
	}

	return s[idx].Predict(a)
}

func (m *Meta) completePredictWorkspaceName() complete.Predictor {
	return complete.PredictFunc(func(a complete.Args) []string {
		// There are lot of things that can fail in here, so if we encounter
		// any error then we'll just return nothing and not support autocomplete
		// until whatever error is fixed. (The user can't actually see the error
		// here, but other commands should produce a user-visible error before
		// too long.)

		// We assume here that we want to autocomplete for the current working
		// directory, since we don't have enough context to know where to
		// find any config path argument, and it might be _after_ the argument
		// we're trying to complete here anyway.
		configPath, err := ModulePath(nil)
		if err != nil {
			return nil
		}

		backendConfig, diags := m.loadBackendConfig(configPath)
		if diags.HasErrors() {
			return nil
		}

		b, diags := m.Backend(&BackendOpts{
			Config: backendConfig,
		})
		if diags.HasErrors() {
			return nil
		}

		names, _ := b.States()
		return names
	})
}
