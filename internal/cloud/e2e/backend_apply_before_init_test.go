// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"
)

func Test_backend_apply_before_init(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)
	skipWithoutRemotemnptuVersion(t)

	cases := testCases{
		"mnptu apply with cloud block - blank state": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						tfBlock := mnptuConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"apply"},
							expectedCmdOutput: `mnptu Cloud initialization required: please run "mnptu init"`,
							expectError:       true,
						},
					},
				},
			},
		},
		"mnptu apply with cloud block - local state": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := mnptuConfigLocalBackend()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "local"!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						tfBlock := mnptuConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"apply"},
							expectedCmdOutput: `mnptu Cloud initialization required: please run "mnptu init"`,
							expectError:       true,
						},
					},
				},
			},
		},
	}

	testRunner(t, cases, 1)
}
