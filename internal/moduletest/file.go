// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type File struct {
	Config *configs.TestFile

	Name   string
	Status Status

	Runs []*Run

	Diagnostics tfdiags.Diagnostics
}

// GetAllVariableDefinitions returns all the definitions we have for named
// variables in any root config files required by this test file.
func (file *File) GetAllVariableDefinitions(config *configs.Config) map[string]*configs.Variable {
	variables := make(map[string]*configs.Variable)
	for name, variable := range config.Module.Variables {
		variables[name] = variable
	}

	for _, run := range file.Runs {
		if config := run.Config.ConfigUnderTest; config != nil {
			for name, variable := range config.Module.Variables {
				if _, exists := variables[name]; exists {
					// It might be that the same variable definition is shared
					// between different modules. That's fine, but they could
					// be different so we'll default to the definition we found
					// first.
					continue
				}
				variables[name] = variable
			}
		}
	}
	return variables
}
