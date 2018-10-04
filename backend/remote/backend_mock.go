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
	"os"
	"path/filepath"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/terraform"
)

type mockClient struct {
	Applies               *mockApplies
	ConfigurationVersions *mockConfigurationVersions
	Organizations         *mockOrganizations
	Plans                 *mockPlans
	PolicyChecks          *mockPolicyChecks
	Runs                  *mockRuns
	StateVersions         *mockStateVersions
	Workspaces            *mockWorkspaces
}

func newMockClient() *mockClient {
	c := &mockClient{}
	c.Applies = newMockApplies(c)
	c.ConfigurationVersions = newMockConfigurationVersions(c)
	c.Organizations = newMockOrganizations(c)
	c.Plans = newMockPlans(c)
	c.PolicyChecks = newMockPolicyChecks(c)
	c.Runs = newMockRuns(c)
	c.StateVersions = newMockStateVersions(c)
	c.Workspaces = newMockWorkspaces(c)
	return c
}

type mockApplies struct {
	client  *mockClient
	applies map[string]*tfe.Apply
	logs    map[string]string
}

func newMockApplies(client *mockClient) *mockApplies {
	return &mockApplies{
		client:  client,
		applies: make(map[string]*tfe.Apply),
		logs:    make(map[string]string),
	}
}

// create is a helper function to create a mock apply that uses the configured
// working directory to find the logfile.
func (m *mockApplies) create(cvID, workspaceID string) (*tfe.Apply, error) {
	c, ok := m.client.ConfigurationVersions.configVersions[cvID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	if c.Speculative {
		// Speculative means its plan-only so we don't create a Apply.
		return nil, nil
	}

	id := generateID("apply-")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	a := &tfe.Apply{
		ID:         id,
		LogReadURL: url,
		Status:     tfe.ApplyPending,
	}

	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	m.logs[url] = filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"apply.log",
	)
	m.applies[a.ID] = a

	return a, nil
}

func (m *mockApplies) Read(ctx context.Context, applyID string) (*tfe.Apply, error) {
	a, ok := m.applies[applyID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	// Together with the mockLogReader this allows testing queued runs.
	if a.Status == tfe.ApplyRunning {
		a.Status = tfe.ApplyFinished
	}
	return a, nil
}

func (m *mockApplies) Logs(ctx context.Context, applyID string) (io.Reader, error) {
	a, err := m.Read(ctx, applyID)
	if err != nil {
		return nil, err
	}

	logfile, ok := m.logs[a.LogReadURL]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	if _, err := os.Stat(logfile); os.IsNotExist(err) {
		return bytes.NewBufferString("logfile does not exist"), nil
	}

	logs, err := ioutil.ReadFile(logfile)
	if err != nil {
		return nil, err
	}

	done := func() (bool, error) {
		a, err := m.Read(ctx, applyID)
		if err != nil {
			return false, err
		}
		if a.Status != tfe.ApplyFinished {
			return false, nil
		}
		return true, nil
	}

	return &mockLogReader{
		done: done,
		logs: bytes.NewBuffer(logs),
	}, nil
}

type mockConfigurationVersions struct {
	client         *mockClient
	configVersions map[string]*tfe.ConfigurationVersion
	uploadPaths    map[string]string
	uploadURLs     map[string]*tfe.ConfigurationVersion
}

func newMockConfigurationVersions(client *mockClient) *mockConfigurationVersions {
	return &mockConfigurationVersions{
		client:         client,
		configVersions: make(map[string]*tfe.ConfigurationVersion),
		uploadPaths:    make(map[string]string),
		uploadURLs:     make(map[string]*tfe.ConfigurationVersion),
	}
}

func (m *mockConfigurationVersions) List(ctx context.Context, workspaceID string, options tfe.ConfigurationVersionListOptions) (*tfe.ConfigurationVersionList, error) {
	cvl := &tfe.ConfigurationVersionList{}
	for _, cv := range m.configVersions {
		cvl.Items = append(cvl.Items, cv)
	}

	cvl.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(cvl.Items),
	}

	return cvl, nil
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
	m.uploadPaths[cv.ID] = path
	cv.Status = tfe.ConfigurationUploaded
	return nil
}

// mockInput is a mock implementation of terraform.UIInput.
type mockInput struct {
	answers map[string]string
}

func (m *mockInput) Input(opts *terraform.InputOpts) (string, error) {
	v, ok := m.answers[opts.Id]
	if !ok {
		return "", fmt.Errorf("unexpected input request in test: %s", opts.Id)
	}
	delete(m.answers, opts.Id)
	return v, nil
}

type mockOrganizations struct {
	client        *mockClient
	organizations map[string]*tfe.Organization
}

