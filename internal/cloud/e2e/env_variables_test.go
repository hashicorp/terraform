// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-tfe"
)

func Test_cloud_organization_env_var(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)

	ctx := context.Background()
	org, cleanup := createOrganization(t)
	t.Cleanup(cleanup)

	cases := testCases{
		"with TF_CLOUD_ORGANIZATION set": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						remoteWorkspace := "cloud-workspace"
						tfBlock := terraformConfigCloudBackendOmitOrg(remoteWorkspace)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "cloud-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, org.Name, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
			},
		},
	}

	testRunner(t, cases, 0, fmt.Sprintf("TF_CLOUD_ORGANIZATION=%s", org.Name))
}

func Test_cloud_workspace_name_env_var(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)

	org, orgCleanup := createOrganization(t)
	t.Cleanup(orgCleanup)

	wk := createWorkspace(t, org.Name, tfe.WorkspaceCreateOptions{
		Name: tfe.String("cloud-workspace"),
	})

	validCases := testCases{
		"a workspace that exists": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendOmitWorkspaces(org.Name)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendOmitWorkspaces(org.Name)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: wk.Name,
						},
					},
				},
			},
		},
	}

	errCases := testCases{
		"a workspace that doesn't exist": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendOmitWorkspaces(org.Name)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:     []string{"init"},
							expectError: true,
						},
					},
				},
			},
		},
	}

	testRunner(t, validCases, 0, fmt.Sprintf(`TF_WORKSPACE=%s`, wk.Name))
	testRunner(t, errCases, 0, fmt.Sprintf(`TF_WORKSPACE=%s`, "the-fires-of-mt-doom"))
}

func Test_cloud_workspace_tags_env_var(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)

	org, orgCleanup := createOrganization(t)
	t.Cleanup(orgCleanup)

	wkValid := createWorkspace(t, org.Name, tfe.WorkspaceCreateOptions{
		Name: tfe.String("cloud-workspace"),
		Tags: []*tfe.Tag{
			{Name: "cloud"},
		},
	})

	// this will be a workspace that won't have a tag listed in our test configuration
	wkInvalid := createWorkspace(t, org.Name, tfe.WorkspaceCreateOptions{
		Name: tfe.String("cloud-workspace-2"),
	})

	validCases := testCases{
		"a workspace with valid tag": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendTags(org.Name, wkValid.TagNames[0])
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendTags(org.Name, wkValid.TagNames[0])
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: wkValid.Name,
						},
					},
				},
			},
		},
	}

	errCases := testCases{
		"a workspace not specified by tags": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendTags(org.Name, wkValid.TagNames[0])
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:     []string{"init"},
							expectError: true,
						},
					},
				},
			},
		},
	}

	testRunner(t, validCases, 0, fmt.Sprintf(`TF_WORKSPACE=%s`, wkValid.Name))
	testRunner(t, errCases, 0, fmt.Sprintf(`TF_WORKSPACE=%s`, wkInvalid.Name))
}

func Test_cloud_null_config(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)

	org, cleanup := createOrganization(t)
	t.Cleanup(cleanup)

	wk := createWorkspace(t, org.Name, tfe.WorkspaceCreateOptions{
		Name: tfe.String("cloud-workspace"),
	})

	cases := testCases{
		"with all env vars set": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendOmitConfig()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigCloudBackendOmitConfig()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: wk.Name,
						},
					},
				},
			},
		},
	}

	testRunner(t, cases, 1,
		fmt.Sprintf(`TF_CLOUD_ORGANIZATION=%s`, org.Name),
		fmt.Sprintf(`TF_CLOUD_HOSTNAME=%s`, tfeHostname),
		fmt.Sprintf(`TF_WORKSPACE=%s`, wk.Name))
}
