//go:build e2e
// +build e2e

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/hashicorp/terraform/internal/e2e"
)

func Test_migrate_tfc_to_other(t *testing.T) {
	cases := map[string]struct {
		operations []operationSets
	}{
		"migrate from cloud to local backend": {
			operations: []operationSets{
				{
					prep: func(t *testing.T, orgName, dir string) {
						wsName := "new-workspace"
						tfBlock := terraformConfigCloudBackendName(orgName, wsName)
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:        []string{"init"},
							expectedOutput: `Successfully configured the backend "cloud"!`,
						},
					},
				},
				{
					prep: func(t *testing.T, orgName, dir string) {
						tfBlock := terraformConfigLocalBackend()
						writeMainTF(t, tfBlock, dir)
					},
					commands: []tfCommand{
						{
							command:        []string{"init", "-migrate-state"},
							expectedOutput: `Migrating state from Terraform Cloud to another backend is not yet implemented.`,
							expectError:    true,
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		fmt.Println("Test: ", name)
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

				if tfCmd.expectedOutput != "" {
					_, err := exp.ExpectString(tfCmd.expectedOutput)
					if err != nil {
						t.Fatal(err)
					}
				}

				if len(tfCmd.userInput) > 0 {
					for _, input := range tfCmd.userInput {
						exp.SendLine(input)
					}
				}

				if tfCmd.postInputOutput != "" {
					_, err := exp.ExpectString(tfCmd.postInputOutput)
					if err != nil {
						t.Fatal(err)
					}
				}

				err = cmd.Wait()
				if err != nil && !tfCmd.expectError {
					t.Fatal(err)
				}
			}
		}
	}
}
