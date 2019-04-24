package tfe

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-uuid"
)

const badIdentifier = "! / nope"

func testClient(t *testing.T) *Client {
	client, err := NewClient(nil)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func createConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	ctx := context.Background()
	cv, err := client.ConfigurationVersions.Create(
		ctx,
		w.ID,
		ConfigurationVersionCreateOptions{AutoQueueRuns: Bool(false)},
	)
	if err != nil {
		t.Fatal(err)
	}

	return cv, func() {
		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createUploadedConfigurationVersion(t *testing.T, client *Client, w *Workspace) (*ConfigurationVersion, func()) {
	cv, cvCleanup := createConfigurationVersion(t, client, w)

	ctx := context.Background()
	err := client.ConfigurationVersions.Upload(ctx, cv.UploadURL, "test-fixtures/config-version")
	if err != nil {
		cvCleanup()
		t.Fatal(err)
	}

	for i := 0; ; i++ {
		cv, err = client.ConfigurationVersions.Read(ctx, cv.ID)
		if err != nil {
			cvCleanup()
			t.Fatal(err)
		}

		if cv.Status == ConfigurationUploaded {
			break
		}

		if i > 10 {
			cvCleanup()
			t.Fatal("Timeout waiting for the configuration version to be uploaded")
		}

		time.Sleep(1 * time.Second)
	}

	return cv, cvCleanup
}

func createNotificationConfiguration(t *testing.T, client *Client, w *Workspace) (*NotificationConfiguration, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	ctx := context.Background()
	nc, err := client.NotificationConfigurations.Create(
		ctx,
		w.ID,
		NotificationConfigurationCreateOptions{
			DestinationType: NotificationDestination(NotificationDestinationTypeGeneric),
			Enabled:         Bool(false),
			Name:            String(randomString(t)),
			Token:           String(randomString(t)),
			URL:             String("http://example.com"),
			Triggers:        []string{NotificationTriggerCreated},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return nc, func() {
		if err := client.NotificationConfigurations.Delete(ctx, nc.ID); err != nil {
			t.Errorf("Error destroying notification configuration! WARNING: Dangling\n"+
				"resources may exist! The full error is shown below.\n\n"+
				"NotificationConfiguration: %s\nError: %s", nc.ID, err)
		}

		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createPolicySet(t *testing.T, client *Client, org *Organization, policies []*Policy, workspaces []*Workspace) (*PolicySet, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	ps, err := client.PolicySets.Create(ctx, org.Name, PolicySetCreateOptions{
		Name:       String(randomString(t)),
		Policies:   policies,
		Workspaces: workspaces,
	})
	if err != nil {
		t.Fatal(err)
	}

	return ps, func() {
		if err := client.PolicySets.Delete(ctx, ps.ID); err != nil {
			t.Errorf("Error destroying policy set! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"PolicySet: %s\nError: %s", ps.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createPolicy(t *testing.T, client *Client, org *Organization) (*Policy, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	name := randomString(t)
	options := PolicyCreateOptions{
		Name: String(name),
		Enforce: []*EnforcementOptions{
			{
				Path: String(name + ".sentinel"),
				Mode: EnforcementMode(EnforcementSoft),
			},
		},
	}

	ctx := context.Background()
	p, err := client.Policies.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return p, func() {
		if err := client.Policies.Delete(ctx, p.ID); err != nil {
			t.Errorf("Error destroying policy! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Policy: %s\nError: %s", p.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createUploadedPolicy(t *testing.T, client *Client, pass bool, org *Organization) (*Policy, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	p, pCleanup := createPolicy(t, client, org)

	ctx := context.Background()
	err := client.Policies.Upload(ctx, p.ID, []byte(fmt.Sprintf("main = rule { %t }", pass)))
	if err != nil {
		t.Fatal(err)
	}

	p, err = client.Policies.Read(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}

	return p, func() {
		pCleanup()

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createOAuthClient(t *testing.T, client *Client, org *Organization) (*OAuthClient, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		t.Fatal("Export a valid GITHUB_TOKEN before running this test!")
	}

	options := OAuthClientCreateOptions{
		APIURL:          String("https://api.github.com"),
		HTTPURL:         String("https://github.com"),
		OAuthToken:      String(githubToken),
		ServiceProvider: ServiceProvider(ServiceProviderGithub),
	}

	ctx := context.Background()
	oc, err := client.OAuthClients.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	// This currently panics as the token will not be there when the client is
	// created. To get a token, the client needs to be connected through the UI
	// first. So the test using this (TestOAuthTokensList) is currently disabled.
	return oc, func() {
		if err := client.OAuthClients.Delete(ctx, oc.ID); err != nil {
			t.Errorf("Error destroying OAuth client! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"OAuthClient: %s\nError: %s", oc.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createOAuthToken(t *testing.T, client *Client, org *Organization) (*OAuthToken, func()) {
	ocTest, ocTestCleanup := createOAuthClient(t, client, org)
	return ocTest.OAuthTokens[0], ocTestCleanup
}

func createOrganization(t *testing.T, client *Client) (*Organization, func()) {
	ctx := context.Background()
	org, err := client.Organizations.Create(ctx, OrganizationCreateOptions{
		Name:  String("tst-" + randomString(t)),
		Email: String(fmt.Sprintf("%s@tfe.local", randomString(t))),
	})
	if err != nil {
		t.Fatal(err)
	}

	return org, func() {
		if err := client.Organizations.Delete(ctx, org.Name); err != nil {
			t.Errorf("Error destroying organization! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Organization: %s\nError: %s", org.Name, err)
		}
	}
}

func createOrganizationToken(t *testing.T, client *Client, org *Organization) (*OrganizationToken, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	tk, err := client.OrganizationTokens.Generate(ctx, org.Name)
	if err != nil {
		t.Fatal(err)
	}

	return tk, func() {
		if err := client.OrganizationTokens.Delete(ctx, org.Name); err != nil {
			t.Errorf("Error destroying organization token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"OrganizationToken: %s\nError: %s", tk.ID, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	cv, cvCleanup := createUploadedConfigurationVersion(t, client, w)

	ctx := context.Background()
	r, err := client.Runs.Create(ctx, RunCreateOptions{
		ConfigurationVersion: cv,
		Workspace:            w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return r, func() {
		if wCleanup != nil {
			wCleanup()
		} else {
			cvCleanup()
		}
	}
}

func createPlannedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	r, rCleanup := createRun(t, client, w)

	var err error
	ctx := context.Background()
	for i := 0; ; i++ {
		r, err = client.Runs.Read(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}

		switch r.Status {
		case RunPlanned, RunCostEstimated, RunPolicyChecked, RunPolicyOverride:
			return r, rCleanup
		}

		if i > 45 {
			rCleanup()
			t.Fatal("Timeout waiting for run to be applied")
		}

		time.Sleep(1 * time.Second)
	}
}

func createAppliedRun(t *testing.T, client *Client, w *Workspace) (*Run, func()) {
	r, rCleanup := createPlannedRun(t, client, w)
	ctx := context.Background()

	err := client.Runs.Apply(ctx, r.ID, RunApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; ; i++ {
		r, err = client.Runs.Read(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}

		if r.Status == RunApplied {
			return r, rCleanup
		}

		if i > 45 {
			rCleanup()
			t.Fatal("Timeout waiting for run to be applied")
		}

		time.Sleep(1 * time.Second)
	}
}

func createSSHKey(t *testing.T, client *Client, org *Organization) (*SSHKey, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	key, err := client.SSHKeys.Create(ctx, org.Name, SSHKeyCreateOptions{
		Name:  String(randomString(t)),
		Value: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return key, func() {
		if err := client.SSHKeys.Delete(ctx, key.ID); err != nil {
			t.Errorf("Error destroying SSH key! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"SSHKey: %s\nError: %s", key.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createStateVersion(t *testing.T, client *Client, serial int64, w *Workspace) (*StateVersion, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	state, err := ioutil.ReadFile("test-fixtures/state-version/terraform.tfstate")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	_, err = client.Workspaces.Lock(ctx, w.ID, WorkspaceLockOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, err := client.Workspaces.Unlock(ctx, w.ID)
		if err != nil {
			t.Fatal(err)
		}
	}()

	sv, err := client.StateVersions.Create(ctx, w.ID, StateVersionCreateOptions{
		MD5:    String(fmt.Sprintf("%x", md5.Sum(state))),
		Serial: Int64(serial),
		State:  String(base64.StdEncoding.EncodeToString(state)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return sv, func() {
		// There currently isn't a way to delete a state, so we
		// can only cleanup by deleting the workspace.
		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createTeam(t *testing.T, client *Client, org *Organization) (*Team, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	tm, err := client.Teams.Create(ctx, org.Name, TeamCreateOptions{
		Name:               String(randomString(t)),
		OrganizationAccess: &OrganizationAccessOptions{ManagePolicies: Bool(true)},
	})
	if err != nil {
		t.Fatal(err)
	}

	return tm, func() {
		if err := client.Teams.Delete(ctx, tm.ID); err != nil {
			t.Errorf("Error destroying team! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Team: %s\nError: %s", tm.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createTeamAccess(t *testing.T, client *Client, tm *Team, w *Workspace, org *Organization) (*TeamAccess, func()) {
	var orgCleanup, tmCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, org)
	}

	if w == nil {
		w, _ = createWorkspace(t, client, org)
	}

	ctx := context.Background()
	ta, err := client.TeamAccess.Add(ctx, TeamAccessAddOptions{
		Access:    Access(AccessAdmin),
		Team:      tm,
		Workspace: w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return ta, func() {
		if err := client.TeamAccess.Remove(ctx, ta.ID); err != nil {
			t.Errorf("Error removing team access! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"TeamAccess: %s\nError: %s", ta.ID, err)
		}

		if tmCleanup != nil {
			tmCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createTeamToken(t *testing.T, client *Client, tm *Team) (*TeamToken, func()) {
	var tmCleanup func()

	if tm == nil {
		tm, tmCleanup = createTeam(t, client, nil)
	}

	ctx := context.Background()
	tt, err := client.TeamTokens.Generate(ctx, tm.ID)
	if err != nil {
		t.Fatal(err)
	}

	return tt, func() {
		if err := client.TeamTokens.Delete(ctx, tm.ID); err != nil {
			t.Errorf("Error destroying team token! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"TeamToken: %s\nError: %s", tm.ID, err)
		}

		if tmCleanup != nil {
			tmCleanup()
		}
	}
}

func createVariable(t *testing.T, client *Client, w *Workspace) (*Variable, func()) {
	var wCleanup func()

	if w == nil {
		w, wCleanup = createWorkspace(t, client, nil)
	}

	ctx := context.Background()
	v, err := client.Variables.Create(ctx, VariableCreateOptions{
		Key:       String(randomString(t)),
		Value:     String(randomString(t)),
		Category:  Category(CategoryTerraform),
		Workspace: w,
	})
	if err != nil {
		t.Fatal(err)
	}

	return v, func() {
		if err := client.Variables.Delete(ctx, v.ID); err != nil {
			t.Errorf("Error destroying variable! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Variable: %s\nError: %s", v.Key, err)
		}

		if wCleanup != nil {
			wCleanup()
		}
	}
}

func createWorkspace(t *testing.T, client *Client, org *Organization) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	ctx := context.Background()
	w, err := client.Workspaces.Create(ctx, org.Name, WorkspaceCreateOptions{
		Name: String(randomString(t)),
	})
	if err != nil {
		t.Fatal(err)
	}

	return w, func() {
		if err := client.Workspaces.Delete(ctx, org.Name, w.Name); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.Name, err)
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func createWorkspaceWithVCS(t *testing.T, client *Client, org *Organization) (*Workspace, func()) {
	var orgCleanup func()

	if org == nil {
		org, orgCleanup = createOrganization(t, client)
	}

	oc, ocCleanup := createOAuthToken(t, client, org)

	githubIdentifier := os.Getenv("GITHUB_IDENTIFIER")
	if githubIdentifier == "" {
		t.Fatal("Export a valid GITHUB_IDENTIFIER before running this test!")
	}

	options := WorkspaceCreateOptions{
		Name: String(randomString(t)),
		VCSRepo: &VCSRepoOptions{
			Identifier:   String(githubIdentifier),
			OAuthTokenID: String(oc.ID),
		},
	}

	ctx := context.Background()
	w, err := client.Workspaces.Create(ctx, org.Name, options)
	if err != nil {
		t.Fatal(err)
	}

	return w, func() {
		if err := client.Workspaces.Delete(ctx, org.Name, w.Name); err != nil {
			t.Errorf("Error destroying workspace! WARNING: Dangling resources\n"+
				"may exist! The full error is shown below.\n\n"+
				"Workspace: %s\nError: %s", w.Name, err)
		}

		if ocCleanup != nil {
			ocCleanup()
		}

		if orgCleanup != nil {
			orgCleanup()
		}
	}
}

func randomString(t *testing.T) string {
	v, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
