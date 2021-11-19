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

func Test_migrate_remote_backend_name_to_tfc_name(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		expectedName := "cloud-workspace"
		ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
		if err != nil {
			t.Fatal(err)
		}
		if ws == nil {
			t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
		}
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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

	if validations != nil {
		validations(t, organization.Name)
	}
}

func Test_migrate_remote_backend_name_to_tfc_same_name(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)
	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		expectedName := "remote-workspace"
		ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
		if err != nil {
			t.Fatal(err)
		}
		if ws == nil {
			t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
		}
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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

	if validations != nil {
		validations(t, organization.Name)
	}
}

func Test_migrate_remote_backend_name_to_tfc_name_different_org(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		expectedName := "remote-workspace"
		ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
		if err != nil {
			t.Fatal(err)
		}
		if ws == nil {
			t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
		}
	}

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

	orgOne, cleanupOne := createOrganization(t)
	orgTwo, cleanupTwo := createOrganization(t)
	defer cleanupOne()
	defer cleanupTwo()
	orgs := []string{orgOne.Name, orgTwo.Name}
	var orgName string
	for index, op := range operations {
		orgName = orgs[index]
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
				t.Fatal(err)
			}
		}
	}

	if validations != nil {
		validations(t, orgName)
	}
}

func Test_migrate_remote_backend_name_to_tfc_tags(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{
			Tags: tfe.String("app"),
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
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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

	if validations != nil {
		validations(t, organization.Name)
	}
}

func Test_migrate_remote_backend_prefix_to_tfc_name_strategy_single_workspace(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		expectedName := "cloud-workspace"
		ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
		if err != nil {
			t.Fatal(err)
		}
		if ws == nil {
			t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
		}
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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
						got, err := exp.ExpectString(output)
						if err != nil {
							t.Fatalf("error while waiting for output\nwant: %s\nerror: %s\noutput\n%s", output, err, got)
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

	if validations != nil {
		validations(t, organization.Name)
	}
}

func Test_migrate_remote_backend_prefix_to_tfc_name_strategy_multi_workspace(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		expectedName := "cloud-workspace"
		ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
		if err != nil {
			t.Fatal(err)
		}
		if ws == nil {
			t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
		}
		wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{})
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
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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

	if validations != nil {
		validations(t, organization.Name)
	}
}

func Test_migrate_remote_backend_prefix_to_tfc_tags_strategy_single_workspace(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		expectedName := "cloud-workspace"
		ws, err := tfeClient.Workspaces.Read(ctx, orgName, expectedName)
		if err != nil {
			t.Fatal(err)
		}
		if ws == nil {
			t.Fatalf("Expected workspace %s to be present, but is not.", expectedName)
		}
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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

	if validations != nil {
		validations(t, organization.Name)
	}
}

func Test_migrate_remote_backend_prefix_to_tfc_tags_strategy_multi_workspace(t *testing.T) {
	skipIfMissingEnvVar(t)
	skipWithoutRemoteTerraformVersion(t)

	ctx := context.Background()
	operations := []operationSets{
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
	}
	validations := func(t *testing.T, orgName string) {
		wsList, err := tfeClient.Workspaces.List(ctx, orgName, tfe.WorkspaceListOptions{
			Tags: tfe.String("app"),
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
	}

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

	organization, cleanup := createOrganization(t)
	defer cleanup()
	for _, op := range operations {
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

	if validations != nil {
		validations(t, organization.Name)
	}
}
