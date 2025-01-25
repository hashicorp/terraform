// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	encJson "encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/moduleref"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/xlab/treeprint"
)

type Modules interface {
	// Display renders the list of module entries.
	Display(manifest moduleref.Manifest) int

	// Diagnostics renders early diagnostics, resulting from argument parsing.
	Diagnostics(diags tfdiags.Diagnostics)
}

func NewModules(vt arguments.ViewType, view *View) Modules {
	switch vt {
	case arguments.ViewJSON:
		return &ModulesJSON{view: view}
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

func (v *ModulesHuman) Display(manifest moduleref.Manifest) int {
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
	view *View
}

var _ Modules = (*ModulesHuman)(nil)

func (v *ModulesJSON) Display(manifest moduleref.Manifest) int {
	var bytes []byte
	var err error

	flattenedManifest := flattenManifest(manifest)
	if bytes, err = encJson.Marshal(flattenedManifest); err != nil {
		v.view.streams.Eprintf("error marshalling manifest: %v", err)
		return 1
	}

	v.view.streams.Println(string(bytes))
	return 0
}

// FlattenManifest returns the nested contents of a moduleref.Manifest in
// a flattened format with the VersionConstraints and Children attributes
// ommited for the purposes of the json format of the modules command
func flattenManifest(m moduleref.Manifest) map[string]interface{} {
	var flatten func(records []*moduleref.Record)
	var recordList []map[string]string
	flatten = func(records []*moduleref.Record) {
		for _, record := range records {
			if record.Version != nil {
				recordList = append(recordList, map[string]string{
					"key":     record.Key,
					"source":  record.Source.String(),
					"version": record.Version.String(),
				})
			} else {
				recordList = append(recordList, map[string]string{
					"key":     record.Key,
					"source":  record.Source.String(),
					"version": "",
				})
			}

			if len(record.Children) > 0 {
				flatten(record.Children)
			}
		}
	}

	flatten(m.Records)
	ret := map[string]interface{}{
		"format_version": m.FormatVersion,
		"modules":        recordList,
	}
	return ret
}

func (v *ModulesJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
