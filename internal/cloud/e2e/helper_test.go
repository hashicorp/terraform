// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-uuid"
	goversion "github.com/hashicorp/go-version"
	tfversion "github.com/hashicorp/terraform/version"
)

const (
	// We need to give the console enough time to hear back.
	// 1 minute was too short in some cases, so this gives it ample time.
	expectConsoleTimeout = 3 * time.Minute
)

type tfCommand struct {
	command           []string
	expectedCmdOutput string
	expectError       bool
	userInput         []string
	postInputOutput   []string
}

type operationSets struct {
	commands []tfCommand
	prep     func(t *testing.T, orgName, dir string)
}

type testCases map[string]struct {
	operations  []operationSets
	validations func(t *testing.T, orgName string)
}

func defaultOpts() []expect.ConsoleOpt {
	opts := []expect.ConsoleOpt{
		expect.WithDefaultTimeout(expectConsoleTimeout),
	}
	if verboseMode {
		opts = append(opts, expect.WithStdout(os.Stdout))
	}
	return opts
}

func createOrganization(t *testing.T) (*tfe.Organization, func()) {
	ctx := context.Background()
	org, err := tfeClient.Organizations.Create(ctx, tfe.OrganizationCreateOptions{
		Name:                  tfe.String("tst-" + randomString(t)),
		Email:                 tfe.String(fmt.Sprintf("%s@tfe.local", randomString(t))),
		CostEstimationEnabled: tfe.Bool(false),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = tfeClient.Admin.Organizations.Update(ctx, org.Name, tfe.AdminOrganizationUpdateOptions{
		AccessBetaTools: tfe.Bool(true),
	})
	if err != nil {
		t.Fatal(err)
	}

	return org, func() {
		if err := tfeClient.Organizations.Delete(ctx, org.Name); err != nil {
			t.Errorf("Error destroying organization! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Organization: %s\nError: %s", org.Name, err)
		}
	}
}

func createWorkspace(t *testing.T, orgName string, wOpts tfe.WorkspaceCreateOptions) *tfe.Workspace {
	ctx := context.Background()
	w, err := tfeClient.Workspaces.Create(ctx, orgName, wOpts)
	if err != nil {
		t.Fatal(err)
	}

	return w
}

func getWorkspace(workspaces []*tfe.Workspace, workspace string) (*tfe.Workspace, bool) {
	for _, ws := range workspaces {
		if ws.Name == workspace {
			return ws, false
		}
	}
	return nil, true
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func terraformConfigLocalBackend() string {
	return `
terraform {
  backend "local" {
  }
}

output "val" {
  value = "${terraform.workspace}"
}
`
}

func terraformConfigRemoteBackendName(org, name string) string {
	return fmt.Sprintf(`
terraform {
  backend "remote" {
    hostname = "%s"
    organization = "%s"

    workspaces {
      name = "%s"
    }
  }
}

output "val" {
  value = "${terraform.workspace}"
}
`, tfeHostname, org, name)
}

func terraformConfigRemoteBackendPrefix(org, prefix string) string {
	return fmt.Sprintf(`
terraform {
  backend "remote" {
    hostname = "%s"
    organization = "%s"

    workspaces {
      prefix = "%s"
    }
  }
}

output "val" {
  value = "${terraform.workspace}"
}
`, tfeHostname, org, prefix)
}

func terraformConfigCloudBackendTags(org, tag string) string {
	return fmt.Sprintf(`
terraform {
  cloud {
    hostname = "%s"
    organization = "%s"

    workspaces {
      tags = ["%s"]
    }
  }
}

output "tag_val" {
  value = "%s"
}
`, tfeHostname, org, tag, tag)
}

func terraformConfigCloudBackendName(org, name string) string {
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

output "val" {
  value = "${terraform.workspace}"
}
`, tfeHostname, org, name)
}

func terraformConfigCloudBackendOmitOrg(workspaceName string) string {
	return fmt.Sprintf(`
terraform {
  cloud {
    hostname = "%s"

	workspaces {
	  name = "%s"
	}
  }
}

output "val" {
  value = "${terraform.workspace}"
}
`, tfeHostname, workspaceName)
}

func terraformConfigCloudBackendOmitWorkspaces(orgName string) string {
	return fmt.Sprintf(`
terraform {
  cloud {
    hostname = "%s"
	organization = "%s"
  }
}

output "val" {
  value = "${terraform.workspace}"
}
`, tfeHostname, orgName)
}

func terraformConfigCloudBackendOmitConfig() string {
	return `
terraform {
  cloud {}
}

output "val" {
  value = "${terraform.workspace}"
}
`
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

// The e2e tests rely on the fact that the terraform version in TFC/E is able to
// run the `cloud` configuration block, which is available in 1.1 and will
// continue to be available in later versions. So this function checks that
// there is a version that is >= 1.1.
func skipWithoutRemoteTerraformVersion(t *testing.T) {
	version := tfversion.Version
	baseVersion, err := goversion.NewVersion(version)
	if err != nil {
		t.Fatalf(fmt.Sprintf("Error instantiating go-version for %s", version))
	}
	opts := &tfe.AdminTerraformVersionsListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100,
		},
	}
	hasVersion := false

findTfVersion:
	for {
		// TODO: update go-tfe Read() to retrieve a terraform version by name.
		// Currently you can only retrieve by ID.
		tfVersionList, err := tfeClient.Admin.TerraformVersions.List(context.Background(), opts)
		if err != nil {
			t.Fatalf("Could not retrieve list of terraform versions: %v", err)
		}
		for _, item := range tfVersionList.Items {
			availableVersion, err := goversion.NewVersion(item.Version)
			if err != nil {
				t.Logf("Error instantiating go-version for %s", item.Version)
				continue
			}
			if availableVersion.Core().GreaterThanOrEqual(baseVersion.Core()) {
				hasVersion = true
				break findTfVersion
			}
		}

		// Exit the loop when we've seen all pages.
		if tfVersionList.CurrentPage >= tfVersionList.TotalPages {
			break
		}

		// Update the page number to get the next page.
		opts.PageNumber = tfVersionList.NextPage
	}

	if !hasVersion {
		t.Skipf("Skipping test because TFC/E does not have current Terraform version to test with (%s)", version)
	}
}
