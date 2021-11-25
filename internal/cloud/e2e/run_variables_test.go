package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/e2e"
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

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
			tf.AddEnv("TF_CLI_ARGS=-no-color")
			tf.AddEnv("TF_VAR_baz=qux")
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
									t.Fatalf(`Expected command output "%s", but got %v `, tfCmd.expectedCmdOutput, err)
								}
							}
						}
					}

					err = cmd.Wait()
					if err != nil && !tfCmd.expectError {
						t.Fatal(err)
					}
				}

				if tc.validations != nil {
					tc.validations(t, organization.Name)
				}
			}
		})
	}
}
