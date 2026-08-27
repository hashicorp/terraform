// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Query renders outputs for query executions.
type Query interface {
	Operation() Operation
	Hooks() []terraform.Hook

	Diagnostics(diags tfdiags.Diagnostics)
	HelpPrompt()
}

func NewQuery(vt arguments.ViewType, view *View) Query {
	switch vt {
	case arguments.ViewJSON:
		jv := NewJSONView(view)
		op := &QueryOperationJSON{view: jv, queryPolicy: newQueryPolicyView()}
		return &QueryJSON{view: jv, op: op}
	case arguments.ViewHuman:
		op := &QueryOperationHuman{
			view:         view,
			inAutomation: view.RunningInAutomation(),
			queryPolicy:  newQueryPolicyView(),
		}
		return &QueryHuman{view: view, op: op}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type QueryHuman struct {
	view *View
	op   *QueryOperationHuman
}

var _ Query = (*QueryHuman)(nil)

func (v *QueryHuman) Operation() Operation {
	return v.op
}

func (v *QueryHuman) Hooks() []terraform.Hook {
	hook := NewUiHook(v.view)
	return []terraform.Hook{&queryUiHook{UiHook: hook, op: v.op}}
}

func (v *QueryHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
func (v *QueryHuman) HelpPrompt() {
	v.view.HelpPrompt("query")
}

type QueryJSON struct {
	view *JSONView
	op   *QueryOperationJSON
}

var _ Query = (*QueryJSON)(nil)

func (v *QueryJSON) Operation() Operation {
	return v.op
}

func (v *QueryJSON) Hooks() []terraform.Hook {
	hook := newJSONHook(v.view)
	return []terraform.Hook{&queryJSONHook{jsonHook: hook, op: v.op}}
}

func (v *QueryJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *QueryJSON) HelpPrompt() {
}

// queryUiHook wraps UiHook and routes PolicyResult through the query operation
// so that query policy results are buffered in the queryPolicyView instead of
// being emitted immediately as generic policy_result records.
type queryUiHook struct {
	*UiHook
	op *QueryOperationHuman
}

func (h *queryUiHook) PolicyResult(addr string, resp policy.EvaluationResponse) (terraform.HookAction, error) {
	h.viewLock.Lock()
	defer h.viewLock.Unlock()
	h.op.PolicyResult(addr, resp)
	return terraform.HookActionContinue, nil
}

// queryJSONHook wraps jsonHook and routes PolicyResult through the query
// operation so that query policy results are buffered in the queryPolicyView
// instead of being emitted immediately as generic policy_result records.
type queryJSONHook struct {
	*jsonHook
	op *QueryOperationJSON
}

func (h *queryJSONHook) PolicyResult(addr string, resp policy.EvaluationResponse) (terraform.HookAction, error) {
	h.op.PolicyResult(addr, resp)
	return terraform.HookActionContinue, nil
}
