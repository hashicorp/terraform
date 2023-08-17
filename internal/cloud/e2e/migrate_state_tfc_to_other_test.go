// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"
)

func Test_migrate_tfc_to_other(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)

	cases := testCases{
		"migrate from cloud to local backend": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						tfBlock := mnptuConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `mnptu Cloud has been successfully initialized!`,
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := mnptuConfigLocalBackend()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Migrating state from mnptu Cloud to another backend is not yet implemented.`,
							expectError:       true,
						},
					},
				},
			},
		},
	}

	testRunner(t, cases, 1)
}
