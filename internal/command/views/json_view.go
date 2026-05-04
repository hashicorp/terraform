// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	encJson "encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// This version describes the schema of JSON UI messages. This version must be
// updated after making any changes to this view, the jsonHook, or any of the
// command/views/json package.
const JSON_UI_VERSION = "1.3"

func NewJSONView(view *View) *JSONView {
	log := hclog.New(&hclog.LoggerOptions{
		Name:       "terraform.ui",
		Output:     view.streams.Stdout.File,
		JSONFormat: true,
	})
	jv := &JSONView{
		log:  log,
		view: view,
	}
	jv.Version()
	return jv
}

type JSONView struct {
	// hclog is used for all output in JSON UI mode. The logger has an internal
	// mutex to ensure that messages are not interleaved.
	log hclog.Logger

	// We hold a reference to the view entirely to allow us to access the
	// ConfigSources function pointer, in order to render source snippets into
	// diagnostics. This is even more unfortunate than the same reference in the
	// view.
	//
	// Do not be tempted to dereference the configSource value upon logger init,
	// as it will likely be updated later.
	view *View
}

func (v *JSONView) Version() {
	version := tfversion.String()
	v.log.Info(
		fmt.Sprintf("Terraform %s", version),
		"type", json.MessageVersion,
		"terraform", version,
		"ui", JSON_UI_VERSION,
	)
}

func (v *JSONView) Log(message string) {
	v.log.Info(message, "type", json.MessageLog)
}

func (v *JSONView) StateDump(state string) {
	v.log.Info(
		"Emergency state dump",
		"type", json.MessageLog,
		"state", encJson.RawMessage(state),
	)
}

func (v *JSONView) Diagnostics(diags tfdiags.Diagnostics, metadata ...interface{}) {
	sources := v.view.configSources()
	for _, diag := range diags {
		diagnostic := json.NewDiagnostic(diag, sources)

		args := []interface{}{"type", json.MessageDiagnostic, "diagnostic", diagnostic}
		args = append(args, metadata...)

		switch diag.Severity() {
		case tfdiags.Warning:
			v.log.Warn(fmt.Sprintf("Warning: %s", diag.Description().Summary), args...)
		default:
			v.log.Error(fmt.Sprintf("Error: %s", diag.Description().Summary), args...)
		}
	}
}

func (v *JSONView) PlannedChange(c *json.ResourceInstanceChange) {
	v.log.Info(
		c.String(),
		"type", json.MessagePlannedChange,
		"change", c,
	)
}

func (v *JSONView) PlannedActionInvocation(action *json.ActionInvocation) {
	v.log.Info(
		fmt.Sprintf("planned action invocation: %s", action.Action.Action),
		"type", json.MessagePlannedActionInvocation,
		"invocation", action,
	)
}

func (v *JSONView) AppliedActionInvocation(action *json.ActionInvocation) {
	v.log.Info(
		fmt.Sprintf("applied action invocation: %s", action.Action.Action),
		"type", json.MessageAppliedActionInvocation,
		"invocation", action,
	)
}

func (v *JSONView) ResourceDrift(c *json.ResourceInstanceChange) {
	v.log.Info(
		fmt.Sprintf("%s: Drift detected (%s)", c.Resource.Addr, c.Action),
		"type", json.MessageResourceDrift,
		"change", c,
	)
}

func (v *JSONView) ChangeSummary(cs *json.ChangeSummary) {
	v.log.Info(
		cs.String(),
		"type", json.MessageChangeSummary,
		"changes", cs,
	)
}

func (v *JSONView) Hook(h json.Hook) {
	v.log.Info(
		h.String(),
		"type", h.HookType(),
		"hook", h,
	)
}

func (v *JSONView) Outputs(outputs json.Outputs) {
	v.log.Info(
		outputs.String(),
		"type", json.MessageOutputs,
		"outputs", outputs,
	)
}

func (v *JSONView) PolicyResults(results *plans.PolicyResults) {
	if results == nil {
		return
	}

	// Log all non-policy-specific diagnostics if any.
	for _, diag := range results.Diagnostics {
		v.logPolicyDiagnostic(diag)
	}

	for addr, result := range results.Iter() {
		// Log all the info messages
		for _, enforcement := range result.EvaluationResponse.Enforcements {
			if enforcement.Message == "" {
				continue
			}
			var src []byte
			if enforcement.LocalRange != nil {
				src = v.view.configSources()[enforcement.LocalRange.Filename]
			}
			info := json.NewPolicyInfo(src, enforcement)
			args := []any{
				"type", json.MessagePolicyInfo,
				"target_address", addr,
				json.MessagePolicyInfo, info,
				"@policy", "true",
				"result", enforcement.Result.String(),
			}
			if enforcement.Policy != nil {
				args = append(args, "policy_metadata", json.MetadataFromEnforcement(enforcement))
			}
			v.log.Info("Policy info", args...)
		}

		for _, diag := range result.EvaluationResponse.Diagnostics {
			v.logPolicyDiagnostic(diag, "target_address", addr)
		}

		for _, policy := range result.EvaluationResponse.Policies {
			v.log.Info(
				"Policy Result",
				"type", json.MessagePolicyEvaluationResult,
				"result", policy.Result.String(),
				"target_address", addr,
				"policy_address", policy.Address,
				"@policy", "true",
				"policy_metadata", json.MetadataFromPolicy(*policy),
			)
		}
	}
}

func (v *JSONView) logPolicyDiagnostic(diag tfdiags.Diagnostic, extraArgs ...any) {
	// Log the policy diagnostics. The severity level here is from the policy engine, and terraform
	// does not use it at all. Therefore, the log level of these diagnostics is only relevant
	// for policies.
	sources := v.view.configSources()
	diagnostic := json.NewDiagnostic(diag, sources)

	args := []any{
		"type", json.MessagePolicyDiagnostic,
		"@policy", "true",
		json.MessagePolicyDiagnostic, diagnostic,
	}
	args = append(args, extraArgs...)
	extra := tfdiags.ExtraInfo[*policy.PolicyExtra](diag)
	if extra != nil {
		policyMetadata := json.MetadataFromPolicy(extra.Policy)
		if extra.EnforceIndex != nil {
			policyMetadata.EnforceIndex = extra.EnforceIndex
		}
		args = append(args, "policy_metadata", policyMetadata)
		args = append(args, "result", extra.Result.String())
	}
	switch extra.Severity {
	case hcl.DiagWarning:
		v.log.Warn(fmt.Sprintf("Warning: %s", diag.Description().Summary), args...)
	default:
		v.log.Error(fmt.Sprintf("Error: %s", diag.Description().Summary), args...)
	}
}
