//go:build e2e
// +build e2e

package main

import (
	"context"
	"fmt"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-uuid"
)

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
