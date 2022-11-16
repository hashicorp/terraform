package cloud

import (
	"context"
	"fmt"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/command/webcommand"
)

// TestCloudWebURLForObject tests the CloudWebURLForObject method of type Cloud.
func TestCloudWebURLForObject(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	orgName := b.organization
	ws, err := b.client.Workspaces.Read(context.Background(), orgName, testBackendSingleWorkspaceName)
	if err != nil {
		t.Fatalf("failed to read the mock workspace: %s", err)
	}

	// This is _just enough_ run to make this test work.
	run := &tfe.Run{
		ID: "mock-run-id",
		// NOTE: Workspace must not be a pointer to ws because ws.CurrentRun
		// will point back at this same run, which will create a circular
		// reference that interferes with the operation of the mock client.
		Workspace: &tfe.Workspace{
			ID:   ws.ID,
			Name: ws.Name,
			Organization: &tfe.Organization{
				Name: ws.Organization.Name,
			},
		},
	}
	b.client.Runs.(*MockRuns).Runs[run.ID] = run

	// Modifying "ws" here works because the mock client returns a pointer
	// directly to the object its mock workspace table refers to. Future
	// calls to b.client.Workspaces.Read will expose these modifications.
	ws.CurrentRun = run

	withoutCurrentRunName := "without-current-run"
	_, err = b.client.Workspaces.Create(context.Background(), orgName, tfe.WorkspaceCreateOptions{
		Name: &withoutCurrentRunName,
	})
	if err != nil {
		t.Fatalf("failed to create the without-current-run workspace: %s", err)
	}

	tests := []struct {
		workspaceName   string
		targetObject    webcommand.TargetObject
		wantURL         string
		wantDiagSummary string
		wantErr         bool
	}{
		{
			workspaceName:   testBackendSingleWorkspaceName,
			targetObject:    webcommand.TargetObjectCurrentWorkspace,
			wantURL:         "https://app.terraform.io/app/" + orgName + "/workspaces/" + testBackendSingleWorkspaceName,
			wantDiagSummary: ``,
			wantErr:         false,
		},
		{
			workspaceName:   "nonexist",
			targetObject:    webcommand.TargetObjectCurrentWorkspace,
			wantURL:         "",
			wantDiagSummary: `Failed to fetch Terraform Cloud workspace`,
			wantErr:         true,
		},
		{
			workspaceName:   testBackendSingleWorkspaceName,
			targetObject:    webcommand.TargetObjectLatestRun,
			wantURL:         "https://app.terraform.io/app/" + orgName + "/workspaces/" + testBackendSingleWorkspaceName + "/runs/" + run.ID,
			wantDiagSummary: ``,
			wantErr:         false,
		},
		{
			workspaceName:   withoutCurrentRunName,
			targetObject:    webcommand.TargetObjectLatestRun,
			wantURL:         "",
			wantDiagSummary: `No Current Run for Workspace`,
			wantErr:         true,
		},
		{
			workspaceName:   testBackendSingleWorkspaceName,
			targetObject:    webcommand.TargetObjectRun{RunID: run.ID},
			wantURL:         "https://app.terraform.io/app/" + orgName + "/workspaces/" + testBackendSingleWorkspaceName + "/runs/" + run.ID,
			wantDiagSummary: ``,
			wantErr:         false,
		},
		{
			workspaceName:   testBackendSingleWorkspaceName,
			targetObject:    webcommand.TargetObjectRun{RunID: "nonexist"},
			wantURL:         "",
			wantDiagSummary: `Failed to Fetch Requested Run`,
			wantErr:         true,
		},
	}

	for _, test := range tests {
		t.Run(
			fmt.Sprintf("%s with %s", test.targetObject.UIDescription(), test.workspaceName),
			func(t *testing.T) {
				gotURL, diags := b.WebURLForObject(context.Background(), test.workspaceName, test.targetObject)

				if test.wantErr != diags.HasErrors() {
					if test.wantErr {
						t.Errorf("succeeded; want error")
					} else {
						t.Errorf("failed; want success\n%s", diags.Err().Error())
					}
				}

				var gotURLStr string
				if gotURL != nil {
					gotURLStr = gotURL.String()
				}
				if gotURLStr != test.wantURL {
					t.Errorf("wrong result\ngot:  %s\nwant: %s", gotURLStr, test.wantURL)
				}

				if test.wantDiagSummary != "" {
					found := false
					for _, diag := range diags {
						if diag.Description().Summary == test.wantDiagSummary {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("missing expected diagnostic with summary %q", test.wantDiagSummary)
					}
				}
			},
		)
	}
}
