//go:build e2e
// +build e2e

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-uuid"
)

const (
	expectConsoleTimeout = 15 * time.Second
)

type tfCommand struct {
	command           []string
	expectedCmdOutput string
	expectedErr       string
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

func createOrganization(t *testing.T) (*tfe.Organization, func()) {
	ctx := context.Background()
	org, err := tfeClient.Organizations.Create(ctx, tfe.OrganizationCreateOptions{
		Name:  tfe.String("tst-" + randomString(t)),
		Email: tfe.String(fmt.Sprintf("%s@tfe.local", randomString(t))),
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

func createWorkspace(t *testing.T, org *tfe.Organization, wOpts tfe.WorkspaceCreateOptions) *tfe.Workspace {
	ctx := context.Background()
	w, err := tfeClient.Workspaces.Create(ctx, org.Name, wOpts)
	if err != nil {
		t.Fatal(err)
	}

	return w
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func terraformConfigLocalBackend() string {
	return fmt.Sprintf(`
terraform {
  backend "local" {
  }
}

output "val" {
  value = "${terraform.workspace}"
}
`)
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

resource "random_pet" "server" {
  keepers = {
    uuid = uuid()
  }

  length = 3
}
`, tfeHostname, org, tag)
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

resource "random_pet" "server" {
  keepers = {
    uuid = uuid()
  }

  length = 3
}
`, tfeHostname, org, name)
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
