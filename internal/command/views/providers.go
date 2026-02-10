// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"strings"

	"github.com/xlab/treeprint"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Providers is the view interface for the providers command.
type Providers interface {
	// Output renders the providers required by configuration and state.
	Output(reqs *configs.ModuleRequirements, stateReqs getproviders.Requirements)

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)

	// HelpPrompt renders a prompt directing the user to the help command.
	HelpPrompt()
}

// NewProviders returns an initialized Providers implementation.
func NewProviders(view *View) Providers {
	return &ProvidersHuman{view: view}
}

// ProvidersHuman is the human-readable implementation of the Providers view.
type ProvidersHuman struct {
	view *View
}

var _ Providers = (*ProvidersHuman)(nil)

func (v *ProvidersHuman) Output(reqs *configs.ModuleRequirements, stateReqs getproviders.Requirements) {
	printRoot := treeprint.New()
	populateProviderTreeNode(printRoot, reqs)

	v.view.streams.Println("\nProviders required by configuration:")
	v.view.streams.Print(printRoot.String())

	if len(stateReqs) > 0 {
		v.view.streams.Println("Providers required by state:")
		v.view.streams.Println("")
		for fqn := range stateReqs {
			v.view.streams.Printf("    provider[%s]\n\n", fqn.String())
		}
	}
}

func (v *ProvidersHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *ProvidersHuman) HelpPrompt() {
	v.view.HelpPrompt("providers")
}

// populateProviderTreeNode recursively populates a tree with provider requirements.
func populateProviderTreeNode(tree treeprint.Tree, node *configs.ModuleRequirements) {
	for fqn, dep := range node.Requirements {
		versionsStr := getproviders.VersionConstraintsString(dep)
		if versionsStr != "" {
			versionsStr = " " + versionsStr
		}
		tree.AddNode(fmt.Sprintf("provider[%s]%s", fqn.String(), versionsStr))
	}
	for name, testNode := range node.Tests {
		name = strings.TrimSuffix(name, ".tftest.hcl")
		name = strings.ReplaceAll(name, "/", ".")
		branch := tree.AddBranch(fmt.Sprintf("test.%s", name))

		for fqn, dep := range testNode.Requirements {
			versionsStr := getproviders.VersionConstraintsString(dep)
			if versionsStr != "" {
				versionsStr = " " + versionsStr
			}
			branch.AddNode(fmt.Sprintf("provider[%s]%s", fqn.String(), versionsStr))
		}

		for _, run := range testNode.Runs {
			branch := branch.AddBranch(fmt.Sprintf("run.%s", run.Name))
			populateProviderTreeNode(branch, run)
		}
	}
	for name, childNode := range node.Children {
		branch := tree.AddBranch(fmt.Sprintf("module.%s", name))
		populateProviderTreeNode(branch, childNode)
	}
}
