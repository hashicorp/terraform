// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

func resourceAttributesToProto(value PolicyValue) (*proto.ResourceAttributes, error) {
	raw, err := msgpack.Marshal(value.Raw, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("error serializing raw value: %w", err)
	}

	redactedPaths, err := pathsToProto(value.RedactedPaths)
	if err != nil {
		return nil, fmt.Errorf("error serializing redacted paths: %w", err)
	}

	return &proto.ResourceAttributes{
		Raw:           raw,
		RedactedPaths: redactedPaths,
	}, nil
}

func pathToProto(path cty.Path) (*proto.Path, error) {
	steps := make([]*proto.Path_Step, 0, len(path))
	for _, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			steps = append(steps, &proto.Path_Step{
				Selector: &proto.Path_Step_AttributeName{AttributeName: step.Name},
			})
		case cty.IndexStep:
			key, err := msgpack.Marshal(step.Key, cty.DynamicPseudoType)
			if err != nil {
				return nil, fmt.Errorf("error serializing index step key: %w", err)
			}
			steps = append(steps, &proto.Path_Step{
				Selector: &proto.Path_Step_ElementKey{ElementKey: key},
			})
		default:
			return nil, fmt.Errorf("unsupported cty path step type %T", step)
		}
	}
	return &proto.Path{Steps: steps}, nil
}

func pathsToProto(paths []cty.Path) ([]*proto.Path, error) {
	ret := make([]*proto.Path, 0, len(paths))
	for _, path := range paths {
		protoPath, err := pathToProto(path)
		if err != nil {
			return nil, err
		}
		ret = append(ret, protoPath)
	}
	return ret, nil
}
