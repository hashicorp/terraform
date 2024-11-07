// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package planfile

import (
	"fmt"
	"io"
	"time"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/proto"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

const tfplanFormatVersion = 3
const tfplanFilename = "tfplan"

// ---------------------------------------------------------------------------
// This file deals with the internal structure of the "tfplan" sub-file within
// the plan file format. It's all private API, wrapped by methods defined
// elsewhere. This is the only file that should import the
// ../internal/planproto package, which contains the ugly stubs generated
// by the protobuf compiler.
// ---------------------------------------------------------------------------

// readTfplan reads a protobuf-encoded description from the plan portion of
// a plan file, which is stored in a special file in the archive called
// "tfplan".
func readTfplan(r io.Reader) (*plans.Plan, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var rawPlan planproto.Plan
	err = proto.Unmarshal(src, &rawPlan)
	if err != nil {
		return nil, fmt.Errorf("parse error: %s", err)
	}

	if rawPlan.Version != tfplanFormatVersion {
		return nil, fmt.Errorf("unsupported plan file format version %d; only version %d is supported", rawPlan.Version, tfplanFormatVersion)
	}

	if rawPlan.TerraformVersion != version.String() {
		return nil, fmt.Errorf("plan file was created by Terraform %s, but this is %s; plan files cannot be transferred between different Terraform versions", rawPlan.TerraformVersion, version.String())
	}

	plan := &plans.Plan{
		VariableValues: map[string]plans.DynamicValue{},
		Changes: &plans.ChangesSrc{
			Outputs:   []*plans.OutputChangeSrc{},
			Resources: []*plans.ResourceInstanceChangeSrc{},
		},
		DriftedResources:  []*plans.ResourceInstanceChangeSrc{},
		DeferredResources: []*plans.DeferredResourceInstanceChangeSrc{},
		Checks:            &states.CheckResults{},
	}

	plan.Applyable = rawPlan.Applyable
	plan.Complete = rawPlan.Complete
	plan.Errored = rawPlan.Errored

	plan.UIMode, err = planproto.FromMode(rawPlan.UiMode)
	if err != nil {
		return nil, err
	}

	for _, rawOC := range rawPlan.OutputChanges {
		name := rawOC.Name
		change, err := changeFromTfplan(rawOC.Change)
		if err != nil {
			return nil, fmt.Errorf("invalid plan for output %q: %s", name, err)
		}

		plan.Changes.Outputs = append(plan.Changes.Outputs, &plans.OutputChangeSrc{
			// All output values saved in the plan file are root module outputs,
			// since we don't retain others. (They can be easily recomputed
			// during apply).
			Addr:      addrs.OutputValue{Name: name}.Absolute(addrs.RootModuleInstance),
			ChangeSrc: *change,
			Sensitive: rawOC.Sensitive,
		})
	}

	checkResults, err := CheckResultsFromPlanProto(rawPlan.CheckResults)
	if err != nil {
		return nil, fmt.Errorf("failed to decode check results: %s", err)
	}
	plan.Checks = checkResults

	for _, rawRC := range rawPlan.ResourceChanges {
		change, err := resourceChangeFromTfplan(rawRC, addrs.ParseAbsResourceInstanceStr)
		if err != nil {
			// errors from resourceChangeFromTfplan already include context
			return nil, err
		}

		plan.Changes.Resources = append(plan.Changes.Resources, change)
	}

	for _, rawRC := range rawPlan.ResourceDrift {
		change, err := resourceChangeFromTfplan(rawRC, addrs.ParseAbsResourceInstanceStr)
		if err != nil {
			// errors from resourceChangeFromTfplan already include context
			return nil, err
		}

		plan.DriftedResources = append(plan.DriftedResources, change)
	}

	for _, rawDC := range rawPlan.DeferredChanges {
		change, err := deferredChangeFromTfplan(rawDC)
		if err != nil {
			return nil, err
		}

		plan.DeferredResources = append(plan.DeferredResources, change)
	}

	for _, rawRA := range rawPlan.RelevantAttributes {
		ra, err := resourceAttrFromTfplan(rawRA)
		if err != nil {
			return nil, err
		}
		plan.RelevantAttributes = append(plan.RelevantAttributes, ra)
	}

	for _, rawTargetAddr := range rawPlan.TargetAddrs {
		target, diags := addrs.ParseTargetStr(rawTargetAddr)
		if diags.HasErrors() {
			return nil, fmt.Errorf("plan contains invalid target address %q: %s", target, diags.Err())
		}
		plan.TargetAddrs = append(plan.TargetAddrs, target.Subject)
	}

	for _, rawReplaceAddr := range rawPlan.ForceReplaceAddrs {
		addr, diags := addrs.ParseAbsResourceInstanceStr(rawReplaceAddr)
		if diags.HasErrors() {
			return nil, fmt.Errorf("plan contains invalid force-replace address %q: %s", addr, diags.Err())
		}
		plan.ForceReplaceAddrs = append(plan.ForceReplaceAddrs, addr)
	}

	for name, rawVal := range rawPlan.Variables {
		val, err := valueFromTfplan(rawVal)
		if err != nil {
			return nil, fmt.Errorf("invalid value for input variable %q: %s", name, err)
		}
		plan.VariableValues[name] = val
	}

	if len(rawPlan.ApplyTimeVariables) != 0 {
		plan.ApplyTimeVariables = collections.NewSetCmp[string]()
		for _, name := range rawPlan.ApplyTimeVariables {
			plan.ApplyTimeVariables.Add(name)
		}
	}

	for _, hash := range rawPlan.ProviderFunctionResults {
		plan.ProviderFunctionResults = append(plan.ProviderFunctionResults,
			providers.FunctionHash{
				Key:    hash.Key,
				Result: hash.Result,
			},
		)
	}

	if rawBackend := rawPlan.Backend; rawBackend == nil {
		return nil, fmt.Errorf("plan file has no backend settings; backend settings are required")
	} else {
		config, err := valueFromTfplan(rawBackend.Config)
		if err != nil {
			return nil, fmt.Errorf("plan file has invalid backend configuration: %s", err)
		}
		plan.Backend = plans.Backend{
			Type:      rawBackend.Type,
			Config:    config,
			Workspace: rawBackend.Workspace,
		}
	}

	if plan.Timestamp, err = time.Parse(time.RFC3339, rawPlan.Timestamp); err != nil {
		return nil, fmt.Errorf("invalid value for timestamp %s: %s", rawPlan.Timestamp, err)
	}

	return plan, nil
}

// ResourceChangeFromProto decodes an isolated resource instance change from
// its representation as a protocol buffers message.
//
// This is used by the stackplan package, which includes planproto messages
// in its own wire format while using a different overall container.
func ResourceChangeFromProto(rawChange *planproto.ResourceInstanceChange) (*plans.ResourceInstanceChangeSrc, error) {
	return resourceChangeFromTfplan(rawChange, addrs.ParseAbsResourceInstanceStr)
}

// DeferredResourceChangeFromProto decodes an isolated deferred resource
// instance change from its representation as a protocol buffers message.
//
// This the same as ResourceChangeFromProto but internally allows for splat
// addresses, which are not allowed outside deferred changes.
func DeferredResourceChangeFromProto(rawChange *planproto.ResourceInstanceChange) (*plans.ResourceInstanceChangeSrc, error) {
	return resourceChangeFromTfplan(rawChange, addrs.ParsePartialResourceInstanceStr)
}

func resourceChangeFromTfplan(rawChange *planproto.ResourceInstanceChange, parseAddr func(str string) (addrs.AbsResourceInstance, tfdiags.Diagnostics)) (*plans.ResourceInstanceChangeSrc, error) {
	if rawChange == nil {
		// Should never happen in practice, since protobuf can't represent
		// a nil value in a list.
		return nil, fmt.Errorf("resource change object is absent")
	}

	ret := &plans.ResourceInstanceChangeSrc{}

	if rawChange.Addr == "" {
		// If "Addr" isn't populated then seems likely that this is a plan
		// file created by an earlier version of Terraform, which had the
		// same information spread over various other fields:
		// ModulePath, Mode, Name, Type, and InstanceKey.
		return nil, fmt.Errorf("no instance address for resource instance change; perhaps this plan was created by a different version of Terraform?")
	}

	instAddr, diags := parseAddr(rawChange.Addr)
	if diags.HasErrors() {
		return nil, fmt.Errorf("invalid resource instance address %q: %w", rawChange.Addr, diags.Err())
	}
	prevRunAddr := instAddr
	if rawChange.PrevRunAddr != "" {
		prevRunAddr, diags = parseAddr(rawChange.PrevRunAddr)
		if diags.HasErrors() {
			return nil, fmt.Errorf("invalid resource instance previous run address %q: %w", rawChange.PrevRunAddr, diags.Err())
		}
	}

	providerAddr, diags := addrs.ParseAbsProviderConfigStr(rawChange.Provider)
	if diags.HasErrors() {
		return nil, diags.Err()
	}
	ret.ProviderAddr = providerAddr

	ret.Addr = instAddr
	ret.PrevRunAddr = prevRunAddr

	if rawChange.DeposedKey != "" {
		if len(rawChange.DeposedKey) != 8 {
			return nil, fmt.Errorf("deposed object for %s has invalid deposed key %q", ret.Addr, rawChange.DeposedKey)
		}
		ret.DeposedKey = states.DeposedKey(rawChange.DeposedKey)
	}

	ret.RequiredReplace = cty.NewPathSet()
	for _, p := range rawChange.RequiredReplace {
		path, err := pathFromTfplan(p)
		if err != nil {
			return nil, fmt.Errorf("invalid path in required replace: %s", err)
		}
		ret.RequiredReplace.Add(path)
	}

	change, err := changeFromTfplan(rawChange.Change)
	if err != nil {
		return nil, fmt.Errorf("invalid plan for resource %s: %s", ret.Addr, err)
	}

	ret.ChangeSrc = *change

	switch rawChange.ActionReason {
	case planproto.ResourceInstanceActionReason_NONE:
		ret.ActionReason = plans.ResourceInstanceChangeNoReason
	case planproto.ResourceInstanceActionReason_REPLACE_BECAUSE_CANNOT_UPDATE:
		ret.ActionReason = plans.ResourceInstanceReplaceBecauseCannotUpdate
	case planproto.ResourceInstanceActionReason_REPLACE_BECAUSE_TAINTED:
		ret.ActionReason = plans.ResourceInstanceReplaceBecauseTainted
	case planproto.ResourceInstanceActionReason_REPLACE_BY_REQUEST:
		ret.ActionReason = plans.ResourceInstanceReplaceByRequest
	case planproto.ResourceInstanceActionReason_REPLACE_BY_TRIGGERS:
		ret.ActionReason = plans.ResourceInstanceReplaceByTriggers
	case planproto.ResourceInstanceActionReason_DELETE_BECAUSE_NO_RESOURCE_CONFIG:
		ret.ActionReason = plans.ResourceInstanceDeleteBecauseNoResourceConfig
	case planproto.ResourceInstanceActionReason_DELETE_BECAUSE_WRONG_REPETITION:
		ret.ActionReason = plans.ResourceInstanceDeleteBecauseWrongRepetition
	case planproto.ResourceInstanceActionReason_DELETE_BECAUSE_COUNT_INDEX:
		ret.ActionReason = plans.ResourceInstanceDeleteBecauseCountIndex
	case planproto.ResourceInstanceActionReason_DELETE_BECAUSE_EACH_KEY:
		ret.ActionReason = plans.ResourceInstanceDeleteBecauseEachKey
	case planproto.ResourceInstanceActionReason_DELETE_BECAUSE_NO_MODULE:
		ret.ActionReason = plans.ResourceInstanceDeleteBecauseNoModule
	case planproto.ResourceInstanceActionReason_READ_BECAUSE_CONFIG_UNKNOWN:
		ret.ActionReason = plans.ResourceInstanceReadBecauseConfigUnknown
	case planproto.ResourceInstanceActionReason_READ_BECAUSE_DEPENDENCY_PENDING:
		ret.ActionReason = plans.ResourceInstanceReadBecauseDependencyPending
	case planproto.ResourceInstanceActionReason_READ_BECAUSE_CHECK_NESTED:
		ret.ActionReason = plans.ResourceInstanceReadBecauseCheckNested
	case planproto.ResourceInstanceActionReason_DELETE_BECAUSE_NO_MOVE_TARGET:
		ret.ActionReason = plans.ResourceInstanceDeleteBecauseNoMoveTarget
	default:
		return nil, fmt.Errorf("resource has invalid action reason %s", rawChange.ActionReason)
	}

	if len(rawChange.Private) != 0 {
		ret.Private = rawChange.Private
	}

	return ret, nil
}

// ActionFromProto translates from the protobuf representation of change actions
// into the "plans" package's representation, or returns an error if the
// given action is unrecognized.
func ActionFromProto(rawAction planproto.Action) (plans.Action, error) {
	switch rawAction {
	case planproto.Action_NOOP:
		return plans.NoOp, nil
	case planproto.Action_CREATE:
		return plans.Create, nil
	case planproto.Action_READ:
		return plans.Read, nil
	case planproto.Action_UPDATE:
		return plans.Update, nil
	case planproto.Action_DELETE:
		return plans.Delete, nil
	case planproto.Action_CREATE_THEN_DELETE:
		return plans.CreateThenDelete, nil
	case planproto.Action_DELETE_THEN_CREATE:
		return plans.DeleteThenCreate, nil
	case planproto.Action_FORGET:
		return plans.Forget, nil
	case planproto.Action_CREATE_THEN_FORGET:
		return plans.CreateThenForget, nil
	default:
		return plans.NoOp, fmt.Errorf("invalid change action %s", rawAction)
	}

}

func changeFromTfplan(rawChange *planproto.Change) (*plans.ChangeSrc, error) {
	if rawChange == nil {
		return nil, fmt.Errorf("change object is absent")
	}

	ret := &plans.ChangeSrc{}

	// -1 indicates that there is no index. We'll customize these below
	// depending on the change action, and then decode.
	beforeIdx, afterIdx := -1, -1

	var err error
	ret.Action, err = ActionFromProto(rawChange.Action)
	if err != nil {
		return nil, err
	}

	switch ret.Action {
	case plans.NoOp:
		beforeIdx = 0
		afterIdx = 0
	case plans.Create:
		afterIdx = 0
	case plans.Read:
		beforeIdx = 0
		afterIdx = 1
	case plans.Update:
		beforeIdx = 0
		afterIdx = 1
	case plans.Delete:
		beforeIdx = 0
	case plans.CreateThenDelete:
		beforeIdx = 0
		afterIdx = 1
	case plans.DeleteThenCreate:
		beforeIdx = 0
		afterIdx = 1
	case plans.Forget:
		beforeIdx = 0
	case plans.CreateThenForget:
		beforeIdx = 0
		afterIdx = 1
	default:
		return nil, fmt.Errorf("invalid change action %s", rawChange.Action)
	}

	if beforeIdx != -1 {
		if l := len(rawChange.Values); l <= beforeIdx {
			return nil, fmt.Errorf("incorrect number of values (%d) for %s change", l, rawChange.Action)
		}
		var err error
		ret.Before, err = valueFromTfplan(rawChange.Values[beforeIdx])
		if err != nil {
			return nil, fmt.Errorf("invalid \"before\" value: %s", err)
		}
		if ret.Before == nil {
			return nil, fmt.Errorf("missing \"before\" value: %s", err)
		}
	}
	if afterIdx != -1 {
		if l := len(rawChange.Values); l <= afterIdx {
			return nil, fmt.Errorf("incorrect number of values (%d) for %s change", l, rawChange.Action)
		}
		var err error
		ret.After, err = valueFromTfplan(rawChange.Values[afterIdx])
		if err != nil {
			return nil, fmt.Errorf("invalid \"after\" value: %s", err)
		}
		if ret.After == nil {
			return nil, fmt.Errorf("missing \"after\" value: %s", err)
		}
	}

	if rawChange.Importing != nil {
		ret.Importing = &plans.ImportingSrc{
			ID:      rawChange.Importing.Id,
			Unknown: rawChange.Importing.Unknown,
		}
	}
	ret.GeneratedConfig = rawChange.GeneratedConfig

	beforeValSensitiveAttrs, err := pathsFromTfplan(rawChange.BeforeSensitivePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to decode before sensitive paths: %s", err)
	}
	afterValSensitiveAttrs, err := pathsFromTfplan(rawChange.AfterSensitivePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to decode after sensitive paths: %s", err)
	}
	if len(beforeValSensitiveAttrs) > 0 {
		ret.BeforeSensitivePaths = beforeValSensitiveAttrs
	}
	if len(afterValSensitiveAttrs) > 0 {
		ret.AfterSensitivePaths = afterValSensitiveAttrs
	}

	return ret, nil
}

func valueFromTfplan(rawV *planproto.DynamicValue) (plans.DynamicValue, error) {
	if len(rawV.Msgpack) == 0 { // len(0) because that's the default value for a "bytes" in protobuf
		return nil, fmt.Errorf("dynamic value does not have msgpack serialization")
	}

	return plans.DynamicValue(rawV.Msgpack), nil
}

func deferredChangeFromTfplan(dc *planproto.DeferredResourceInstanceChange) (*plans.DeferredResourceInstanceChangeSrc, error) {
	if dc == nil {
		return nil, fmt.Errorf("deferred change object is absent")
	}

	change, err := resourceChangeFromTfplan(dc.Change, addrs.ParsePartialResourceInstanceStr)
	if err != nil {
		return nil, err
	}

	reason, err := DeferredReasonFromProto(dc.Deferred.Reason)
	if err != nil {
		return nil, err
	}

	return &plans.DeferredResourceInstanceChangeSrc{
		DeferredReason: reason,
		ChangeSrc:      change,
	}, nil
}

func DeferredReasonFromProto(reason planproto.DeferredReason) (providers.DeferredReason, error) {
	switch reason {
	case planproto.DeferredReason_INSTANCE_COUNT_UNKNOWN:
		return providers.DeferredReasonInstanceCountUnknown, nil
	case planproto.DeferredReason_RESOURCE_CONFIG_UNKNOWN:
		return providers.DeferredReasonResourceConfigUnknown, nil
	case planproto.DeferredReason_PROVIDER_CONFIG_UNKNOWN:
		return providers.DeferredReasonProviderConfigUnknown, nil
	case planproto.DeferredReason_ABSENT_PREREQ:
		return providers.DeferredReasonAbsentPrereq, nil
	case planproto.DeferredReason_DEFERRED_PREREQ:
		return providers.DeferredReasonDeferredPrereq, nil
	default:
		return providers.DeferredReasonInvalid, fmt.Errorf("invalid deferred reason %s", reason)
	}
}

// writeTfplan serializes the given plan into the protobuf-based format used
// for the "tfplan" portion of a plan file.
func writeTfplan(plan *plans.Plan, w io.Writer) error {
	if plan == nil {
		return fmt.Errorf("cannot write plan file for nil plan")
	}
	if plan.Changes == nil {
		return fmt.Errorf("cannot write plan file with nil changeset")
	}

	rawPlan := &planproto.Plan{
		Version:          tfplanFormatVersion,
		TerraformVersion: version.String(),

		Variables:       map[string]*planproto.DynamicValue{},
		OutputChanges:   []*planproto.OutputChange{},
		CheckResults:    []*planproto.CheckResults{},
		ResourceChanges: []*planproto.ResourceInstanceChange{},
		ResourceDrift:   []*planproto.ResourceInstanceChange{},
		DeferredChanges: []*planproto.DeferredResourceInstanceChange{},
	}

	rawPlan.Applyable = plan.Applyable
	rawPlan.Complete = plan.Complete
	rawPlan.Errored = plan.Errored

	var err error
	rawPlan.UiMode, err = planproto.NewMode(plan.UIMode)
	if err != nil {
		return err
	}

	for _, oc := range plan.Changes.Outputs {
		// When serializing a plan we only retain the root outputs, since
		// changes to these are externally-visible side effects (e.g. via
		// terraform_remote_state).
		if !oc.Addr.Module.IsRoot() {
			continue
		}

		name := oc.Addr.OutputValue.Name

		// Writing outputs as cty.DynamicPseudoType forces the stored values
		// to also contain dynamic type information, so we can recover the
		// original type when we read the values back in readTFPlan.
		protoChange, err := changeToTfplan(&oc.ChangeSrc)
		if err != nil {
			return fmt.Errorf("cannot write output value %q: %s", name, err)
		}

		rawPlan.OutputChanges = append(rawPlan.OutputChanges, &planproto.OutputChange{
			Name:      name,
			Change:    protoChange,
			Sensitive: oc.Sensitive,
		})
	}

	checkResults, err := CheckResultsToPlanProto(plan.Checks)
	if err != nil {
		return fmt.Errorf("failed to encode check results: %s", err)
	}
	rawPlan.CheckResults = checkResults

	for _, rc := range plan.Changes.Resources {
		rawRC, err := resourceChangeToTfplan(rc)
		if err != nil {
			return err
		}
		rawPlan.ResourceChanges = append(rawPlan.ResourceChanges, rawRC)
	}

	for _, rc := range plan.DriftedResources {
		rawRC, err := resourceChangeToTfplan(rc)
		if err != nil {
			return err
		}
		rawPlan.ResourceDrift = append(rawPlan.ResourceDrift, rawRC)
	}

	for _, dc := range plan.DeferredResources {
		rawDC, err := deferredChangeToTfplan(dc)
		if err != nil {
			return err
		}
		rawPlan.DeferredChanges = append(rawPlan.DeferredChanges, rawDC)
	}

	for _, ra := range plan.RelevantAttributes {
		rawRA, err := resourceAttrToTfplan(ra)
		if err != nil {
			return err
		}
		rawPlan.RelevantAttributes = append(rawPlan.RelevantAttributes, rawRA)
	}

	for _, targetAddr := range plan.TargetAddrs {
		rawPlan.TargetAddrs = append(rawPlan.TargetAddrs, targetAddr.String())
	}

	for _, replaceAddr := range plan.ForceReplaceAddrs {
		rawPlan.ForceReplaceAddrs = append(rawPlan.ForceReplaceAddrs, replaceAddr.String())
	}

	for name, val := range plan.VariableValues {
		rawPlan.Variables[name] = valueToTfplan(val)
	}
	if plan.ApplyTimeVariables.Len() != 0 {
		rawPlan.ApplyTimeVariables = make([]string, 0, plan.ApplyTimeVariables.Len())
		for name := range plan.ApplyTimeVariables.All() {
			rawPlan.ApplyTimeVariables = append(rawPlan.ApplyTimeVariables, name)
		}
	}

	for _, hash := range plan.ProviderFunctionResults {
		rawPlan.ProviderFunctionResults = append(rawPlan.ProviderFunctionResults,
			&planproto.ProviderFunctionCallHash{
				Key:    hash.Key,
				Result: hash.Result,
			},
		)
	}

	if plan.Backend.Type == "" || plan.Backend.Config == nil {
		// This suggests a bug in the code that created the plan, since it
		// ought to always have a backend populated, even if it's the default
		// "local" backend with a local state file.
		return fmt.Errorf("plan does not have a backend configuration")
	}

	rawPlan.Backend = &planproto.Backend{
		Type:      plan.Backend.Type,
		Config:    valueToTfplan(plan.Backend.Config),
		Workspace: plan.Backend.Workspace,
	}

	rawPlan.Timestamp = plan.Timestamp.Format(time.RFC3339)

	src, err := proto.Marshal(rawPlan)
	if err != nil {
		return fmt.Errorf("serialization error: %s", err)
	}

	_, err = w.Write(src)
	if err != nil {
		return fmt.Errorf("failed to write plan to plan file: %s", err)
	}

	return nil
}

func resourceAttrToTfplan(ra globalref.ResourceAttr) (*planproto.PlanResourceAttr, error) {
	res := &planproto.PlanResourceAttr{}

	res.Resource = ra.Resource.String()
	attr, err := pathToTfplan(ra.Attr)
	if err != nil {
		return res, err
	}
	res.Attr = attr
	return res, nil
}

func resourceAttrFromTfplan(ra *planproto.PlanResourceAttr) (globalref.ResourceAttr, error) {
	var res globalref.ResourceAttr
	if ra.Resource == "" {
		return res, fmt.Errorf("missing resource address from relevant attribute")
	}

	instAddr, diags := addrs.ParseAbsResourceInstanceStr(ra.Resource)
	if diags.HasErrors() {
		return res, fmt.Errorf("invalid resource instance address %q in relevant attributes: %w", ra.Resource, diags.Err())
	}

	res.Resource = instAddr
	path, err := pathFromTfplan(ra.Attr)
	if err != nil {
		return res, fmt.Errorf("invalid path in %q relevant attribute: %s", res.Resource, err)
	}

	res.Attr = path
	return res, nil
}

// ResourceChangeToProto encodes an isolated resource instance change into
// its representation as a protocol buffers message.
//
// This is used by the stackplan package, which includes planproto messages
// in its own wire format while using a different overall container.
func ResourceChangeToProto(change *plans.ResourceInstanceChangeSrc) (*planproto.ResourceInstanceChange, error) {
	if change == nil {
		// We assume this represents the absense of a change, then.
		return nil, nil
	}
	return resourceChangeToTfplan(change)
}

func resourceChangeToTfplan(change *plans.ResourceInstanceChangeSrc) (*planproto.ResourceInstanceChange, error) {
	ret := &planproto.ResourceInstanceChange{}

	if change.PrevRunAddr.Resource.Resource.Type == "" {
		// Suggests that an old caller wasn't yet updated to populate this
		// properly. All code that generates plans should populate this field,
		// even if it's just to write in the same value as in change.Addr.
		change.PrevRunAddr = change.Addr
	}

	ret.Addr = change.Addr.String()
	ret.PrevRunAddr = change.PrevRunAddr.String()
	if ret.PrevRunAddr == ret.Addr {
		// In the on-disk format we leave PrevRunAddr unpopulated in the common
		// case where it's the same as Addr, and then fill it back in again on
		// read.
		ret.PrevRunAddr = ""
	}

	ret.DeposedKey = string(change.DeposedKey)
	ret.Provider = change.ProviderAddr.String()

	requiredReplace := change.RequiredReplace.List()
	ret.RequiredReplace = make([]*planproto.Path, 0, len(requiredReplace))
	for _, p := range requiredReplace {
		path, err := pathToTfplan(p)
		if err != nil {
			return nil, fmt.Errorf("invalid path in required replace: %s", err)
		}
		ret.RequiredReplace = append(ret.RequiredReplace, path)
	}

	valChange, err := changeToTfplan(&change.ChangeSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize resource %s change: %s", change.Addr, err)
	}
	ret.Change = valChange

	switch change.ActionReason {
	case plans.ResourceInstanceChangeNoReason:
		ret.ActionReason = planproto.ResourceInstanceActionReason_NONE
	case plans.ResourceInstanceReplaceBecauseCannotUpdate:
		ret.ActionReason = planproto.ResourceInstanceActionReason_REPLACE_BECAUSE_CANNOT_UPDATE
	case plans.ResourceInstanceReplaceBecauseTainted:
		ret.ActionReason = planproto.ResourceInstanceActionReason_REPLACE_BECAUSE_TAINTED
	case plans.ResourceInstanceReplaceByRequest:
		ret.ActionReason = planproto.ResourceInstanceActionReason_REPLACE_BY_REQUEST
	case plans.ResourceInstanceReplaceByTriggers:
		ret.ActionReason = planproto.ResourceInstanceActionReason_REPLACE_BY_TRIGGERS
	case plans.ResourceInstanceDeleteBecauseNoResourceConfig:
		ret.ActionReason = planproto.ResourceInstanceActionReason_DELETE_BECAUSE_NO_RESOURCE_CONFIG
	case plans.ResourceInstanceDeleteBecauseWrongRepetition:
		ret.ActionReason = planproto.ResourceInstanceActionReason_DELETE_BECAUSE_WRONG_REPETITION
	case plans.ResourceInstanceDeleteBecauseCountIndex:
		ret.ActionReason = planproto.ResourceInstanceActionReason_DELETE_BECAUSE_COUNT_INDEX
	case plans.ResourceInstanceDeleteBecauseEachKey:
		ret.ActionReason = planproto.ResourceInstanceActionReason_DELETE_BECAUSE_EACH_KEY
	case plans.ResourceInstanceDeleteBecauseNoModule:
		ret.ActionReason = planproto.ResourceInstanceActionReason_DELETE_BECAUSE_NO_MODULE
	case plans.ResourceInstanceReadBecauseConfigUnknown:
		ret.ActionReason = planproto.ResourceInstanceActionReason_READ_BECAUSE_CONFIG_UNKNOWN
	case plans.ResourceInstanceReadBecauseDependencyPending:
		ret.ActionReason = planproto.ResourceInstanceActionReason_READ_BECAUSE_DEPENDENCY_PENDING
	case plans.ResourceInstanceReadBecauseCheckNested:
		ret.ActionReason = planproto.ResourceInstanceActionReason_READ_BECAUSE_CHECK_NESTED
	case plans.ResourceInstanceDeleteBecauseNoMoveTarget:
		ret.ActionReason = planproto.ResourceInstanceActionReason_DELETE_BECAUSE_NO_MOVE_TARGET
	default:
		return nil, fmt.Errorf("resource %s has unsupported action reason %s", change.Addr, change.ActionReason)
	}

	if len(change.Private) > 0 {
		ret.Private = change.Private
	}

	return ret, nil
}

// ActionToProto translates from the "plans" package's representation of change
// actions into the protobuf representation, or returns an error if the
// given action is unrecognized.
func ActionToProto(action plans.Action) (planproto.Action, error) {
	switch action {
	case plans.NoOp:
		return planproto.Action_NOOP, nil
	case plans.Create:
		return planproto.Action_CREATE, nil
	case plans.Read:
		return planproto.Action_READ, nil
	case plans.Update:
		return planproto.Action_UPDATE, nil
	case plans.Delete:
		return planproto.Action_DELETE, nil
	case plans.DeleteThenCreate:
		return planproto.Action_DELETE_THEN_CREATE, nil
	case plans.CreateThenDelete:
		return planproto.Action_CREATE_THEN_DELETE, nil
	case plans.Forget:
		return planproto.Action_FORGET, nil
	case plans.CreateThenForget:
		return planproto.Action_CREATE_THEN_FORGET, nil
	default:
		return planproto.Action_NOOP, fmt.Errorf("invalid change action %s", action)
	}
}

func changeToTfplan(change *plans.ChangeSrc) (*planproto.Change, error) {
	ret := &planproto.Change{}

	before := valueToTfplan(change.Before)
	after := valueToTfplan(change.After)

	beforeSensitivePaths, err := pathsToTfplan(change.BeforeSensitivePaths)
	if err != nil {
		return nil, err
	}
	afterSensitivePaths, err := pathsToTfplan(change.AfterSensitivePaths)
	if err != nil {
		return nil, err
	}
	ret.BeforeSensitivePaths = beforeSensitivePaths
	ret.AfterSensitivePaths = afterSensitivePaths

	if change.Importing != nil {
		ret.Importing = &planproto.Importing{
			Id:      change.Importing.ID,
			Unknown: change.Importing.Unknown,
		}

	}
	ret.GeneratedConfig = change.GeneratedConfig

	ret.Action, err = ActionToProto(change.Action)
	if err != nil {
		return nil, err
	}

	switch ret.Action {
	case planproto.Action_NOOP:
		ret.Values = []*planproto.DynamicValue{before} // before and after should be identical
	case planproto.Action_CREATE:
		ret.Values = []*planproto.DynamicValue{after}
	case planproto.Action_READ:
		ret.Values = []*planproto.DynamicValue{before, after}
	case planproto.Action_UPDATE:
		ret.Values = []*planproto.DynamicValue{before, after}
	case planproto.Action_DELETE:
		ret.Values = []*planproto.DynamicValue{before}
	case planproto.Action_DELETE_THEN_CREATE:
		ret.Values = []*planproto.DynamicValue{before, after}
	case planproto.Action_CREATE_THEN_DELETE:
		ret.Values = []*planproto.DynamicValue{before, after}
	case planproto.Action_FORGET:
		ret.Values = []*planproto.DynamicValue{before}
	case planproto.Action_CREATE_THEN_FORGET:
		ret.Values = []*planproto.DynamicValue{before, after}
	default:
		return nil, fmt.Errorf("invalid change action %s", change.Action)
	}

	return ret, nil
}

func valueToTfplan(val plans.DynamicValue) *planproto.DynamicValue {
	return planproto.NewPlanDynamicValue(val)
}

func pathsFromTfplan(paths []*planproto.Path) ([]cty.Path, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	ret := make([]cty.Path, 0, len(paths))
	for _, p := range paths {
		path, err := pathFromTfplan(p)
		if err != nil {
			return nil, err
		}
		ret = append(ret, path)
	}
	return ret, nil
}

func pathsToTfplan(paths []cty.Path) ([]*planproto.Path, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	ret := make([]*planproto.Path, 0, len(paths))
	for _, p := range paths {
		path, err := pathToTfplan(p)
		if err != nil {
			return nil, err
		}
		ret = append(ret, path)
	}
	return ret, nil
}

// PathFromProto decodes a path to a nested attribute into a cty.Path for
// use in tracking marked values.
//
// This is used by the stackstate package, which uses planproto.Path messages
// while using a different overall container.
func PathFromProto(path *planproto.Path) (cty.Path, error) {
	if path == nil {
		return nil, nil
	}
	return pathFromTfplan(path)
}

func pathFromTfplan(path *planproto.Path) (cty.Path, error) {
	ret := make([]cty.PathStep, 0, len(path.Steps))
	for _, step := range path.Steps {
		switch s := step.Selector.(type) {
		case *planproto.Path_Step_ElementKey:
			dynamicVal, err := valueFromTfplan(s.ElementKey)
			if err != nil {
				return nil, fmt.Errorf("error decoding path index step: %s", err)
			}
			ty, err := dynamicVal.ImpliedType()
			if err != nil {
				return nil, fmt.Errorf("error determining path index type: %s", err)
			}
			val, err := dynamicVal.Decode(ty)
			if err != nil {
				return nil, fmt.Errorf("error decoding path index value: %s", err)
			}
			ret = append(ret, cty.IndexStep{Key: val})
		case *planproto.Path_Step_AttributeName:
			ret = append(ret, cty.GetAttrStep{Name: s.AttributeName})
		default:
			return nil, fmt.Errorf("Unsupported path step %t", step.Selector)
		}
	}
	return ret, nil
}

func pathToTfplan(path cty.Path) (*planproto.Path, error) {
	return planproto.NewPath(path)
}

func deferredChangeToTfplan(dc *plans.DeferredResourceInstanceChangeSrc) (*planproto.DeferredResourceInstanceChange, error) {
	change, err := resourceChangeToTfplan(dc.ChangeSrc)
	if err != nil {
		return nil, err
	}

	reason, err := DeferredReasonToProto(dc.DeferredReason)
	if err != nil {
		return nil, err
	}

	return &planproto.DeferredResourceInstanceChange{
		Change: change,
		Deferred: &planproto.Deferred{
			Reason: reason,
		},
	}, nil
}

func DeferredReasonToProto(reason providers.DeferredReason) (planproto.DeferredReason, error) {
	switch reason {
	case providers.DeferredReasonInstanceCountUnknown:
		return planproto.DeferredReason_INSTANCE_COUNT_UNKNOWN, nil
	case providers.DeferredReasonResourceConfigUnknown:
		return planproto.DeferredReason_RESOURCE_CONFIG_UNKNOWN, nil
	case providers.DeferredReasonProviderConfigUnknown:
		return planproto.DeferredReason_PROVIDER_CONFIG_UNKNOWN, nil
	case providers.DeferredReasonAbsentPrereq:
		return planproto.DeferredReason_ABSENT_PREREQ, nil
	case providers.DeferredReasonDeferredPrereq:
		return planproto.DeferredReason_DEFERRED_PREREQ, nil
	default:
		return planproto.DeferredReason_INVALID, fmt.Errorf("invalid deferred reason %s", reason)
	}
}

// CheckResultsFromPlanProto decodes a slice of check results from their protobuf
// representation into the "states" package's representation.
//
// It's used by the stackplan package, which includes an identical representation
// of check results within a different overall container.
func CheckResultsFromPlanProto(proto []*planproto.CheckResults) (*states.CheckResults, error) {
	configResults := addrs.MakeMap[addrs.ConfigCheckable, *states.CheckResultAggregate]()

	for _, rawCheckResults := range proto {
		aggr := &states.CheckResultAggregate{}
		switch rawCheckResults.Status {
		case planproto.CheckResults_UNKNOWN:
			aggr.Status = checks.StatusUnknown
		case planproto.CheckResults_PASS:
			aggr.Status = checks.StatusPass
		case planproto.CheckResults_FAIL:
			aggr.Status = checks.StatusFail
		case planproto.CheckResults_ERROR:
			aggr.Status = checks.StatusError
		default:
			return nil,
				fmt.Errorf("aggregate check results for %s have unsupported status %#v",
					rawCheckResults.ConfigAddr, rawCheckResults.Status)
		}

		var objKind addrs.CheckableKind
		switch rawCheckResults.Kind {
		case planproto.CheckResults_RESOURCE:
			objKind = addrs.CheckableResource
		case planproto.CheckResults_OUTPUT_VALUE:
			objKind = addrs.CheckableOutputValue
		case planproto.CheckResults_CHECK:
			objKind = addrs.CheckableCheck
		case planproto.CheckResults_INPUT_VARIABLE:
			objKind = addrs.CheckableInputVariable
		default:
			return nil, fmt.Errorf("aggregate check results for %s have unsupported object kind %s",
				rawCheckResults.ConfigAddr, objKind)
		}

		// Some trickiness here: we only have an address parser for
		// addrs.Checkable and not for addrs.ConfigCheckable, but that's okay
		// because once we have an addrs.Checkable we can always derive an
		// addrs.ConfigCheckable from it, and a ConfigCheckable should always
		// be the same syntax as a Checkable with no index information and
		// thus we can reuse the same parser for both here.
		configAddrProxy, diags := addrs.ParseCheckableStr(objKind, rawCheckResults.ConfigAddr)
		if diags.HasErrors() {
			return nil, diags.Err()
		}
		configAddr := configAddrProxy.ConfigCheckable()
		if configAddr.String() != configAddrProxy.String() {
			// This is how we catch if the config address included index
			// information that would be allowed in a Checkable but not
			// in a ConfigCheckable.
			return nil, fmt.Errorf("invalid checkable config address %s", rawCheckResults.ConfigAddr)
		}

		aggr.ObjectResults = addrs.MakeMap[addrs.Checkable, *states.CheckResultObject]()
		for _, rawCheckResult := range rawCheckResults.Objects {
			objectAddr, diags := addrs.ParseCheckableStr(objKind, rawCheckResult.ObjectAddr)
			if diags.HasErrors() {
				return nil, diags.Err()
			}
			if !addrs.Equivalent(objectAddr.ConfigCheckable(), configAddr) {
				return nil, fmt.Errorf("checkable object %s should not be grouped under %s", objectAddr, configAddr)
			}

			obj := &states.CheckResultObject{
				FailureMessages: rawCheckResult.FailureMessages,
			}
			switch rawCheckResult.Status {
			case planproto.CheckResults_UNKNOWN:
				obj.Status = checks.StatusUnknown
			case planproto.CheckResults_PASS:
				obj.Status = checks.StatusPass
			case planproto.CheckResults_FAIL:
				obj.Status = checks.StatusFail
			case planproto.CheckResults_ERROR:
				obj.Status = checks.StatusError
			default:
				return nil, fmt.Errorf("object check results for %s has unsupported status %#v",
					rawCheckResult.ObjectAddr, rawCheckResult.Status)
			}

			aggr.ObjectResults.Put(objectAddr, obj)
		}

		// If we ended up with no elements in the map then we'll just nil it,
		// primarily just to make life easier for our round-trip tests.
		if aggr.ObjectResults.Len() == 0 {
			aggr.ObjectResults.Elems = nil
		}

		configResults.Put(configAddr, aggr)
	}

	// If we ended up with no elements in the map then we'll just nil it,
	// primarily just to make life easier for our round-trip tests.
	if configResults.Len() == 0 {
		configResults.Elems = nil
	}

	return &states.CheckResults{
		ConfigResults: configResults,
	}, nil
}

// CheckResultsToPlanProto encodes a slice of check results from the "states"
// package's representation into their protobuf representation.
//
// It's used by the stackplan package, which includes identical representation
// of check results within a different overall container.
func CheckResultsToPlanProto(checkResults *states.CheckResults) ([]*planproto.CheckResults, error) {
	if checkResults != nil {
		protoResults := make([]*planproto.CheckResults, 0)
		for _, configElem := range checkResults.ConfigResults.Elems {
			crs := configElem.Value
			pcrs := &planproto.CheckResults{
				ConfigAddr: configElem.Key.String(),
			}
			switch crs.Status {
			case checks.StatusUnknown:
				pcrs.Status = planproto.CheckResults_UNKNOWN
			case checks.StatusPass:
				pcrs.Status = planproto.CheckResults_PASS
			case checks.StatusFail:
				pcrs.Status = planproto.CheckResults_FAIL
			case checks.StatusError:
				pcrs.Status = planproto.CheckResults_ERROR
			default:
				return nil,
					fmt.Errorf("checkable configuration %s has unsupported aggregate status %s", configElem.Key, crs.Status)
			}
			switch kind := configElem.Key.CheckableKind(); kind {
			case addrs.CheckableResource:
				pcrs.Kind = planproto.CheckResults_RESOURCE
			case addrs.CheckableOutputValue:
				pcrs.Kind = planproto.CheckResults_OUTPUT_VALUE
			case addrs.CheckableCheck:
				pcrs.Kind = planproto.CheckResults_CHECK
			case addrs.CheckableInputVariable:
				pcrs.Kind = planproto.CheckResults_INPUT_VARIABLE
			default:
				return nil,
					fmt.Errorf("checkable configuration %s has unsupported object type kind %s", configElem.Key, kind)
			}

			for _, objectElem := range configElem.Value.ObjectResults.Elems {
				cr := objectElem.Value
				pcr := &planproto.CheckResults_ObjectResult{
					ObjectAddr:      objectElem.Key.String(),
					FailureMessages: objectElem.Value.FailureMessages,
				}
				switch cr.Status {
				case checks.StatusUnknown:
					pcr.Status = planproto.CheckResults_UNKNOWN
				case checks.StatusPass:
					pcr.Status = planproto.CheckResults_PASS
				case checks.StatusFail:
					pcr.Status = planproto.CheckResults_FAIL
				case checks.StatusError:
					pcr.Status = planproto.CheckResults_ERROR
				default:
					return nil,
						fmt.Errorf("checkable object %s has unsupported status %s", objectElem.Key, crs.Status)
				}
				pcrs.Objects = append(pcrs.Objects, pcr)
			}

			protoResults = append(protoResults, pcrs)
		}

		return protoResults, nil
	} else {
		return nil, nil
	}
}
