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
	skipIfMissingEnvVar(t)
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
							command:         []string{"apply", "-auto-approve"},
							postInputOutput: []string{`Apply complete!`},
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
							command:           []string{"init"},
							expectedCmdOutput: `Migrating from backend "local" to Terraform Cloud.`,
							userInput:         []string{"yes", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
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
							command:           []string{"init"},
							expectedCmdOutput: `Migrating from backend "local" to Terraform Cloud.`,
							userInput:         []string{"yes", "new-workspace", "yes"},
							postInputOutput: []string{
								`Should Terraform migrate your existing state?`,
								`Terraform Cloud requires all workspaces to be given an explicit name.`,
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
					if err != nil {
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
