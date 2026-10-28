// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/xlab/treeprint"
)

// Providers is the view used for the `providers` command.
type Providers interface {
	LogProviderRequirements(configReqs *configs.ModuleRequirements, stateReqs getproviders.Requirements, diags tfdiags.Diagnostics)

	Diagnostics(diags tfdiags.Diagnostics)
}

func NewProviders(viewType arguments.ViewType, view *View) Providers {
	switch viewType {
	case arguments.ViewHuman:
		return &ProvidersHuman{view: view}
	default:
		panic("unsupported view type: " + string(viewType))
	}
}

// ProvidersHuman renders module installation progress and diagnostics for direct consumption.
type ProvidersHuman struct {
	view *View
}

var _ Providers = (*ProvidersHuman)(nil)

func (v *ProvidersHuman) LogProviderRequirements(configReqs *configs.ModuleRequirements, stateReqs getproviders.Requirements, diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)

	// Handle config requirements
	printRoot := treeprint.New()

	if configReqs != nil {
		v.populateTreeNode(printRoot, configReqs)
	}

	v.view.streams.Println("\nProviders required by configuration:")
	v.view.streams.Println(printRoot.String())

	// Handle state requirements, if any
	if len(stateReqs) > 0 {
		v.view.streams.Print("Providers required by state:\n\n")

		// Ensure deterministic order
		ps := slices.SortedFunc(maps.Keys(stateReqs), func(a, b addrs.Provider) int {
			if a.LessThan(b) {
				return -1
			} else if b.LessThan(a) {
				return 1
			}
			return 0
		})

		for _, fqn := range ps {
			v.view.streams.Print(fmt.Sprintf("    provider[%s]\n\n", fqn.String()))
		}
	}
}

func (v *ProvidersHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (c *ProvidersHuman) populateTreeNode(tree treeprint.Tree, reqs *configs.ModuleRequirements) {
	// Ensure deterministic order when rendering requirements
	sortFunc := func(a, b addrs.Provider) int {
		if a.LessThan(b) {
			return -1
		} else if b.LessThan(a) {
			return 1
		}
		return 0
	}

	ps := slices.SortedFunc(maps.Keys(reqs.Requirements), sortFunc)
	for _, fqn := range ps {
		dep := reqs.Requirements[fqn]
		versionsStr := getproviders.VersionConstraintsString(dep)
		if versionsStr != "" {
			versionsStr = " " + versionsStr
		}
		tree.AddNode(fmt.Sprintf("provider[%s]%s", fqn.String(), versionsStr))
	}
	for name, testNode := range reqs.Tests {
		name = strings.TrimSuffix(name, ".tftest.hcl")
		name = strings.ReplaceAll(name, "/", ".")
		branch := tree.AddBranch(fmt.Sprintf("test.%s", name))

		ps := slices.SortedFunc(maps.Keys(testNode.Requirements), sortFunc)
		for _, fqn := range ps {
			dep := testNode.Requirements[fqn]
			versionsStr := getproviders.VersionConstraintsString(dep)
			if versionsStr != "" {
				versionsStr = " " + versionsStr
			}
			branch.AddNode(fmt.Sprintf("provider[%s]%s", fqn.String(), versionsStr))
		}

		for _, run := range testNode.Runs {
			branch := branch.AddBranch(fmt.Sprintf("run.%s", run.Name))
			c.populateTreeNode(branch, run)
		}
	}
	for name, childNode := range reqs.Children {
		branch := tree.AddBranch(fmt.Sprintf("module.%s", name))
		c.populateTreeNode(branch, childNode)
	}
}
