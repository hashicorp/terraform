// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/arguments"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/moduleref"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/xlab/treeprint"
)

type Modules interface {
	// Display renders the list of module entries.
	Display(manifest moduleref.Manifest, diags tfdiags.Diagnostics) int

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewModules(vt arguments.ViewType, view *View) Modules {
	switch vt {
	case arguments.ViewJSON:
		return NewModulesJSON(view)
	case arguments.ViewHuman:
		return &ModulesHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type ModulesHuman struct {
	view *View
}

var _ Modules = (*ModulesHuman)(nil)

func (v *ModulesHuman) Display(manifest moduleref.Manifest, diags tfdiags.Diagnostics) int {
	v.view.Diagnostics(diags) // Print any warnings

	if len(manifest.Records) == 0 {
		v.view.streams.Println("No modules found in configuration.")
		return 0
	}
	printRoot := treeprint.New()

	// ensure output is deterministic
	sort.Sort(manifest.Records)

	populateTreeNode(printRoot, &moduleref.Record{
		Children: manifest.Records,
	})

	v.view.streams.Println(fmt.Sprintf("\nModules declared by configuration:\n%s", printRoot.String()))
	return 0
}

func populateTreeNode(tree treeprint.Tree, node *moduleref.Record) {
	for _, childNode := range node.Children {
		item := fmt.Sprintf("\"%s\"[%s]", childNode.Key, childNode.Source.String())
		if childNode.Version != nil {
			item += fmt.Sprintf(" %s", childNode.Version)
			// Avoid rendering the version constraint if an exact version is given i.e. 'version = "1.2.3"'
			if childNode.VersionConstraints != nil && childNode.VersionConstraints.String() != childNode.Version.String() {
				item += fmt.Sprintf(" (%s)", childNode.VersionConstraints.String())
			}
		}
		branch := tree.AddBranch(item)
		populateTreeNode(branch, childNode)
	}
}

func (v *ModulesHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type ModulesJSON struct {
	view *JSONStaticView[ModulesOutput]

	formatVersion string
}

func NewModulesJSON(view *View) *ModulesJSON {
	// FormatVersion represents the version of the json format and will be
	// incremented for any change to this format that requires changes to a
	// consuming parser.
	const formatVersion = "1.1"

	return &ModulesJSON{
		view:          NewJSONStaticView[ModulesOutput](view),
		formatVersion: formatVersion,
	}
}

type ModulesOutput struct {
	FormatVersion string                  `json:"format_version"`
	Modules       []ModuleOutput          `json:"modules"`
	Diagnostics   []*viewsjson.Diagnostic `json:"diagnostics"`
}

type ModuleOutput struct {
	Key     string `json:"key"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

var _ Modules = (*ModulesJSON)(nil)

func (v *ModulesJSON) Display(manifest moduleref.Manifest, diags tfdiags.Diagnostics) int {
	flattenedManifest := flattenManifest(manifest, v.formatVersion, v.view.prepareDiagnostics(diags))
	v.view.Print(flattenedManifest)
	return 0
}

// FlattenManifest returns the nested contents of a moduleref.Manifest as
// a ModulesOutput with the VersionConstraints and Children attributes
// ommited for the purposes of the json format of the modules command
func flattenManifest(m moduleref.Manifest, formatVersion string, diags []*viewsjson.Diagnostic) ModulesOutput {
	var flatten func(records []*moduleref.Record)
	recordList := make([]ModuleOutput, 0) // Ensure non-nil slice for JSON output
	flatten = func(records []*moduleref.Record) {
		for _, record := range records {
			if record.Version != nil {
				recordList = append(recordList, ModuleOutput{
					Key:     record.Key,
					Source:  record.Source.String(),
					Version: record.Version.String(),
				})
			} else {
				recordList = append(recordList, ModuleOutput{
					Key:     record.Key,
					Source:  record.Source.String(),
					Version: "",
				})
			}

			if len(record.Children) > 0 {
				flatten(record.Children)
			}
		}
	}

	flatten(m.Records)
	ret := ModulesOutput{
		FormatVersion: formatVersion,
		Modules:       recordList,
		Diagnostics:   diags,
	}
	return ret
}

// Diagnostics prints a JSON output that contains only the format and diagnostics information.
func (v *ModulesJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Print(ModulesOutput{
		FormatVersion: v.formatVersion,
		Diagnostics:   v.view.prepareDiagnostics(diags),
	})
}
