// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Query renders outputs for query executions.
type Query interface {
	// List renders the list of resources that were discovered in the
	// query.
	List(terraform.ListStates)

	// Resource renders the output for a single resource.
	Resource(addrs.AbsResourceInstance, *states.ResourceInstanceObjectSrc)

	// Conclusion should print out a summary of the tests including their
	// completed status.
	Conclusion(suite *moduletest.Suite)

	// Diagnostics prints out the provided diagnostics.
	Diagnostics(list addrs.List, diags tfdiags.Diagnostics)

	// Interrupted prints out a message stating that an interrupt has been
	// received and testing will stop.
	Interrupted()

	// FatalInterrupt prints out a message stating that a hard interrupt has
	// been received and testing will stop and cleanup will be skipped.
	FatalInterrupt()
}

func NewQuery(vt arguments.ViewType, view *View) Query {
	switch vt {
	case arguments.ViewJSON:
		return &QueryJSON{
			view: NewJSONView(view),
		}
	case arguments.ViewHuman:
		return &QueryHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type QueryHuman struct {
	CloudHooks

	view *View
}

var _ Query = (*QueryHuman)(nil)

func (t *QueryHuman) Abstract(_ *moduletest.Suite) {
	// Do nothing, we don't print an abstract for the human view.
}

func (t *QueryHuman) List(states terraform.ListStates) {
	for _, state := range states.All() {
		for _, instance := range state {
			t.view.streams.Printf("  - %s\n", instance)
		}
	}
}

func (t *QueryHuman) Resource(list addrs.AbsResourceInstance, src *states.ResourceInstanceObjectSrc) {
	t.view.streams.Println()
	t.view.streams.Printf("identity: %s\n", src.IdentityJSON)
	t.view.streams.Printf("resource: %s\n", src.AttrsJSON)
}

func (t *QueryHuman) Conclusion(suite *moduletest.Suite) {
	t.view.streams.Println()

	counts := make(map[moduletest.Status]int)
	for _, file := range suite.Files {
		for _, run := range file.Runs {
			count := counts[run.Status]
			counts[run.Status] = count + 1
		}
	}

	if suite.Status <= moduletest.Skip {
		// Then no tests.
		t.view.streams.Print("Executed 0 tests")
		if counts[moduletest.Skip] > 0 {
			t.view.streams.Printf(", %d skipped.\n", counts[moduletest.Skip])
		} else {
			t.view.streams.Println(".")
		}
		return
	}

	if suite.Status == moduletest.Pass {
		t.view.streams.Print(t.view.colorize.Color("[green]Success![reset]"))
	} else {
		t.view.streams.Print(t.view.colorize.Color("[red]Failure![reset]"))
	}

	t.view.streams.Printf(" %d passed, %d failed", counts[moduletest.Pass], counts[moduletest.Fail]+counts[moduletest.Error])
	if counts[moduletest.Skip] > 0 {
		t.view.streams.Printf(", %d skipped.\n", counts[moduletest.Skip])
	} else {
		t.view.streams.Println(".")
	}
}

func (t *QueryHuman) Diagnostics(list addrs.List, diags tfdiags.Diagnostics) {
	t.view.Diagnostics(diags)
}

func (t *QueryHuman) Interrupted() {
	t.view.streams.Eprintln(format.WordWrap(interrupted, t.view.errorColumns()))
}

func (t *QueryHuman) FatalInterrupt() {
	t.view.streams.Eprintln(format.WordWrap(fatalInterrupt, t.view.errorColumns()))
}

type QueryJSON struct {
	CloudHooks

	view *JSONView
}

var _ Query = (*QueryJSON)(nil)

func (t *QueryJSON) List(states terraform.ListStates) {
	jsonStates := make(map[string][]string)

	for addr, state := range states.All() {
		var instances []string
		for _, instance := range state {
			instances = append(instances, string(instance.AttrsJSON))
		}
		jsonStates[addr.Name] = instances
	}

	t.view.log.Info(
		"Resource list",
		"resources", jsonStates)
}

func (t *QueryJSON) Resource(list addrs.AbsResourceInstance, src *states.ResourceInstanceObjectSrc) {
	t.view.log.Info(
		"Resource",
		"identity", src.IdentityJSON,
		"resource", src.AttrsJSON)
}

func (t *QueryJSON) Conclusion(suite *moduletest.Suite) {
	summary := json.TestSuiteSummary{
		Status: json.ToTestStatus(suite.Status),
	}
	for _, file := range suite.Files {
		for _, run := range file.Runs {
			switch run.Status {
			case moduletest.Skip:
				summary.Skipped++
			case moduletest.Pass:
				summary.Passed++
			case moduletest.Error:
				summary.Errored++
			case moduletest.Fail:
				summary.Failed++
			}
		}
	}

	var message bytes.Buffer
	if suite.Status <= moduletest.Skip {
		// Then no tests.
		message.WriteString("Executed 0 tests")
		if summary.Skipped > 0 {
			message.WriteString(fmt.Sprintf(", %d skipped.", summary.Skipped))
		} else {
			message.WriteString(".")
		}
	} else {
		if suite.Status == moduletest.Pass {
			message.WriteString("Success!")
		} else {
			message.WriteString("Failure!")
		}

		message.WriteString(fmt.Sprintf(" %d passed, %d failed", summary.Passed, summary.Failed+summary.Errored))
		if summary.Skipped > 0 {
			message.WriteString(fmt.Sprintf(", %d skipped.", summary.Skipped))
		} else {
			message.WriteString(".")
		}
	}

	t.view.log.Info(
		message.String(),
		"type", json.MessageTestSummary,
		json.MessageTestSummary, summary)
}

func (t *QueryJSON) Diagnostics(list addrs.List, diags tfdiags.Diagnostics) {
	var metadata []interface{}
	t.view.Diagnostics(diags, metadata...)
}

func (t *QueryJSON) Interrupted() {
	t.view.Log(interrupted)
}

func (t *QueryJSON) FatalInterrupt() {
	t.view.Log(fatalInterrupt)
}
