// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	tfversion "github.com/hashicorp/terraform/version"
)

func terraformConfigRequiredVariable(org, name string) string {
	return fmt.Sprintf(`
terraform {
  cloud {
    hostname = "%s"
    organization = "%s"

    workspaces {
      name = "%s"
    }
  }
}

variable "foo" {
  type = string
}

variable "baz" {
	type = string
}

output "test_cli" {
  value = var.foo
}

output "test_env" {
  value = var.baz
}

`, tfeHostname, org, name)
}

func Test_cloud_run_variables(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	cases := testCases{
		"run variables from CLI arg": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(wsName),
							TerraformVersion: tfe.String(tfversion.String()),
						})
						tfBlock := terraformConfigRequiredVariable(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"plan", "-var", "foo=bar"},
							expectedCmdOutput: `  + test_cli = "bar"`,
						},
						{
							command:           []string{"plan", "-var", "foo=bar"},
							expectedCmdOutput: `  + test_env = "qux"`,
						},
					},
				},
			},
		},
	}

	testRunner(t, cases, 1, "TF_CLI_ARGS=-no-color", "TF_VAR_baz=qux")
}
