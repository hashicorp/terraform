// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfstackdata1

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/states"
)

func ResourceInstanceObjectStateToTFStackData1(objSrc *states.ResourceInstanceObjectSrc, providerConfigAddr addrs.AbsProviderConfig) *StateResourceInstanceObjectV1 {
	if objSrc == nil {
		// This is presumably representing the absense of any prior state,
		// such as when an object is being planned for creation.
		return nil
	}

	// Hack: we'll borrow NewDynamicValue's treatment of the sensitive
	// attribute paths here just so we don't need to reimplement the
	// slice-of-paths conversion in yet another place. We don't
	// actually do anything with the value part of this.
	protoValue := stacks.NewDynamicValue(plans.DynamicValue(nil), objSrc.AttrSensitivePaths)
	rawMsg := &StateResourceInstanceObjectV1{
		SchemaVersion:        objSrc.SchemaVersion,
		ValueJson:            objSrc.AttrsJSON,
		SensitivePaths:       Terraform1ToPlanProtoAttributePaths(protoValue.Sensitive),
		CreateBeforeDestroy:  objSrc.CreateBeforeDestroy,
		ProviderConfigAddr:   providerConfigAddr.String(),
		ProviderSpecificData: objSrc.Private,
	}
	switch objSrc.Status {
	case states.ObjectReady:
		rawMsg.Status = StateResourceInstanceObjectV1_READY
	case states.ObjectTainted:
		rawMsg.Status = StateResourceInstanceObjectV1_DAMAGED
	default:
		rawMsg.Status = StateResourceInstanceObjectV1_UNKNOWN
	}

	rawMsg.Dependencies = make([]string, len(objSrc.Dependencies))
	for i, addr := range objSrc.Dependencies {
		rawMsg.Dependencies[i] = addr.String()
	}

	return rawMsg
}

func ComponentInstanceResultsToTFStackData1(outputValues map[addrs.OutputValue]cty.Value) (*StateComponentInstanceV1, error) {
	protoOutputs := make(map[string]*DynamicValue, len(outputValues))
	for addr, val := range outputValues {
		protoVal, err := DynamicValueToTFStackData1(val, cty.DynamicPseudoType)
		if err != nil {
			return nil, fmt.Errorf("encoding %s: %w", addr, err)
		}
		protoOutputs[addr.Name] = protoVal
	}
	return &StateComponentInstanceV1{
		OutputValues: protoOutputs,
	}, nil
}

func DynamicValueToTFStackData1(val cty.Value, ty cty.Type) (*DynamicValue, error) {
	unmarkedVal, markPaths := val.UnmarkDeepWithPaths()
	sensitivePaths, withOtherMarks := marks.PathsWithMark(markPaths, marks.Sensitive)
	if len(withOtherMarks) != 0 {
		return nil, withOtherMarks[0].Path.NewErrorf(
			"can't serialize value marked with %#v (this is a bug in Terraform)",
			withOtherMarks[0].Marks,
		)
	}

	rawVal, err := msgpack.Marshal(unmarkedVal, ty)
	if err != nil {
		return nil, err
	}
	ret := &DynamicValue{
		Value: &planproto.DynamicValue{
			Msgpack: rawVal,
		},
	}
	if len(markPaths) == 0 {
		return ret, nil
	}

	ret.SensitivePaths = make([]*planproto.Path, 0, len(markPaths))
	for _, path := range sensitivePaths {
		protoPath, err := planproto.NewPath(path)
		if err != nil {
			return nil, path.NewErrorf("failed to encode path: %w", err)
		}
		ret.SensitivePaths = append(ret.SensitivePaths, protoPath)
	}
	return ret, nil
}

func DynamicValueFromTFStackData1(protoVal *DynamicValue, ty cty.Type) (cty.Value, error) {
	raw := protoVal.Value.Msgpack

	unmarkedV, err := msgpack.Unmarshal(raw, ty)
	if err != nil {
		return cty.NilVal, err
	}

	var markses []cty.PathValueMarks
	if len(protoVal.SensitivePaths) != 0 {
		markses = make([]cty.PathValueMarks, 0, len(protoVal.SensitivePaths))
		marks := cty.NewValueMarks(marks.Sensitive)
		for _, protoPath := range protoVal.SensitivePaths {
			path, err := planfile.PathFromProto(protoPath)
			if err != nil {
				return cty.NilVal, fmt.Errorf("invalid sensitive value path: %w", err)
			}
			markses = append(markses, cty.PathValueMarks{
				Path:  path,
				Marks: marks,
			})
		}
	}
	return unmarkedV.MarkWithPaths(markses), nil
}

func Terraform1ToPlanProtoAttributePaths(paths []*stacks.AttributePath) []*planproto.Path {
	if len(paths) == 0 {
		return nil
	}
	ret := make([]*planproto.Path, len(paths))
	for i, tf1Path := range paths {
		ret[i] = Terraform1ToPlanProtoAttributePath(tf1Path)
	}
	return ret
}

func Terraform1ToPlanProtoAttributePath(path *stacks.AttributePath) *planproto.Path {
	if path == nil {
		return nil
	}
	ret := &planproto.Path{}
	if len(path.Steps) == 0 {
		return ret
	}
	ret.Steps = make([]*planproto.Path_Step, len(path.Steps))
	for i, tf1Step := range path.Steps {
		ret.Steps[i] = Terraform1ToPlanProtoAttributePathStep(tf1Step)
	}
	return ret
}

func Terraform1ToPlanProtoAttributePathStep(step *stacks.AttributePath_Step) *planproto.Path_Step {
	if step == nil {
		return nil
	}
	ret := &planproto.Path_Step{}
	switch sel := step.Selector.(type) {
	case *stacks.AttributePath_Step_AttributeName:
		ret.Selector = &planproto.Path_Step_AttributeName{
			AttributeName: sel.AttributeName,
		}
	case *stacks.AttributePath_Step_ElementKeyInt:
		encInt, err := msgpack.Marshal(cty.NumberIntVal(sel.ElementKeyInt), cty.Number)
		if err != nil {
			// This should not be possible because all integers have a cty msgpack encoding
			panic(fmt.Sprintf("unencodable element index: %s", err))
		}
		ret.Selector = &planproto.Path_Step_ElementKey{
			ElementKey: &planproto.DynamicValue{
				Msgpack: encInt,
			},
		}
	case *stacks.AttributePath_Step_ElementKeyString:
		encStr, err := msgpack.Marshal(cty.StringVal(sel.ElementKeyString), cty.String)
		if err != nil {
			// This should not be possible because all strings have a cty msgpack encoding
			panic(fmt.Sprintf("unencodable element key: %s", err))
		}
		ret.Selector = &planproto.Path_Step_ElementKey{
			ElementKey: &planproto.DynamicValue{
				Msgpack: encStr,
			},
		}
	default:
		// Should not get here, because the above cases should be exhaustive
		// for all possible *terraform1.AttributePath_Step selector types.
		panic(fmt.Sprintf("unsupported path step selector type %T", sel))
	}
	return ret
}
