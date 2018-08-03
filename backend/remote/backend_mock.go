package remote

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"

	tfe "github.com/hashicorp/go-tfe"
)

type mockConfigurationVersions struct {
	configVersions map[string]*tfe.ConfigurationVersion
	uploadURLs     map[string]*tfe.ConfigurationVersion
	workspaces     map[string]*tfe.ConfigurationVersion
}

func newMockConfigurationVersions() *mockConfigurationVersions {
	return &mockConfigurationVersions{
		configVersions: make(map[string]*tfe.ConfigurationVersion),
		uploadURLs:     make(map[string]*tfe.ConfigurationVersion),
		workspaces:     make(map[string]*tfe.ConfigurationVersion),
	}
}

func (m *mockConfigurationVersions) List(ctx context.Context, workspaceID string, options tfe.ConfigurationVersionListOptions) ([]*tfe.ConfigurationVersion, error) {
	var cvs []*tfe.ConfigurationVersion
	for _, cv := range m.configVersions {
		cvs = append(cvs, cv)
	}
	return cvs, nil
}

func (m *mockConfigurationVersions) Create(ctx context.Context, workspaceID string, options tfe.ConfigurationVersionCreateOptions) (*tfe.ConfigurationVersion, error) {
	id := generateID("cv-")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	cv := &tfe.ConfigurationVersion{
		ID:        id,
		Status:    tfe.ConfigurationPending,
		UploadURL: url,
	}

	m.configVersions[cv.ID] = cv
	m.uploadURLs[url] = cv
	m.workspaces[workspaceID] = cv

	return cv, nil
}

