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

func Test_migrate_tfc_to_tfc_single_workspace(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)
	ctx := context.Background()

	cases := map[string]struct {
		setup       func(t *testing.T) (string, func())
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"migrating from name to name": {
			setup: func(t *testing.T) (string, func()) {
				organization, cleanup := createOrganization(t)
				return organization.Name, cleanup
			},
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "prod"
						// Creating the workspace here instead of it being created
						// dynamically in the Cloud StateMgr because we want to ensure that
						// the terraform version selected for the workspace matches the
						// terraform version of this current branch.
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("prod"),
							TerraformVersion: tfe.String(tfversion.String()),
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
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `prod`, // this comes from the `prep` function
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "dev"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(wsName),
							TerraformVersion: tfe.String(tfversion.String()),
						})
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:         []string{"init", "-ignore-remote-version"},
							postInputOutput: []string{`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `dev`, // this comes from the `prep` function
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{})
				if err != nil {
					t.Fatal(err)
				}
				// this workspace name is what exists in the cloud backend configuration block
				if len(wsList.Items) != 2 {
					t.Fatal("Expected number of workspaces to be 2")
				}
			},
		},
		"migrating from name to tags": {
			setup: func(t *testing.T) (string, func()) {
				organization, cleanup := createOrganization(t)
				return organization.Name, cleanup
			},
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "prod"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("prod"),
							TerraformVersion: tfe.String(tfversion.String()),
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
							expectedCmdOutput: `There are no workspaces with the configured tags (app)`,
							userInput:         []string{"new-workspace"},
							postInputOutput: []string{
								`Terraform can create a properly tagged workspace for you now.`,
								`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `new-workspace`, // this comes from the `prep` function
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
				// this workspace name is what exists in the cloud backend configuration block
				if len(wsList.Items) != 1 {
					t.Fatal("Expected number of workspaces to be 1")
				}
			},
		},
		"migrating from name to tags without ignore-version flag": {
			setup: func(t *testing.T) (string, func()) {
				organization, cleanup := createOrganization(t)
				return organization.Name, cleanup
			},
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "prod"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("prod"),
							TerraformVersion: tfe.String(tfversion.String()),
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
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "app"
						// This is only here to ensure that the updated terraform version is
						// present in the workspace, and it does not default to a lower
						// version that does not support `cloud`.
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("new-workspace"),
							TerraformVersion: tfe.String(tfversion.String()),
						})
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `There are no workspaces with the configured tags (app)`,
							userInput:         []string{"new-workspace"},
							postInputOutput: []string{
								`Terraform can create a properly tagged workspace for you now.`,
								`Terraform Cloud has been successfully initialized!`},
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				// We created the workspace, so it will be there. We could not complete the state migration,
				// though, so the workspace should be empty.
				ws, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, "new-workspace", &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if ws.CurrentRun != nil {
					t.Fatal("Expected to workspace be empty")
				}
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
			defer tf.Close()
			tf.AddEnv(cliConfigFileEnv)

			orgName, cleanup := tc.setup(t)
			defer cleanup()
			for _, op := range tc.operations {
				op.prep(t, orgName, tf.WorkDir())
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
						t.Fatal(err.Error())
					}
				}
			}

			if tc.validations != nil {
				tc.validations(t, orgName)
			}
		})
	}
}

