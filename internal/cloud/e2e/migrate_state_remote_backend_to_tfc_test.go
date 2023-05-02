// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
)

func Test_migrate_remote_backend_single_org(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	cases := testCases{
		"migrate remote backend name to tfc name": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						remoteWorkspace := "remote-workspace"
						tfBlock := terraformConfigRemoteBackendName(orgName, remoteWorkspace)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "remote"!`,
						},
						{
							command:           []string{"apply", "-auto-approve"},
							expectedCmdOutput: `Apply complete!`,
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "cloud-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Migrating from backend "remote" to Terraform Cloud.`,
							userInput:         []string{"yes", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `cloud-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "cloud-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
			},
		},
		"migrate remote backend name to tfc same name": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						remoteWorkspace := "remote-workspace"
						tfBlock := terraformConfigRemoteBackendName(orgName, remoteWorkspace)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "remote"!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "remote-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Migrating from backend "remote" to Terraform Cloud.`,
							userInput:         []string{"yes", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `remote-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "remote-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
			},
		},
		"migrate remote backend name to tfc tags": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						remoteWorkspace := "remote-workspace"
						tfBlock := terraformConfigRemoteBackendName(orgName, remoteWorkspace)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "remote"!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `default`,
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "app"
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Migrating from backend "remote" to Terraform Cloud.`,
							userInput:         []string{"yes", "cloud-workspace", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud requires all workspaces to be given an explicit name.`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `cloud-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, &tfe.WorkspaceListOptions{
					Tags: "app",
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(wsList.Items) != 1 {
					t.Fatalf("Expected number of workspaces to be 1, but got %d", len(wsList.Items))
				}
				ws := wsList.Items[0]
				if ws.Name != "cloud-workspace" {
					t.Fatalf("Expected workspace to be `cloud-workspace`, but is %s", ws.Name)
				}
			},
		},
		"migrate remote backend prefix to tfc name strategy single workspace": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{Name: tfe.String("app-one")})
						prefix := "app-"
						tfBlock := terraformConfigRemoteBackendPrefix(orgName, prefix)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform has been successfully initialized!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "cloud-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Migrating from backend "remote" to Terraform Cloud.`,
							userInput:         []string{"yes", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `cloud-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "cloud-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
			},
		},
		"migrate remote backend prefix to tfc name strategy multi workspace": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{Name: tfe.String("app-one")})
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{Name: tfe.String("app-two")})
						prefix := "app-"
						tfBlock := terraformConfigRemoteBackendPrefix(orgName, prefix)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `The currently selected workspace (default) does not exist.`,
							userInput:         []string{"1"},
							postInputOutput:   []string{`Terraform has been successfully initialized!`},
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
						{
							command:           []string{"workspace", "list"},
							expectedCmdOutput: "* one", // app name retrieved via prefix
						},
						{
							command:           []string{"workspace", "select", "two"},
							expectedCmdOutput: `Switched to workspace "two".`, // app name retrieved via prefix
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "cloud-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Do you want to copy only your current workspace?`,
							userInput:         []string{"yes"},
							postInputOutput: []string{
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `cloud-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "cloud-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, nil)
				if err != nil {
					t.Fatal(err)
				}
				if len(wsList.Items) != 3 {
					t.Fatalf("expected number of workspaces in this org to be 3, but got %d", len(wsList.Items))
				}
				_, empty := getWorkspace(wsList.Items, "cloud-workspace")
				if empty {
					t.Fatalf("expected workspaces to include 'cloud-workspace' but didn't.")
				}
				_, empty = getWorkspace(wsList.Items, "app-one")
				if empty {
					t.Fatalf("expected workspaces to include 'app-one' but didn't.")
				}
				_, empty = getWorkspace(wsList.Items, "app-two")
				if empty {
					t.Fatalf("expected workspaces to include 'app-two' but didn't.")
				}
			},
		},
		"migrate remote backend prefix to tfc tags strategy single workspace": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{Name: tfe.String("app-one")})
						prefix := "app-"
						tfBlock := terraformConfigRemoteBackendPrefix(orgName, prefix)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform has been successfully initialized!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "app"
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Migrating from backend "remote" to Terraform Cloud.`,
							userInput:         []string{"yes", "cloud-workspace", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud requires all workspaces to be given an explicit name.`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "list"},
							expectedCmdOutput: `cloud-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "cloud-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
			},
		},
		"migrate remote backend prefix to tfc tags strategy multi workspace": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{Name: tfe.String("app-one")})
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{Name: tfe.String("app-two")})
						prefix := "app-"
						tfBlock := terraformConfigRemoteBackendPrefix(orgName, prefix)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `The currently selected workspace (default) does not exist.`,
							userInput:         []string{"1"},
							postInputOutput:   []string{`Terraform has been successfully initialized!`},
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions in workspace "app-one"?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Apply complete!`},
						},
						{
							command: []string{"workspace", "select", "two"},
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions in workspace "app-two"?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "app"
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Do you wish to proceed?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: "app-two",
						},
						{
							command:           []string{"workspace", "select", "app-one"},
							expectedCmdOutput: `Switched to workspace "app-one".`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, &tfe.WorkspaceListOptions{
					Tags: "app",
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(wsList.Items) != 2 {
					t.Logf("Expected the number of workspaces to be 2, but got %d", len(wsList.Items))
				}
				ws, empty := getWorkspace(wsList.Items, "app-one")
				if empty {
					t.Fatalf("expected workspaces to include 'app-one' but didn't.")
				}
				if len(ws.TagNames) == 0 {
					t.Fatalf("expected workspaces 'one' to have tags.")
				}
				ws, empty = getWorkspace(wsList.Items, "app-two")
				if empty {
					t.Fatalf("expected workspaces to include 'app-two' but didn't.")
				}
				if len(ws.TagNames) == 0 {
					t.Fatalf("expected workspaces 'app-two' to have tags.")
				}
			},
		},
	}

	testRunner(t, cases, 1)
}

func Test_migrate_remote_backend_multi_org(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	cases := testCases{
		"migrate remote backend name to tfc name": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						remoteWorkspace := "remote-workspace"
						tfBlock := terraformConfigRemoteBackendName(orgName, remoteWorkspace)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "remote"!`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "remote-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Migrating from backend "remote" to Terraform Cloud.`,
							userInput:         []string{"yes", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `remote-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				expectedName := "remote-workspace"
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
				}
			},
		},
	}

	testRunner(t, cases, 2)
}
