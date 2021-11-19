package main

import (
	"io/ioutil"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/hashicorp/terraform/internal/e2e"
)

func Test_init_with_empty_tags(t *testing.T) {
	skipIfMissingEnvVar(t)
	// t.Parallel()
	skipWithoutRemoteTerraformVersion(t)

	cases := map[string]struct {
		operations []operationSets
	}{
		"terraform init with cloud block - no tagged workspaces exist yet": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsTag := "emptytag"
						tfBlock := terraformConfigCloudBackendTags(orgName, wsTag)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:           []string{"init"},
							expectedCmdOutput: `There are no workspaces with the configured tags`,
							userInput:         []string{"emptytag-prod"},
							postInputOutput:   []string{`Terraform Cloud has been successfully initialized!`},
						},
					},
				},
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
		})
	}
}
