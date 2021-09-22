//go:build e2e
// +build e2e

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/e2e"
)

func Test_terraform_apply_autoApprove(t *testing.T) {
	ctx := context.Background()
	cases := map[string]struct {
		setup       func(t *testing.T) (map[string]string, func())
		commands    []tfCommand
		validations func(t *testing.T, orgName, wsName string)
	}{
		"workspace manual apply, terraform apply without auto-approve": {
			setup: func(t *testing.T) (map[string]string, func()) {
				org, orgCleanup := createOrganization(t)
				wOpts := tfe.WorkspaceCreateOptions{
					Name:             tfe.String(randomString(t)),
					TerraformVersion: tfe.String(terraformVersion),
					AutoApply:        tfe.Bool(false),
				}
				workspace := createWorkspace(t, org, wOpts)
				cleanup := func() {
					defer orgCleanup()
				}
				names := map[string]string{
					"organization": org.Name,
					"workspace":    workspace.Name,
				}

				return names, cleanup
			},
			commands: []tfCommand{
				{
					command:        []string{"init"},
					expectedOutput: "Terraform has been successfully initialized",
					expectedErr:    "",
				},
				{
					command:        []string{"apply"},
					expectedOutput: "Do you want to perform these actions in workspace",
					expectedErr:    "Error asking approve",
				},
			},
			validations: func(t *testing.T, orgName, wsName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, wsName, &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatal("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunPlanned {
					t.Fatalf("Expected run status to be `planned`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
		"workspace auto apply, terraform apply without auto-approve": {
			setup: func(t *testing.T) (map[string]string, func()) {
				org, orgCleanup := createOrganization(t)
				wOpts := tfe.WorkspaceCreateOptions{
					Name:             tfe.String(randomString(t)),
					TerraformVersion: tfe.String(terraformVersion),
					AutoApply:        tfe.Bool(true),
				}
				workspace := createWorkspace(t, org, wOpts)
				cleanup := func() {
					defer orgCleanup()
				}
				names := map[string]string{
					"organization": org.Name,
					"workspace":    workspace.Name,
				}

				return names, cleanup
			},
			commands: []tfCommand{
				{
					command:        []string{"init"},
					expectedOutput: "Terraform has been successfully initialized",
					expectedErr:    "",
				},
				{
					command:        []string{"apply"},
					expectedOutput: "Do you want to perform these actions in workspace",
					expectedErr:    "Error asking approve",
				},
			},
			validations: func(t *testing.T, orgName, wsName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, wsName, &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatalf("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunPlanned {
					t.Fatalf("Expected run status to be `planned`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
		"workspace manual apply, terraform apply auto-approve": {
			setup: func(t *testing.T) (map[string]string, func()) {
				org, orgCleanup := createOrganization(t)
				wOpts := tfe.WorkspaceCreateOptions{
					Name:             tfe.String(randomString(t)),
					TerraformVersion: tfe.String(terraformVersion),
					AutoApply:        tfe.Bool(false),
				}
				workspace := createWorkspace(t, org, wOpts)
				cleanup := func() {
					defer orgCleanup()
				}
				names := map[string]string{
					"organization": org.Name,
					"workspace":    workspace.Name,
				}

				return names, cleanup
			},
			commands: []tfCommand{
				{
					command:        []string{"init"},
					expectedOutput: "Terraform has been successfully initialized",
					expectedErr:    "",
				},
				{
					command:        []string{"apply", "-auto-approve"},
					expectedOutput: "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.",
					expectedErr:    "",
				},
			},
			validations: func(t *testing.T, orgName, wsName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, wsName, &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatalf("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunApplied {
					t.Fatalf("Expected run status to be `applied`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
		"workspace auto apply, terraform apply auto-approve": {
			setup: func(t *testing.T) (map[string]string, func()) {
				org, orgCleanup := createOrganization(t)

				wOpts := tfe.WorkspaceCreateOptions{
					Name:             tfe.String(randomString(t)),
					TerraformVersion: tfe.String(terraformVersion),
					AutoApply:        tfe.Bool(true),
				}
				workspace := createWorkspace(t, org, wOpts)
				cleanup := func() {
					defer orgCleanup()
				}
				names := map[string]string{
					"organization": org.Name,
					"workspace":    workspace.Name,
				}

				return names, cleanup
			},
			commands: []tfCommand{
				{
					command:        []string{"init"},
					expectedOutput: "Terraform has been successfully initialized",
					expectedErr:    "",
				},
				{
					command:        []string{"apply", "-auto-approve"},
					expectedOutput: "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.",
					expectedErr:    "",
				},
			},
			validations: func(t *testing.T, orgName, wsName string) {
				workspace, err := tfeClient.Workspaces.ReadWithOptions(ctx, orgName, wsName, &tfe.WorkspaceReadOptions{Include: "current_run"})
				if err != nil {
					t.Fatal(err)
				}
				if workspace.CurrentRun == nil {
					t.Fatalf("Expected workspace to have run, but got nil")
				}
				if workspace.CurrentRun.Status != tfe.RunApplied {
					t.Fatalf("Expected run status to be `applied`, but is %s", workspace.CurrentRun.Status)
				}
			},
		},
	}
	for name, tc := range cases {
		log.Println("Test: ", name)
		resourceData, cleanup := tc.setup(t)
		defer cleanup()

		tmpDir, err := ioutil.TempDir("", "terraform-test")
		if err != nil {
			t.Fatal(err)
		}
		orgName := resourceData["organization"]
		wsName := resourceData["workspace"]
		tfBlock := terraformConfigCloudBackendName(orgName, wsName)
		writeMainTF(t, tfBlock, tmpDir)
		tf := e2e.NewBinary(terraformBin, tmpDir)
		defer tf.Close()
		tf.AddEnv("TF_LOG=debug")
		tf.AddEnv(cliConfigFileEnv)

		for _, cmd := range tc.commands {
			stdout, stderr, err := tf.Run(cmd.command...)
			if cmd.expectedErr == "" && err != nil {
				t.Fatalf("Expected no error, but got %v. stderr\n: %s", err, stderr)
			}
			if cmd.expectedErr != "" {
				if !strings.Contains(stderr, cmd.expectedErr) {
					t.Fatalf("Expected to find error %s, but got %s", cmd.expectedErr, stderr)
				}
			}

			if cmd.expectedOutput != "" && !strings.Contains(stdout, cmd.expectedOutput) {
				t.Fatalf("Expected to find output %s, but did not find in\n%s", cmd.expectedOutput, stdout)
			}
		}

		tc.validations(t, orgName, wsName)
	}
}

func writeMainTF(t *testing.T, block string, dir string) {
	f, err := os.Create(fmt.Sprintf("%s/main.tf", dir))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(block)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