func Test_migrate_tfc_to_tfc_multiple_workspace(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()

	cases := map[string]struct {
		setup       func(t *testing.T) (string, func())
		operations  []operationSets
		validations func(t *testing.T, orgName string)
	}{
		"migrating from multiple workspaces via tags to name": {
			setup: func(t *testing.T) (string, func()) {
				organization, cleanup := createOrganization(t)
				return organization.Name, cleanup
			},
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "app"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("app-prod"),
							Tags:             []*tfe.Tag{{Name: tag}},
							TerraformVersion: tfe.String(tfversion.String()),
						})
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("app-staging"),
							Tags:             []*tfe.Tag{{Name: tag}},
							TerraformVersion: tfe.String(tfversion.String()),
						})
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `The currently selected workspace (default) does not exist.`,
							userInput:         []string{"1"},
							postInputOutput:   []string{`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"apply"},
							expectedCmdOutput: `Do you want to perform these actions in workspace "app-prod"?`,
							userInput:         []string{"yes"},
							postInputOutput:   []string{`Apply complete!`},
						},
						{
							command:           []string{"workspace", "select", "app-staging"},
							expectedCmdOutput: `Switched to workspace "app-staging".`,
						},
						{
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
						},
						{
							command:           []string{"output"},
							expectedCmdOutput: `tag_val = "app"`,
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						name := "service"
						// Doing this here instead of relying on dynamic workspace creation
						// because we want to set the terraform version here so that it is
						// using the right version for post init operations.
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String(name),
							TerraformVersion: tfe.String(tfversion.String()),
						})
						tfBlock := terraformConfigCloudBackendName(orgName, name)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `Terraform Cloud has been successfully initialized!`,
							postInputOutput:   []string{`tag_val = "service"`},
						},
						{
							command:           []string{"workspace", "show"},
							expectedCmdOutput: `service`, // this comes from the `prep` function
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				ws, err := tfeClient.Workspaces.Read(ctx, orgName, "service")
				if err != nil {
					t.Fatal(err)
				}
				if ws == nil {
					t.Fatal("Expected to workspace not be empty")
				}
			},
		},
		"migrating from multiple workspaces via tags to other tags": {
			setup: func(t *testing.T) (string, func()) {
				organization, cleanup := createOrganization(t)
				return organization.Name, cleanup
			},
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "app"
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("app-prod"),
							Tags:             []*tfe.Tag{{Name: tag}},
							TerraformVersion: tfe.String(tfversion.String()),
						})
						_ = createWorkspace(t, orgName, tfe.WorkspaceCreateOptions{
							Name:             tfe.String("app-staging"),
							Tags:             []*tfe.Tag{{Name: tag}},
							TerraformVersion: tfe.String(tfversion.String()),
						})
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `The currently selected workspace (default) does not exist.`,
							userInput:         []string{"1"},
							postInputOutput:   []string{`Terraform Cloud has been successfully initialized!`},
						},
						{
							command:           []string{"apply", "-auto-approve"},
							expectedCmdOutput: `Apply complete!`,
						},
						{
							command:           []string{"workspace", "select", "app-staging"},
							expectedCmdOutput: `Switched to workspace "app-staging".`,
						},
						{
							command:           []string{"apply", "-auto-approve"},
							expectedCmdOutput: `Apply complete!`,
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tag := "billing"
						tfBlock := terraformConfigCloudBackendTags(orgName, tag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init", "-ignore-remote-version"},
							expectedCmdOutput: `There are no workspaces with the configured tags (billing)`,
							userInput:         []string{"new-app-prod"},
							postInputOutput:   []string{`Terraform Cloud has been successfully initialized!`},
						},
					},
				},
			},
			validations: func(t *testing.T, orgName string) {
				wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{
					Tags: tfe.String("billing"),
				})
				if err != nil {
					t.Fatal(err)
				}
				if len(wsList.Items) != 1 {
					t.Logf("Expected the number of workspaces to be 2, but got %d", len(wsList.Items))
				}
				_, empty := getWorkspace(wsList.Items, "new-app-prod")
				if empty {
					t.Fatalf("expected workspaces to include 'new-app-prod' but didn't.")
				}
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
			defer tf.Close()
			tf.AddEnv(cliConfigFileEnv)

			orgName, cleanup := tc.setup(t)
			defer cleanup()
			for _, op := range tc.operations {
				op.prep(t, orgName, tf.WorkDir())
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
					if err != nil {
						t.Fatal(err.Error())
					}
				}
			}

			if tc.validations != nil {
				tc.validations(t, orgName)
			}
		})
	}
}
