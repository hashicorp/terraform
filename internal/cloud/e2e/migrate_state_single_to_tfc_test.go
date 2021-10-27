//go:build e2e
// +build e2e

package main

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/e2e"
)

func Test_migrate_single_to_tfc(t *testing.T) {
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()

	cases := map[string]struct {
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"migrate using cloud workspace name strategy": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigLocalBackend()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "local"!`,
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-migrate-state"},
							expectedCmdOutput: `Do you want to copy existing state to Terraform Cloud?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "list"},
							expectedCmdOutput: `new-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{})
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
						tfBlock := terraformConfigLocalBackend()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Successfully configured the backend "local"!`,
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions?`,
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
							command:           []string{"init", "-migrate-state"},
							expectedCmdOutput: `Terraform Cloud configuration only allows named workspaces!`,
							userInput:         []string{"new-workspace", "yes"},
							postInputOutput: []string{
								`Do you want to copy existing state to Terraform Cloud?`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "list"},
							expectedCmdOutput: `new-workspace`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{
					Tags: tfe.String("app"),
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

	for name, tc := range cases {
		t.Log("Test: ", name)
		organization, cleanup := createOrganization(t)
		defer cleanup()
		exp, err := expect.NewConsole(expect.WithStdout(os.Stdout), expect.WithDefaultTimeout(expectConsoleTimeout))
		if err != nil {
			t.Fatal(err)
		}
		defer exp.Close()

		tmpDir, err := ioutil.TempDir("", "terraform-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		tf := e2e.NewBinary(terraformBin, tmpDir)
		tf.AddEnv("TF_LOG=info")
		tf.AddEnv(cliConfigFileEnv)
		defer tf.Close()

		for _, op := range tc.operations {
			op.prep(t, organization.Name, tf.WorkDir())
			for _, tfCmd := range op.commands {
				cmd := tf.Cmd(tfCmd.command...)
				cmd.Stdin = exp.Tty()
				cmd.Stdout = exp.Tty()
				cmd.Stderr = exp.Tty()

				err = cmd.Start()
				if err != nil {
					t.Fatal(err)
				}

				if tfCmd.expectedCmdOutput != "" {
					_, err := exp.ExpectString(tfCmd.expectedCmdOutput)
					if err != nil {
						t.Fatal(err)
					}
				}

				lenInput := len(tfCmd.userInput)
				lenInputOutput := len(tfCmd.postInputOutput)
				if lenInput > 0 {
					for i := 0; i < lenInput; i++ {
						input := tfCmd.userInput[i]
						exp.SendLine(input)
						// use the index to find the corresponding
						// output that matches the input.
						if lenInputOutput-1 >= i {
							output := tfCmd.postInputOutput[i]
							_, err := exp.ExpectString(output)
							if err != nil {
								t.Fatal(err)
							}
						}
					}
				}

				err = cmd.Wait()
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		if tc.validations != nil {
			tc.validations(t, organization.Name)
		}
	}
}
