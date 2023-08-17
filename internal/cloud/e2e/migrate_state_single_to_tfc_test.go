// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
)

func Test_migrate_single_to_tfc(t *testing.T) {
	t.Parallel()
	skipIfMissingEnvVar(t)
	skipWithoutRemotemnptuVersion(t)

	ctx := context.Background()

	cases := testCases{
		"migrate using cloud workspace name strategy": {
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
							command:           []string{"init"},
							expectedCmdOutput: `Migrating from backend "local" to mnptu Cloud.`,
							userInput:         []string{"yes", "yes"},
							postInputOutput: []string{
								`Should mnptu migrate your existing state?`,
								`mnptu Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "list"},
							expectedCmdOutput: `new-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, nil)
				if err != nil {
					t.Fatal(err)
				}
				ws := wsList.Items[0]
				if ws.Name != "new-workspace" {
					t.Fatalf("Expected workspace to be `new-workspace`, but is %s", ws.Name)
				}
			},
		},
		"migrate using cloud workspace tags strategy": {
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
						tag := "app"
						tfBlock := mnptuConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Migrating from backend "local" to mnptu Cloud.`,
							userInput:         []string{"yes", "new-workspace", "yes"},
							postInputOutput: []string{
								`Should mnptu migrate your existing state?`,
								`mnptu Cloud requires all workspaces to be given an explicit name.`,
								`mnptu Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "list"},
							expectedCmdOutput: `new-workspace`,
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
				ws := wsList.Items[0]
				if ws.Name != "new-workspace" {
					t.Fatalf("Expected workspace to be `new-workspace`, but is %s", ws.Name)
				}
			},
		},
	}

	testRunner(t, cases, 1)
}