func (m *mockConfigurationVersions) Read(ctx context.Context, cvID string) (*tfe.ConfigurationVersion, error) {
	cv, ok := m.configVersions[cvID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return cv, nil
}

func (m *mockConfigurationVersions) Upload(ctx context.Context, url, path string) error {
	cv, ok := m.uploadURLs[url]
	if !ok {
		return errors.New("404 not found")
	}
	cv.Status = tfe.ConfigurationUploaded
	return nil
}

type mockOrganizations struct {
	organizations map[string]*tfe.Organization
}

func newMockOrganizations() *mockOrganizations {
	return &mockOrganizations{
		organizations: make(map[string]*tfe.Organization),
	}
}

func (m *mockOrganizations) List(ctx context.Context, options tfe.OrganizationListOptions) ([]*tfe.Organization, error) {
	var orgs []*tfe.Organization
	for _, org := range m.organizations {
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (m *mockOrganizations) Create(ctx context.Context, options tfe.OrganizationCreateOptions) (*tfe.Organization, error) {
	org := &tfe.Organization{Name: *options.Name}
	m.organizations[org.Name] = org
	return org, nil
}

func (m *mockOrganizations) Read(ctx context.Context, name string) (*tfe.Organization, error) {
	org, ok := m.organizations[name]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return org, nil
}

func (m *mockOrganizations) Update(ctx context.Context, name string, options tfe.OrganizationUpdateOptions) (*tfe.Organization, error) {
	org, ok := m.organizations[name]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	org.Name = *options.Name
	return org, nil

}

func (m *mockOrganizations) Delete(ctx context.Context, name string) error {
	delete(m.organizations, name)
	return nil
}

type mockPlans struct {
	logs  map[string]string
	plans map[string]*tfe.Plan
}

func newMockPlans() *mockPlans {
	return &mockPlans{
		logs:  make(map[string]string),
		plans: make(map[string]*tfe.Plan),
	}
}

func (m *mockPlans) Read(ctx context.Context, planID string) (*tfe.Plan, error) {
	p, ok := m.plans[planID]
	if !ok {
		url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", planID)

		p = &tfe.Plan{
			ID:         planID,
			LogReadURL: url,
			Status:     tfe.PlanFinished,
		}

		m.logs[url] = "plan/output.log"
		m.plans[p.ID] = p
	}

	return p, nil
}

func (m *mockPlans) Logs(ctx context.Context, planID string) (io.Reader, error) {
	p, err := m.Read(ctx, planID)
	if err != nil {
		return nil, err
	}

	logfile, ok := m.logs[p.LogReadURL]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	logs, err := ioutil.ReadFile("./test-fixtures/" + logfile)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(logs), nil
}

type mockRuns struct {
	runs       map[string]*tfe.Run
	workspaces map[string][]*tfe.Run
}

func newMockRuns() *mockRuns {
	return &mockRuns{
		runs:       make(map[string]*tfe.Run),
		workspaces: make(map[string][]*tfe.Run),
	}
}

func (m *mockRuns) List(ctx context.Context, workspaceID string, options tfe.RunListOptions) ([]*tfe.Run, error) {
	var rs []*tfe.Run
	for _, r := range m.workspaces[workspaceID] {
		rs = append(rs, r)
	}
	return rs, nil
}

func (m *mockRuns) Create(ctx context.Context, options tfe.RunCreateOptions) (*tfe.Run, error) {
	id := generateID("run-")
	p := &tfe.Plan{
		ID:     generateID("plan-"),
		Status: tfe.PlanPending,
	}

	r := &tfe.Run{
		ID:     id,
		Plan:   p,
		Status: tfe.RunPending,
	}

	m.runs[r.ID] = r
	m.workspaces[options.Workspace.ID] = append(m.workspaces[options.Workspace.ID], r)

	return r, nil
}

func (m *mockRuns) Read(ctx context.Context, runID string) (*tfe.Run, error) {
	r, ok := m.runs[runID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return r, nil
}

func (m *mockRuns) Apply(ctx context.Context, runID string, options tfe.RunApplyOptions) error {
	panic("not implemented")
}

func (m *mockRuns) Cancel(ctx context.Context, runID string, options tfe.RunCancelOptions) error {
	panic("not implemented")
}

func (m *mockRuns) Discard(ctx context.Context, runID string, options tfe.RunDiscardOptions) error {
	panic("not implemented")
}

type mockStateVersions struct {
	states        map[string][]byte
	stateVersions map[string]*tfe.StateVersion
	workspaces    map[string][]string
}

func newMockStateVersions() *mockStateVersions {
	return &mockStateVersions{
		states:        make(map[string][]byte),
		stateVersions: make(map[string]*tfe.StateVersion),
		workspaces:    make(map[string][]string),
	}
}

func (m *mockStateVersions) List(ctx context.Context, options tfe.StateVersionListOptions) ([]*tfe.StateVersion, error) {
	var svs []*tfe.StateVersion
	for _, sv := range m.stateVersions {
		svs = append(svs, sv)
	}
	return svs, nil
}

func (m *mockStateVersions) Create(ctx context.Context, workspaceID string, options tfe.StateVersionCreateOptions) (*tfe.StateVersion, error) {
	id := generateID("sv-")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	sv := &tfe.StateVersion{
		ID:          id,
		DownloadURL: url,
		Serial:      *options.Serial,
	}

	state, err := base64.StdEncoding.DecodeString(*options.State)
	if err != nil {
		return nil, err
	}

	m.states[sv.DownloadURL] = state
	m.stateVersions[sv.ID] = sv
	m.workspaces[workspaceID] = append(m.workspaces[workspaceID], sv.ID)

	return sv, nil
}

func (m *mockStateVersions) Read(ctx context.Context, svID string) (*tfe.StateVersion, error) {
	sv, ok := m.stateVersions[svID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return sv, nil
}

func (m *mockStateVersions) Current(ctx context.Context, workspaceID string) (*tfe.StateVersion, error) {
	svs, ok := m.workspaces[workspaceID]
	if !ok || len(svs) == 0 {
		return nil, tfe.ErrResourceNotFound
	}
	sv, ok := m.stateVersions[svs[len(svs)-1]]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return sv, nil
}

func (m *mockStateVersions) Download(ctx context.Context, url string) ([]byte, error) {
	state, ok := m.states[url]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return state, nil
}

type mockWorkspaces struct {
	workspaceIDs   map[string]*tfe.Workspace
	workspaceNames map[string]*tfe.Workspace
}

func newMockWorkspaces() *mockWorkspaces {
	return &mockWorkspaces{
		workspaceIDs:   make(map[string]*tfe.Workspace),
		workspaceNames: make(map[string]*tfe.Workspace),
	}
}

func (m *mockWorkspaces) List(ctx context.Context, organization string, options tfe.WorkspaceListOptions) ([]*tfe.Workspace, error) {
	var ws []*tfe.Workspace
	for _, w := range m.workspaceIDs {
		ws = append(ws, w)
	}
	return ws, nil
}

func (m *mockWorkspaces) Create(ctx context.Context, organization string, options tfe.WorkspaceCreateOptions) (*tfe.Workspace, error) {
	id := generateID("ws-")
	w := &tfe.Workspace{
		ID:   id,
		Name: *options.Name,
	}
	m.workspaceIDs[w.ID] = w
	m.workspaceNames[w.Name] = w
	return w, nil
}

func (m *mockWorkspaces) Read(ctx context.Context, organization, workspace string) (*tfe.Workspace, error) {
	w, ok := m.workspaceNames[workspace]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return w, nil
}

func (m *mockWorkspaces) Update(ctx context.Context, organization, workspace string, options tfe.WorkspaceUpdateOptions) (*tfe.Workspace, error) {
	w, ok := m.workspaceNames[workspace]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	w.Name = *options.Name
	w.TerraformVersion = *options.TerraformVersion

	delete(m.workspaceNames, workspace)
	m.workspaceNames[w.Name] = w

	return w, nil
}

func (m *mockWorkspaces) Delete(ctx context.Context, organization, workspace string) error {
	if w, ok := m.workspaceNames[workspace]; ok {
		delete(m.workspaceIDs, w.ID)
	}
	delete(m.workspaceNames, workspace)
	return nil
}

func (m *mockWorkspaces) Lock(ctx context.Context, workspaceID string, options tfe.WorkspaceLockOptions) (*tfe.Workspace, error) {
	panic("not implemented")
}

func (m *mockWorkspaces) Unlock(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	panic("not implemented")
}

func (m *mockWorkspaces) AssignSSHKey(ctx context.Context, workspaceID string, options tfe.WorkspaceAssignSSHKeyOptions) (*tfe.Workspace, error) {
	panic("not implemented")
}

func (m *mockWorkspaces) UnassignSSHKey(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	panic("not implemented")
}

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateID(s string) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = alphanumeric[rand.Intn(len(alphanumeric))]
	}
	return s + string(b)
}
