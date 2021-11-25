package main

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/e2e"
	tfversion "github.com/hashicorp/terraform/version"
)

func Test_terraform_apply_autoApprove(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()

	cases := map[string]struct {
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"workspace manual apply, terraform apply without auto-approve, expect prompt": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "app"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(wsName),
							TerraformVersion: tfe.String(tfversion.String()),
							AutoApply:        tfe.Bool(false),
						})
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions in workspace "app"?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Apply complete!`},
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, "app", &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatal("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunApplied {
					t.Fatalf("Expected run status to be `applied`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
		"workspace auto apply, terraform apply without auto-approve, expect prompt": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "app"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(wsName),
							TerraformVersion: tfe.String(tfversion.String()),
							AutoApply:        tfe.Bool(true),
						})
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions in workspace "app"?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Apply complete!`},
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, "app", &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatal("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunApplied {
					t.Fatalf("Expected run status to be `applied`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
		"workspace manual apply, terraform apply with auto-approve, no prompt": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "app"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(wsName),
							TerraformVersion: tfe.String(tfversion.String()),
							AutoApply:        tfe.Bool(false),
						})
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"apply", "-auto-approve"},
							expectedCmdOutput: `Apply complete!`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, "app", &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatal("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunApplied {
					t.Fatalf("Expected run status to be `applied`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
		"workspace auto apply, terraform apply with auto-approve, no prompt": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "app"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(wsName),
							TerraformVersion: tfe.String(tfversion.String()),
							AutoApply:        tfe.Bool(true),
						})
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
						},
						{
							command:           []string{"apply", "-auto-approve"},
							expectedCmdOutput: `Apply complete!`,
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, "app", &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatal("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunApplied {
					t.Fatalf("Expected run status to be `applied`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// t.Parallel()
			organization, cleanup := createOrganization(t)
			defer cleanup()
			exp, err := expect.NewConsole(defaultOpts()...)
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
						got, err := exp.ExpectString(tfCmd.expectedCmdOutput)
						if err != nil {
							t.Fatalf("error while waiting for output\nwant: %s\nerror: %s\noutput\n%s", tfCmd.expectedCmdOutput, err, got)
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
					if err != nil && !tfCmd.expectError {
						t.Fatal(err)
					}
				}
			}

			if tc.validations != nil {
				tc.validations(t, organization.Name)
			}
		})
	}
}
