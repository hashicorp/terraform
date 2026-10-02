// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestDiagsFromProto_failingMembers(t *testing.T) {
	values := []*proto.ExpressionValue{
		{
			Traversal: &proto.AttributePath{Steps: []*proto.AttributePath_Step{
				{Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "mode"}},
			}},
			Value: []byte("subject"),
		},
		{
			Traversal: &proto.AttributePath{Steps: []*proto.AttributePath_Step{
				{Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "original"}},
			}},
			Value:  []byte("member"),
			Member: "local_file.readme",
		},
	}
	diags := DiagsFromProto([]*proto.Diagnostic{{
		Severity:         proto.Severity_ERROR,
		Summary:          "Condition not met",
		ExpressionValues: values,
	}}, nil).AsTerraformDiags()

	extra := tfdiags.ExtraInfo[*PolicyExtra](diags[0])
	if extra == nil {
		t.Fatalf("expected policy extra, got nil")
	}
	if diff := cmp.Diff(values, extra.ExpressionValues, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected expression values:\n%s", diff)
	}
}