func newMockOrganizations(client *mockClient) *mockOrganizations {
	return &mockOrganizations{
		client:        client,
		organizations: make(map[string]*tfe.Organization),
	}
}

func (m *mockOrganizations) List(ctx context.Context, options tfe.OrganizationListOptions) (*tfe.OrganizationList, error) {
	orgl := &tfe.OrganizationList{}
	for _, org := range m.organizations {
		orgl.Items = append(orgl.Items, org)
	}

	orgl.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(orgl.Items),
	}

	return orgl, nil
}

// mockLogReader is a mock logreader that enables testing queued runs.
type mockLogReader struct {
	done func() (bool, error)
	logs *bytes.Buffer
}

func (m *mockLogReader) Read(l []byte) (int, error) {
	for {
		if written, err := m.read(l); err != io.ErrNoProgress {
			return written, err
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (m *mockLogReader) read(l []byte) (int, error) {
	done, err := m.done()
	if err != nil {
		return 0, err
	}
	if !done {
		return 0, io.ErrNoProgress
	}
	return m.logs.Read(l)
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
	client *mockClient
	logs   map[string]string
	plans  map[string]*tfe.Plan
}

func newMockPlans(client *mockClient) *mockPlans {
	return &mockPlans{
		client: client,
		logs:   make(map[string]string),
		plans:  make(map[string]*tfe.Plan),
	}
}

// create is a helper function to create a mock plan that uses the configured
// working directory to find the logfile.
func (m *mockPlans) create(cvID, workspaceID string) (*tfe.Plan, error) {
	id := generateID("plan-")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	p := &tfe.Plan{
		ID:         id,
		LogReadURL: url,
		Status:     tfe.PlanPending,
	}

	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	m.logs[url] = filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"plan.log",
	)
	m.plans[p.ID] = p

	return p, nil
}

func (m *mockPlans) Read(ctx context.Context, planID string) (*tfe.Plan, error) {
	p, ok := m.plans[planID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	// Together with the mockLogReader this allows testing queued runs.
	if p.Status == tfe.PlanRunning {
		p.Status = tfe.PlanFinished
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

	if _, err := os.Stat(logfile); os.IsNotExist(err) {
		return bytes.NewBufferString("logfile does not exist"), nil
	}

	logs, err := ioutil.ReadFile(logfile)
	if err != nil {
		return nil, err
	}

	done := func() (bool, error) {
		p, err := m.Read(ctx, planID)
		if err != nil {
			return false, err
		}
		if p.Status != tfe.PlanFinished {
			return false, nil
		}
		return true, nil
	}

	return &mockLogReader{
		done: done,
		logs: bytes.NewBuffer(logs),
	}, nil
}

type mockPolicyChecks struct {
	client *mockClient
	checks map[string]*tfe.PolicyCheck
	logs   map[string]string
}

func newMockPolicyChecks(client *mockClient) *mockPolicyChecks {
	return &mockPolicyChecks{
		client: client,
		checks: make(map[string]*tfe.PolicyCheck),
		logs:   make(map[string]string),
	}
}

// create is a helper function to create a mock policy check that uses the
// configured working directory to find the logfile.
func (m *mockPolicyChecks) create(cvID, workspaceID string) (*tfe.PolicyCheck, error) {
	id := generateID("pc-")

	pc := &tfe.PolicyCheck{
		ID:          id,
		Actions:     &tfe.PolicyActions{},
		Permissions: &tfe.PolicyPermissions{},
		Scope:       tfe.PolicyScopeOrganization,
		Status:      tfe.PolicyPending,
	}

	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	logfile := filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"policy.log",
	)

	if _, err := os.Stat(logfile); os.IsNotExist(err) {
		return nil, nil
	}

	m.logs[pc.ID] = logfile
	m.checks[pc.ID] = pc

	return pc, nil
}

func (m *mockPolicyChecks) List(ctx context.Context, runID string, options tfe.PolicyCheckListOptions) (*tfe.PolicyCheckList, error) {
	_, ok := m.client.Runs.runs[runID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	pcl := &tfe.PolicyCheckList{}
	for _, pc := range m.checks {
		pcl.Items = append(pcl.Items, pc)
	}

	pcl.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(pcl.Items),
	}

	return pcl, nil
}

func (m *mockPolicyChecks) Read(ctx context.Context, policyCheckID string) (*tfe.PolicyCheck, error) {
	pc, ok := m.checks[policyCheckID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	logfile, ok := m.logs[pc.ID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	if _, err := os.Stat(logfile); os.IsNotExist(err) {
		return nil, fmt.Errorf("logfile does not exist")
	}

	logs, err := ioutil.ReadFile(logfile)
	if err != nil {
		return nil, err
	}

	switch {
	case bytes.Contains(logs, []byte("Sentinel Result: true")):
		pc.Status = tfe.PolicyPasses
	case bytes.Contains(logs, []byte("Sentinel Result: false")):
		switch {
		case bytes.Contains(logs, []byte("hard-mandatory")):
			pc.Status = tfe.PolicyHardFailed
		case bytes.Contains(logs, []byte("soft-mandatory")):
			pc.Actions.IsOverridable = true
			pc.Permissions.CanOverride = true
			pc.Status = tfe.PolicySoftFailed
		}
	default:
		// As this is an unexpected state, we say the policy errored.
		pc.Status = tfe.PolicyErrored
	}

	return pc, nil
}

func (m *mockPolicyChecks) Override(ctx context.Context, policyCheckID string) (*tfe.PolicyCheck, error) {
	pc, ok := m.checks[policyCheckID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	pc.Status = tfe.PolicyOverridden
	return pc, nil
}

func (m *mockPolicyChecks) Logs(ctx context.Context, policyCheckID string) (io.Reader, error) {
	pc, ok := m.checks[policyCheckID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	logfile, ok := m.logs[pc.ID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	if _, err := os.Stat(logfile); os.IsNotExist(err) {
		return bytes.NewBufferString("logfile does not exist"), nil
	}

	logs, err := ioutil.ReadFile(logfile)
	if err != nil {
		return nil, err
	}

	switch {
	case bytes.Contains(logs, []byte("Sentinel Result: true")):
		pc.Status = tfe.PolicyPasses
	case bytes.Contains(logs, []byte("Sentinel Result: false")):
		switch {
		case bytes.Contains(logs, []byte("hard-mandatory")):
			pc.Status = tfe.PolicyHardFailed
		case bytes.Contains(logs, []byte("soft-mandatory")):
			pc.Actions.IsOverridable = true
			pc.Permissions.CanOverride = true
			pc.Status = tfe.PolicySoftFailed
		}
	default:
		// As this is an unexpected state, we say the policy errored.
		pc.Status = tfe.PolicyErrored
	}

	return bytes.NewBuffer(logs), nil
}

type mockRuns struct {
	client     *mockClient
	runs       map[string]*tfe.Run
	workspaces map[string][]*tfe.Run
}

func newMockRuns(client *mockClient) *mockRuns {
	return &mockRuns{
		client:     client,
		runs:       make(map[string]*tfe.Run),
		workspaces: make(map[string][]*tfe.Run),
	}
}

func (m *mockRuns) List(ctx context.Context, workspaceID string, options tfe.RunListOptions) (*tfe.RunList, error) {
	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	rl := &tfe.RunList{}
	for _, r := range m.workspaces[w.ID] {
		rl.Items = append(rl.Items, r)
	}

	rl.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(rl.Items),
	}

	return rl, nil
}

func (m *mockRuns) Create(ctx context.Context, options tfe.RunCreateOptions) (*tfe.Run, error) {
	a, err := m.client.Applies.create(options.ConfigurationVersion.ID, options.Workspace.ID)
	if err != nil {
		return nil, err
	}

	p, err := m.client.Plans.create(options.ConfigurationVersion.ID, options.Workspace.ID)
	if err != nil {
		return nil, err
	}

	pc, err := m.client.PolicyChecks.create(options.ConfigurationVersion.ID, options.Workspace.ID)
	if err != nil {
		return nil, err
	}

	r := &tfe.Run{
		ID:          generateID("run-"),
		Actions:     &tfe.RunActions{},
		Apply:       a,
		HasChanges:  true,
		Permissions: &tfe.RunPermissions{},
		Plan:        p,
		Status:      tfe.RunPending,
	}

	if pc != nil {
		r.PolicyChecks = []*tfe.PolicyCheck{pc}
	}

	if options.IsDestroy != nil {
		r.IsDestroy = *options.IsDestroy
	}

	logs, _ := ioutil.ReadFile(m.client.Plans.logs[p.LogReadURL])
	if r.IsDestroy || !bytes.Contains(logs, []byte("No changes. Infrastructure is up-to-date.")) {
		r.Actions.IsCancelable = true
		r.Actions.IsConfirmable = true
		r.HasChanges = true
		r.Permissions.CanApply = true
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

	pending := false
	for _, r := range m.runs {
		if r.ID != runID && r.Status == tfe.RunPending {
			pending = true
			break
		}
	}

	if !pending {
		// Only update the status if there are no other pending runs.
		r.Status = tfe.RunPlanning
		r.Plan.Status = tfe.PlanRunning
	}

	return r, nil
}

func (m *mockRuns) Apply(ctx context.Context, runID string, options tfe.RunApplyOptions) error {
	r, ok := m.runs[runID]
	if !ok {
		return tfe.ErrResourceNotFound
	}
	if r.Status != tfe.RunPending {
		// Only update the status if the run is not pending anymore.
		r.Status = tfe.RunApplying
		r.Apply.Status = tfe.ApplyRunning
	}
	return nil
}

func (m *mockRuns) Cancel(ctx context.Context, runID string, options tfe.RunCancelOptions) error {
	panic("not implemented")
}

func (m *mockRuns) Discard(ctx context.Context, runID string, options tfe.RunDiscardOptions) error {
	panic("not implemented")
}

type mockStateVersions struct {
	client        *mockClient
	states        map[string][]byte
	stateVersions map[string]*tfe.StateVersion
	workspaces    map[string][]string
}

func newMockStateVersions(client *mockClient) *mockStateVersions {
	return &mockStateVersions{
		client:        client,
		states:        make(map[string][]byte),
		stateVersions: make(map[string]*tfe.StateVersion),
		workspaces:    make(map[string][]string),
	}
}

func (m *mockStateVersions) List(ctx context.Context, options tfe.StateVersionListOptions) (*tfe.StateVersionList, error) {
	svl := &tfe.StateVersionList{}
	for _, sv := range m.stateVersions {
		svl.Items = append(svl.Items, sv)
	}

	svl.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(svl.Items),
	}

	return svl, nil
}

func (m *mockStateVersions) Create(ctx context.Context, workspaceID string, options tfe.StateVersionCreateOptions) (*tfe.StateVersion, error) {
	id := generateID("sv-")
	runID := os.Getenv("TFE_RUN_ID")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	if runID != "" && (options.Run == nil || runID != options.Run.ID) {
		return nil, fmt.Errorf("option.Run.ID does not contain the ID exported by TFE_RUN_ID")
	}

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
	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	svs, ok := m.workspaces[w.ID]
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
	client         *mockClient
	workspaceIDs   map[string]*tfe.Workspace
	workspaceNames map[string]*tfe.Workspace
}

func newMockWorkspaces(client *mockClient) *mockWorkspaces {
	return &mockWorkspaces{
		client:         client,
		workspaceIDs:   make(map[string]*tfe.Workspace),
		workspaceNames: make(map[string]*tfe.Workspace),
	}
}

func (m *mockWorkspaces) List(ctx context.Context, organization string, options tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	dummyWorkspaces := 10
	wl := &tfe.WorkspaceList{}

	// Get the prefix from the search options.
	prefix := ""
	if options.Search != nil {
		prefix = *options.Search
	}

	// Get all the workspaces that match the prefix.
	var ws []*tfe.Workspace
	for _, w := range m.workspaceIDs {
		if strings.HasPrefix(w.Name, prefix) {
			ws = append(ws, w)
		}
	}

	// Return an empty result if we have no matches.
	if len(ws) == 0 {
		wl.Pagination = &tfe.Pagination{
			CurrentPage: 1,
		}
		return wl, nil
	}

	// Return dummy workspaces for the first page to test pagination.
	if options.PageNumber <= 1 {
		for i := 0; i < dummyWorkspaces; i++ {
			wl.Items = append(wl.Items, &tfe.Workspace{
				ID:   generateID("ws-"),
				Name: fmt.Sprintf("dummy-workspace-%d", i),
			})
		}

		wl.Pagination = &tfe.Pagination{
			CurrentPage: 1,
			NextPage:    2,
			TotalPages:  2,
			TotalCount:  len(wl.Items) + len(ws),
		}

		return wl, nil
	}

	// Return the actual workspaces that matched as the second page.
	wl.Items = ws
	wl.Pagination = &tfe.Pagination{
		CurrentPage:  2,
		PreviousPage: 1,
		TotalPages:   2,
		TotalCount:   len(wl.Items) + dummyWorkspaces,
	}

	return wl, nil
}

func (m *mockWorkspaces) Create(ctx context.Context, organization string, options tfe.WorkspaceCreateOptions) (*tfe.Workspace, error) {
	w := &tfe.Workspace{
		ID:   generateID("ws-"),
		Name: *options.Name,
		Permissions: &tfe.WorkspacePermissions{
			CanQueueRun: true,
			CanUpdate:   true,
		},
	}
	if options.VCSRepo != nil {
		w.VCSRepo = &tfe.VCSRepo{}
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

	if options.Name != nil {
		w.Name = *options.Name
	}
	if options.TerraformVersion != nil {
		w.TerraformVersion = *options.TerraformVersion
	}
	if options.WorkingDirectory != nil {
		w.WorkingDirectory = *options.WorkingDirectory
	}

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
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	w.Locked = true
	return w, nil
}

func (m *mockWorkspaces) Unlock(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	w.Locked = false
	return w, nil
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
