// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"testing"
)

func Test_apply_no_input_flag(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)

	cases := testCases{
		"terraform apply with -input=false": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-input=false"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized`,
						},
						{
							command:           []string{"apply", "-input=false"},
							expectedCmdOutput: `Cannot confirm apply due to -input=false. Please handle run confirmation in the UI.`,
							expectError:       true,
						},
					},
				},
			},
		},
		"terraform apply with auto approve and -input=false": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "cloud-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-input=false"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized`,
						},
						{
							command:           []string{"apply", "-auto-approve", "-input=false"},
							expectedCmdOutput: `Apply complete!`,
						},
					},
				},
			},
		},
	}

	testRunner(t, cases, 1)
}
