// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

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
	"sync"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/mitchellh/copystructure"

	tfversion "github.com/hashicorp/terraform/version"
)

type MockClient struct {
	Applies               *MockApplies
	ConfigurationVersions *MockConfigurationVersions
	CostEstimates         *MockCostEstimates
	Organizations         *MockOrganizations
	Plans                 *MockPlans
	PolicySetOutcomes     *MockPolicySetOutcomes
	TaskStages            *MockTaskStages
	RedactedPlans         *MockRedactedPlans
	PolicyChecks          *MockPolicyChecks
	Projects              *MockProjects
	Runs                  *MockRuns
	RunEvents             *MockRunEvents
	StateVersions         *MockStateVersions
	StateVersionOutputs   *MockStateVersionOutputs
	Variables             *MockVariables
	Workspaces            *MockWorkspaces
}

func NewMockClient() *MockClient {
	c := &MockClient{}
	c.Applies = newMockApplies(c)
	c.ConfigurationVersions = newMockConfigurationVersions(c)
	c.CostEstimates = newMockCostEstimates(c)
	c.Organizations = newMockOrganizations(c)
	c.Plans = newMockPlans(c)
	c.TaskStages = newMockTaskStages(c)
	c.PolicySetOutcomes = newMockPolicySetOutcomes(c)
	c.PolicyChecks = newMockPolicyChecks(c)
	c.Projects = newMockProjects(c)
	c.Runs = newMockRuns(c)
	c.RunEvents = newMockRunEvents(c)
	c.StateVersions = newMockStateVersions(c)
	c.StateVersionOutputs = newMockStateVersionOutputs(c)
	c.Variables = newMockVariables(c)
	c.Workspaces = newMockWorkspaces(c)
	c.RedactedPlans = newMockRedactedPlans(c)
	return c
}

type MockApplies struct {
	client  *MockClient
	applies map[string]*tfe.Apply
	logs    map[string]string
}

func newMockApplies(client *MockClient) *MockApplies {
	return &MockApplies{
		client:  client,
		applies: make(map[string]*tfe.Apply),
		logs:    make(map[string]string),
	}
}

// create is a helper function to create a mock apply that uses the configured
// working directory to find the logfile.
func (m *MockApplies) create(cvID, workspaceID string) (*tfe.Apply, error) {
	c, ok := m.client.ConfigurationVersions.configVersions[cvID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	if c.Speculative {
		// Speculative means its plan-only so we don't create a Apply.
		return nil, nil
	}

	id := GenerateID("apply-")
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

	if w.AutoApply {
		a.Status = tfe.ApplyRunning
	}

	m.logs[url] = filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"apply.log",
	)
	m.applies[a.ID] = a

	return a, nil
}

func (m *MockApplies) Read(ctx context.Context, applyID string) (*tfe.Apply, error) {
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

func (m *MockApplies) Logs(ctx context.Context, applyID string) (io.Reader, error) {
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

type MockConfigurationVersions struct {
	client         *MockClient
	configVersions map[string]*tfe.ConfigurationVersion
	uploadPaths    map[string]string
	uploadURLs     map[string]*tfe.ConfigurationVersion
}

func newMockConfigurationVersions(client *MockClient) *MockConfigurationVersions {
	return &MockConfigurationVersions{
		client:         client,
		configVersions: make(map[string]*tfe.ConfigurationVersion),
		uploadPaths:    make(map[string]string),
		uploadURLs:     make(map[string]*tfe.ConfigurationVersion),
	}
}

func (m *MockConfigurationVersions) List(ctx context.Context, workspaceID string, options *tfe.ConfigurationVersionListOptions) (*tfe.ConfigurationVersionList, error) {
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

func (m *MockConfigurationVersions) Create(ctx context.Context, workspaceID string, options tfe.ConfigurationVersionCreateOptions) (*tfe.ConfigurationVersion, error) {
	id := GenerateID("cv-")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	cv := &tfe.ConfigurationVersion{
		ID:        id,
		Status:    tfe.ConfigurationPending,
		UploadURL: url,
	}

	if options.Provisional != nil && *options.Provisional {
		cv.Provisional = true
	}

	if options.Speculative != nil && *options.Speculative {
		cv.Speculative = true
	}

	m.configVersions[cv.ID] = cv
	m.uploadURLs[url] = cv

	return cv, nil
}

func (m *MockConfigurationVersions) Read(ctx context.Context, cvID string) (*tfe.ConfigurationVersion, error) {
	cv, ok := m.configVersions[cvID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return cv, nil
}

func (m *MockConfigurationVersions) ReadWithOptions(ctx context.Context, cvID string, options *tfe.ConfigurationVersionReadOptions) (*tfe.ConfigurationVersion, error) {
	cv, ok := m.configVersions[cvID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return cv, nil
}

func (m *MockConfigurationVersions) Upload(ctx context.Context, url, path string) error {
	cv, ok := m.uploadURLs[url]
	if !ok {
		return errors.New("404 not found")
	}
	m.uploadPaths[cv.ID] = path
	cv.Status = tfe.ConfigurationUploaded

	return m.UploadTarGzip(ctx, url, nil)
}

func (m *MockConfigurationVersions) UploadTarGzip(ctx context.Context, url string, archive io.Reader) error {
	return nil
}

func (m *MockConfigurationVersions) Archive(ctx context.Context, cvID string) error {
	panic("not implemented")
}

func (m *MockConfigurationVersions) Download(ctx context.Context, cvID string) ([]byte, error) {
	panic("not implemented")
}

type MockCostEstimates struct {
	client      *MockClient
	Estimations map[string]*tfe.CostEstimate
	logs        map[string]string
}

func newMockCostEstimates(client *MockClient) *MockCostEstimates {
	return &MockCostEstimates{
		client:      client,
		Estimations: make(map[string]*tfe.CostEstimate),
		logs:        make(map[string]string),
	}
}

// create is a helper function to create a mock cost estimation that uses the
// configured working directory to find the logfile.
func (m *MockCostEstimates) create(cvID, workspaceID string) (*tfe.CostEstimate, error) {
	id := GenerateID("ce-")

	ce := &tfe.CostEstimate{
		ID:                    id,
		MatchedResourcesCount: 1,
		ResourcesCount:        1,
		DeltaMonthlyCost:      "0.00",
		ProposedMonthlyCost:   "0.00",
		Status:                tfe.CostEstimateFinished,
	}

	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	logfile := filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"cost-estimate.log",
	)

	if _, err := os.Stat(logfile); os.IsNotExist(err) {
		return nil, nil
	}

	m.logs[ce.ID] = logfile
	m.Estimations[ce.ID] = ce

	return ce, nil
}

func (m *MockCostEstimates) Read(ctx context.Context, costEstimateID string) (*tfe.CostEstimate, error) {
	ce, ok := m.Estimations[costEstimateID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return ce, nil
}

func (m *MockCostEstimates) Logs(ctx context.Context, costEstimateID string) (io.Reader, error) {
	ce, ok := m.Estimations[costEstimateID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	logfile, ok := m.logs[ce.ID]
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

	ce.Status = tfe.CostEstimateFinished

	return bytes.NewBuffer(logs), nil
}

type MockOrganizations struct {
	client        *MockClient
	organizations map[string]*tfe.Organization
}

func newMockOrganizations(client *MockClient) *MockOrganizations {
	return &MockOrganizations{
		client:        client,
		organizations: make(map[string]*tfe.Organization),
	}
}

func (m *MockOrganizations) List(ctx context.Context, options *tfe.OrganizationListOptions) (*tfe.OrganizationList, error) {
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
		time.Sleep(1 * time.Millisecond)
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

func (m *MockOrganizations) Create(ctx context.Context, options tfe.OrganizationCreateOptions) (*tfe.Organization, error) {
	org := &tfe.Organization{Name: *options.Name}
	m.organizations[org.Name] = org
	return org, nil
}

func (m *MockOrganizations) Read(ctx context.Context, name string) (*tfe.Organization, error) {
	return m.ReadWithOptions(ctx, name, tfe.OrganizationReadOptions{})
}

func (m *MockOrganizations) ReadWithOptions(ctx context.Context, name string, options tfe.OrganizationReadOptions) (*tfe.Organization, error) {
	org, ok := m.organizations[name]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return org, nil
}

func (m *MockOrganizations) Update(ctx context.Context, name string, options tfe.OrganizationUpdateOptions) (*tfe.Organization, error) {
	org, ok := m.organizations[name]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	org.Name = *options.Name
	return org, nil

}

func (m *MockOrganizations) Delete(ctx context.Context, name string) error {
	delete(m.organizations, name)
	return nil
}

func (m *MockOrganizations) ReadCapacity(ctx context.Context, name string) (*tfe.Capacity, error) {
	var pending, running int
	for _, r := range m.client.Runs.Runs {
		if r.Status == tfe.RunPending {
			pending++
			continue
		}
		running++
	}
	return &tfe.Capacity{Pending: pending, Running: running}, nil
}

func (m *MockOrganizations) ReadEntitlements(ctx context.Context, name string) (*tfe.Entitlements, error) {
	return &tfe.Entitlements{
		Operations:            true,
		PrivateModuleRegistry: true,
		Sentinel:              true,
		StateStorage:          true,
		Teams:                 true,
		VCSIntegrations:       true,
	}, nil
}

func (m *MockOrganizations) ReadRunQueue(ctx context.Context, name string, options tfe.ReadRunQueueOptions) (*tfe.RunQueue, error) {
	rq := &tfe.RunQueue{}

	for _, r := range m.client.Runs.Runs {
		rq.Items = append(rq.Items, r)
	}

	rq.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(rq.Items),
	}

	return rq, nil
}

type MockRedactedPlans struct {
	client        *MockClient
	redactedPlans map[string][]byte
}

func newMockRedactedPlans(client *MockClient) *MockRedactedPlans {
	return &MockRedactedPlans{
		client:        client,
		redactedPlans: make(map[string][]byte),
	}
}

func (m *MockRedactedPlans) create(cvID, workspaceID, planID string) error {
	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return tfe.ErrResourceNotFound
	}

	planPath := filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"plan-redacted.json",
	)

	redactedPlanFile, err := os.Open(planPath)
	if err != nil {
		return err
	}

	raw, err := io.ReadAll(redactedPlanFile)
	if err != nil {
		return err
	}

	m.redactedPlans[planID] = raw

	return nil
}

func (m *MockRedactedPlans) Read(ctx context.Context, hostname, token, planID string) ([]byte, error) {
	if p, ok := m.redactedPlans[planID]; ok {
		return p, nil
	}
	return nil, tfe.ErrResourceNotFound
}

type MockPlans struct {
	client      *MockClient
	logs        map[string]string
	planOutputs map[string][]byte
	plans       map[string]*tfe.Plan
}

func newMockPlans(client *MockClient) *MockPlans {
	return &MockPlans{
		client:      client,
		logs:        make(map[string]string),
		planOutputs: make(map[string][]byte),
		plans:       make(map[string]*tfe.Plan),
	}
}

// create is a helper function to create a mock plan that uses the configured
// working directory to find the logfile.
func (m *MockPlans) create(cvID, workspaceID string) (*tfe.Plan, error) {
	id := GenerateID("plan-")
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

	// Try to load unredacted json output, if it exists
	outputPath := filepath.Join(
		m.client.ConfigurationVersions.uploadPaths[cvID],
		w.WorkingDirectory,
		"plan-unredacted.json",
	)
	if outBytes, err := os.ReadFile(outputPath); err == nil {
		m.planOutputs[p.ID] = outBytes
	}

	m.plans[p.ID] = p

	return p, nil
}

func (m *MockPlans) Read(ctx context.Context, planID string) (*tfe.Plan, error) {
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

func (m *MockPlans) Logs(ctx context.Context, planID string) (io.Reader, error) {
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

func (m *MockPlans) ReadJSONOutput(ctx context.Context, planID string) ([]byte, error) {
	planOutput, ok := m.planOutputs[planID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	return planOutput, nil
}

type MockTaskStages struct {
	client *MockClient
}

func newMockTaskStages(client *MockClient) *MockTaskStages {
	return &MockTaskStages{
		client: client,
	}
}

func (m *MockTaskStages) Override(ctx context.Context, taskStageID string, options tfe.TaskStageOverrideOptions) (*tfe.TaskStage, error) {
	switch taskStageID {
	case "ts-err":
		return nil, errors.New("test error")

	default:
		return nil, nil
	}
}

func (m *MockTaskStages) Read(ctx context.Context, taskStageID string, options *tfe.TaskStageReadOptions) (*tfe.TaskStage, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockTaskStages) List(ctx context.Context, runID string, options *tfe.TaskStageListOptions) (*tfe.TaskStageList, error) {
	//TODO implement me
	panic("implement me")
}

type MockPolicySetOutcomes struct {
	client *MockClient
}

func newMockPolicySetOutcomes(client *MockClient) *MockPolicySetOutcomes {
	return &MockPolicySetOutcomes{
		client: client,
	}
}

func (m *MockPolicySetOutcomes) List(ctx context.Context, policyEvaluationID string, options *tfe.PolicySetOutcomeListOptions) (*tfe.PolicySetOutcomeList, error) {
	switch policyEvaluationID {
	case "pol-pass":
		return &tfe.PolicySetOutcomeList{
			Items: []*tfe.PolicySetOutcome{
				{
					ID: policyEvaluationID,
					Outcomes: []tfe.Outcome{
						{
							EnforcementLevel: "mandatory",
							Query:            "data.example.rule",
							Status:           "passed",
							PolicyName:       "policy-pass",
							Description:      "This policy will pass",
						},
					},
					Overridable:          tfe.Bool(true),
					Error:                "",
					PolicySetName:        "policy-set-that-passes",
					PolicySetDescription: "This policy set will always pass",
					ResultCount: tfe.PolicyResultCount{
						AdvisoryFailed:  0,
						MandatoryFailed: 0,
						Passed:          1,
						Errored:         0,
					},
				},
			},
		}, nil
	case "pol-fail":
		return &tfe.PolicySetOutcomeList{
			Items: []*tfe.PolicySetOutcome{
				{
					ID: policyEvaluationID,
					Outcomes: []tfe.Outcome{
						{
							EnforcementLevel: "mandatory",
							Query:            "data.example.rule",
							Status:           "failed",
							PolicyName:       "policy-fail",
							Description:      "This policy will fail",
						},
					},
					Overridable:          tfe.Bool(true),
					Error:                "",
					PolicySetName:        "policy-set-that-fails",
					PolicySetDescription: "This policy set will always fail",
					ResultCount: tfe.PolicyResultCount{
						AdvisoryFailed:  0,
						MandatoryFailed: 1,
						Passed:          0,
						Errored:         0,
					},
				},
			},
		}, nil

	case "adv-fail":
		return &tfe.PolicySetOutcomeList{
			Items: []*tfe.PolicySetOutcome{
				{
					ID: policyEvaluationID,
					Outcomes: []tfe.Outcome{
						{
							EnforcementLevel: "advisory",
							Query:            "data.example.rule",
							Status:           "failed",
							PolicyName:       "policy-fail",
							Description:      "This policy will fail",
						},
					},
					Overridable:          tfe.Bool(true),
					Error:                "",
					PolicySetName:        "policy-set-that-fails",
					PolicySetDescription: "This policy set will always fail",
					ResultCount: tfe.PolicyResultCount{
						AdvisoryFailed:  1,
						MandatoryFailed: 0,
						Passed:          0,
						Errored:         0,
					},
				},
			},
		}, nil
	default:
		return &tfe.PolicySetOutcomeList{
			Items: []*tfe.PolicySetOutcome{
				{
					ID: policyEvaluationID,
					Outcomes: []tfe.Outcome{
						{
							EnforcementLevel: "mandatory",
							Query:            "data.example.rule",
							Status:           "passed",
							PolicyName:       "policy-pass",
							Description:      "This policy will pass",
						},
					},
					Overridable:          tfe.Bool(true),
					Error:                "",
					PolicySetName:        "policy-set-that-passes",
					PolicySetDescription: "This policy set will always pass",
					ResultCount: tfe.PolicyResultCount{
						AdvisoryFailed:  0,
						MandatoryFailed: 0,
						Passed:          1,
						Errored:         0,
					},
				},
			},
		}, nil
	}
}

func (m *MockPolicySetOutcomes) Read(ctx context.Context, policySetOutcomeID string) (*tfe.PolicySetOutcome, error) {
	return nil, nil
}

type MockPolicyChecks struct {
	client *MockClient
	checks map[string]*tfe.PolicyCheck
	logs   map[string]string
}

func newMockPolicyChecks(client *MockClient) *MockPolicyChecks {
	return &MockPolicyChecks{
		client: client,
		checks: make(map[string]*tfe.PolicyCheck),
		logs:   make(map[string]string),
	}
}

// create is a helper function to create a mock policy check that uses the
// configured working directory to find the logfile.
func (m *MockPolicyChecks) create(cvID, workspaceID string) (*tfe.PolicyCheck, error) {
	id := GenerateID("pc-")

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

func (m *MockPolicyChecks) List(ctx context.Context, runID string, options *tfe.PolicyCheckListOptions) (*tfe.PolicyCheckList, error) {
	_, ok := m.client.Runs.Runs[runID]
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

func (m *MockPolicyChecks) Read(ctx context.Context, policyCheckID string) (*tfe.PolicyCheck, error) {
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

func (m *MockPolicyChecks) Override(ctx context.Context, policyCheckID string) (*tfe.PolicyCheck, error) {
	pc, ok := m.checks[policyCheckID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	pc.Status = tfe.PolicyOverridden
	return pc, nil
}

func (m *MockPolicyChecks) Logs(ctx context.Context, policyCheckID string) (io.Reader, error) {
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

type MockProjects struct {
	client   *MockClient
	projects map[string]*tfe.Project
}

func newMockProjects(client *MockClient) *MockProjects {
	return &MockProjects{
		client:   client,
		projects: make(map[string]*tfe.Project),
	}
}

func (m *MockProjects) Create(ctx context.Context, organization string, options tfe.ProjectCreateOptions) (*tfe.Project, error) {
	id := GenerateID("prj-")

	p := &tfe.Project{
		ID:   id,
		Name: options.Name,
	}

	m.projects[p.ID] = p

	return p, nil
}

func (m *MockProjects) List(ctx context.Context, organization string, options *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
	pl := &tfe.ProjectList{}

	for _, project := range m.projects {
		pc, err := copystructure.Copy(project)
		if err != nil {
			panic(err)
		}
		pl.Items = append(pl.Items, pc.(*tfe.Project))
	}

	pl.Pagination = &tfe.Pagination{
		CurrentPage:  1,
		NextPage:     1,
		PreviousPage: 1,
		TotalPages:   1,
		TotalCount:   len(pl.Items),
	}

	return pl, nil
}

func (m *MockProjects) Read(ctx context.Context, projectID string) (*tfe.Project, error) {
	p, ok := m.projects[projectID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	// we must return a copy for the client
	pc, err := copystructure.Copy(p)
	if err != nil {
		panic(err)
	}

	return pc.(*tfe.Project), nil
}

func (m *MockProjects) Update(ctx context.Context, projectID string, options tfe.ProjectUpdateOptions) (*tfe.Project, error) {
	p, ok := m.projects[projectID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	p.Name = *options.Name

	// we must return a copy for the client
	pc, err := copystructure.Copy(p)
	if err != nil {
		panic(err)
	}

	return pc.(*tfe.Project), nil
}

func (m *MockProjects) Delete(ctx context.Context, projectID string) error {
	var p *tfe.Project = nil
	for _, p := range m.projects {
		if p.ID == projectID {

			break
		}
	}
	if p == nil {
		return tfe.ErrResourceNotFound
	}

	delete(m.projects, p.Name)

	return nil
}

type MockRuns struct {
	sync.Mutex

	client     *MockClient
	Runs       map[string]*tfe.Run
	workspaces map[string][]*tfe.Run

	// If ModifyNewRun is non-nil, the create method will call it just before
	// saving a new run in the runs map, so that a calling test can mimic
	// side-effects that a real server might apply in certain situations.
	ModifyNewRun func(client *MockClient, options tfe.RunCreateOptions, run *tfe.Run)
}

func newMockRuns(client *MockClient) *MockRuns {
	return &MockRuns{
		client:     client,
		Runs:       make(map[string]*tfe.Run),
		workspaces: make(map[string][]*tfe.Run),
	}
}

func (m *MockRuns) List(ctx context.Context, workspaceID string, options *tfe.RunListOptions) (*tfe.RunList, error) {
	m.Lock()
	defer m.Unlock()

	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	rl := &tfe.RunList{}
	for _, run := range m.workspaces[w.ID] {
		rc, err := copystructure.Copy(run)
		if err != nil {
			panic(err)
		}
		rl.Items = append(rl.Items, rc.(*tfe.Run))
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

func (m *MockRuns) Create(ctx context.Context, options tfe.RunCreateOptions) (*tfe.Run, error) {
	m.Lock()
	defer m.Unlock()

	a, err := m.client.Applies.create(options.ConfigurationVersion.ID, options.Workspace.ID)
	if err != nil {
		return nil, err
	}

	ce, err := m.client.CostEstimates.create(options.ConfigurationVersion.ID, options.Workspace.ID)
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
		ID:                    GenerateID("run-"),
		Actions:               &tfe.RunActions{IsCancelable: true},
		Apply:                 a,
		CostEstimate:          ce,
		HasChanges:            false,
		Permissions:           &tfe.RunPermissions{},
		Plan:                  p,
		ReplaceAddrs:          options.ReplaceAddrs,
		Status:                tfe.RunPending,
		TargetAddrs:           options.TargetAddrs,
		AllowConfigGeneration: options.AllowConfigGeneration,
	}

	if options.Message != nil {
		r.Message = *options.Message
	}

	if pc != nil {
		r.PolicyChecks = []*tfe.PolicyCheck{pc}
	}

	if options.IsDestroy != nil {
		r.IsDestroy = *options.IsDestroy
	}

	if options.Refresh != nil {
		r.Refresh = *options.Refresh
	}

	if options.RefreshOnly != nil {
		r.RefreshOnly = *options.RefreshOnly
	}

	if options.AllowConfigGeneration != nil && *options.AllowConfigGeneration {
		r.Plan.GeneratedConfiguration = true
	}

	w, ok := m.client.Workspaces.workspaceIDs[options.Workspace.ID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	if w.CurrentRun == nil {
		w.CurrentRun = r
	}

	r.Workspace = &tfe.Workspace{
		ID:                         w.ID,
		StructuredRunOutputEnabled: w.StructuredRunOutputEnabled,
		TerraformVersion:           w.TerraformVersion,
	}

	if w.StructuredRunOutputEnabled {
		err := m.client.RedactedPlans.create(options.ConfigurationVersion.ID, options.Workspace.ID, p.ID)
		if err != nil {
			return nil, err
		}
	}

	if m.ModifyNewRun != nil {
		// caller-provided callback may modify the run in-place to mimic
		// side-effects that a real server might take in some situations.
		m.ModifyNewRun(m.client, options, r)
	}

	m.Runs[r.ID] = r
	m.workspaces[options.Workspace.ID] = append(m.workspaces[options.Workspace.ID], r)

	return r, nil
}

func (m *MockRuns) Read(ctx context.Context, runID string) (*tfe.Run, error) {
	return m.ReadWithOptions(ctx, runID, nil)
}

func (m *MockRuns) ReadWithOptions(ctx context.Context, runID string, options *tfe.RunReadOptions) (*tfe.Run, error) {
	m.Lock()
	defer m.Unlock()

	r, ok := m.Runs[runID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	pending := false
	for _, r := range m.Runs {
		if r.ID != runID && r.Status == tfe.RunPending {
			pending = true
			break
		}
	}

	if !pending && r.Status == tfe.RunPending {
		// Only update the status if there are no other pending runs.
		r.Status = tfe.RunPlanning
		r.Plan.Status = tfe.PlanRunning
	}

	logs, _ := ioutil.ReadFile(m.client.Plans.logs[r.Plan.LogReadURL])
	if (r.Status == tfe.RunPlanning || r.Status == tfe.RunPlannedAndSaved) && r.Plan.Status == tfe.PlanFinished {
		hasChanges := r.IsDestroy ||
			bytes.Contains(logs, []byte("1 to add")) ||
			bytes.Contains(logs, []byte("1 to change")) ||
			bytes.Contains(logs, []byte("1 to import"))
		if hasChanges {
			r.Actions.IsCancelable = false
			r.Actions.IsConfirmable = true
			r.HasChanges = true
			r.Plan.HasChanges = true
			r.Permissions.CanApply = true
		}

		hasError := bytes.Contains(logs, []byte("null_resource.foo: 1 error")) ||
			bytes.Contains(logs, []byte("Error: Unsupported block type")) ||
			bytes.Contains(logs, []byte("Error: Conflicting configuration arguments"))
		if hasError {
			r.Actions.IsCancelable = false
			r.HasChanges = false
			r.Status = tfe.RunErrored
		}
	}

	// we must return a copy for the client
	rc, err := copystructure.Copy(r)
	if err != nil {
		panic(err)
	}
	r = rc.(*tfe.Run)

	// After copying, handle includes... or at least, any includes we're known to rely on.
	if options != nil {
		for _, n := range options.Include {
			switch n {
			case tfe.RunWorkspace:
				ws, ok := m.client.Workspaces.workspaceIDs[r.Workspace.ID]
				if ok {
					r.Workspace = ws
				}
			}
		}
	}

	return r, nil
}

func (m *MockRuns) Apply(ctx context.Context, runID string, options tfe.RunApplyOptions) error {
	m.Lock()
	defer m.Unlock()

	r, ok := m.Runs[runID]
	if !ok {
		return tfe.ErrResourceNotFound
	}
	if r.Status != tfe.RunPending {
		// Only update the status if the run is not pending anymore.
		r.Status = tfe.RunApplying
		r.Actions.IsConfirmable = false
		r.Apply.Status = tfe.ApplyRunning
	}
	return nil
}

func (m *MockRuns) Cancel(ctx context.Context, runID string, options tfe.RunCancelOptions) error {
	panic("not implemented")
}

func (m *MockRuns) ForceCancel(ctx context.Context, runID string, options tfe.RunForceCancelOptions) error {
	panic("not implemented")
}

func (m *MockRuns) ForceExecute(ctx context.Context, runID string) error {
	panic("implement me")
}

func (m *MockRuns) Discard(ctx context.Context, runID string, options tfe.RunDiscardOptions) error {
	m.Lock()
	defer m.Unlock()

	r, ok := m.Runs[runID]
	if !ok {
		return tfe.ErrResourceNotFound
	}
	r.Status = tfe.RunDiscarded
	r.Actions.IsConfirmable = false
	return nil
}

type MockRunEvents struct{}

func newMockRunEvents(_ *MockClient) *MockRunEvents {
	return &MockRunEvents{}
}

// List all the runs events of the given run.
func (m *MockRunEvents) List(ctx context.Context, runID string, options *tfe.RunEventListOptions) (*tfe.RunEventList, error) {
	return &tfe.RunEventList{
		Items: []*tfe.RunEvent{},
	}, nil
}

func (m *MockRunEvents) Read(ctx context.Context, runEventID string) (*tfe.RunEvent, error) {
	return m.ReadWithOptions(ctx, runEventID, nil)
}

func (m *MockRunEvents) ReadWithOptions(ctx context.Context, runEventID string, options *tfe.RunEventReadOptions) (*tfe.RunEvent, error) {
	return &tfe.RunEvent{
		ID:        GenerateID("re-"),
		Action:    "created",
		CreatedAt: time.Now(),
	}, nil
}

type MockStateVersions struct {
	client        *MockClient
	states        map[string][]byte
	stateVersions map[string]*tfe.StateVersion
	workspaces    map[string][]string
	outputStates  map[string][]byte
}

func newMockStateVersions(client *MockClient) *MockStateVersions {
	return &MockStateVersions{
		client:        client,
		states:        make(map[string][]byte),
		stateVersions: make(map[string]*tfe.StateVersion),
		workspaces:    make(map[string][]string),
		outputStates:  make(map[string][]byte),
	}
}

func (m *MockStateVersions) List(ctx context.Context, options *tfe.StateVersionListOptions) (*tfe.StateVersionList, error) {
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

func (m *MockStateVersions) Create(ctx context.Context, workspaceID string, options tfe.StateVersionCreateOptions) (*tfe.StateVersion, error) {
	id := GenerateID("sv-")
	runID := os.Getenv("TFE_RUN_ID")
	url := fmt.Sprintf("https://app.terraform.io/_archivist/%s", id)

	if runID != "" && (options.Run == nil || runID != options.Run.ID) {
		return nil, fmt.Errorf("option.Run.ID does not contain the ID exported by TFE_RUN_ID")
	}

	sv := &tfe.StateVersion{
		ID:          id,
		DownloadURL: url,
		UploadURL:   fmt.Sprintf("/_archivist/upload/%s", id),
		Serial:      *options.Serial,
	}

	state, err := base64.StdEncoding.DecodeString(*options.State)
	if err != nil {
		return nil, err
	}
	m.states[sv.DownloadURL] = state
	m.outputStates[sv.ID] = []byte(*options.JSONStateOutputs)
	m.stateVersions[sv.ID] = sv
	m.workspaces[workspaceID] = append(m.workspaces[workspaceID], sv.ID)

	return sv, nil
}

func (m *MockStateVersions) Upload(ctx context.Context, workspaceID string, options tfe.StateVersionUploadOptions) (*tfe.StateVersion, error) {
	createOptions := options.StateVersionCreateOptions
	createOptions.State = tfe.String(base64.StdEncoding.EncodeToString(options.RawState))

	return m.Create(ctx, workspaceID, createOptions)
}

func (m *MockStateVersions) Read(ctx context.Context, svID string) (*tfe.StateVersion, error) {
	return m.ReadWithOptions(ctx, svID, nil)
}

func (m *MockStateVersions) ReadWithOptions(ctx context.Context, svID string, options *tfe.StateVersionReadOptions) (*tfe.StateVersion, error) {
	sv, ok := m.stateVersions[svID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return sv, nil
}

func (m *MockStateVersions) ReadCurrent(ctx context.Context, workspaceID string) (*tfe.StateVersion, error) {
	return m.ReadCurrentWithOptions(ctx, workspaceID, nil)
}

func (m *MockStateVersions) ReadCurrentWithOptions(ctx context.Context, workspaceID string, options *tfe.StateVersionCurrentOptions) (*tfe.StateVersion, error) {
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

func (m *MockStateVersions) Download(ctx context.Context, url string) ([]byte, error) {
	state, ok := m.states[url]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return state, nil
}

func (m *MockStateVersions) ListOutputs(ctx context.Context, svID string, options *tfe.StateVersionOutputsListOptions) (*tfe.StateVersionOutputsList, error) {
	panic("not implemented")
}

type MockStateVersionOutputs struct {
	client  *MockClient
	outputs map[string]*tfe.StateVersionOutput
}

func newMockStateVersionOutputs(client *MockClient) *MockStateVersionOutputs {
	return &MockStateVersionOutputs{
		client:  client,
		outputs: make(map[string]*tfe.StateVersionOutput),
	}
}

// This is a helper function in order to create mocks to be read later
func (m *MockStateVersionOutputs) create(id string, svo *tfe.StateVersionOutput) {
	m.outputs[id] = svo
}

func (m *MockStateVersionOutputs) Read(ctx context.Context, outputID string) (*tfe.StateVersionOutput, error) {
	result, ok := m.outputs[outputID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	return result, nil
}

func (m *MockStateVersionOutputs) ReadCurrent(ctx context.Context, workspaceID string) (*tfe.StateVersionOutputsList, error) {
	svl := &tfe.StateVersionOutputsList{}
	for _, sv := range m.outputs {
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

type MockVariables struct {
	client     *MockClient
	workspaces map[string]*tfe.VariableList
}

var _ tfe.Variables = (*MockVariables)(nil)

func newMockVariables(client *MockClient) *MockVariables {
	return &MockVariables{
		client:     client,
		workspaces: make(map[string]*tfe.VariableList),
	}
}

func (m *MockVariables) List(ctx context.Context, workspaceID string, options *tfe.VariableListOptions) (*tfe.VariableList, error) {
	vl := m.workspaces[workspaceID]
	return vl, nil
}

func (m *MockVariables) Create(ctx context.Context, workspaceID string, options tfe.VariableCreateOptions) (*tfe.Variable, error) {
	v := &tfe.Variable{
		ID:       GenerateID("var-"),
		Key:      *options.Key,
		Category: *options.Category,
	}
	if options.Value != nil {
		v.Value = *options.Value
	}
	if options.HCL != nil {
		v.HCL = *options.HCL
	}
	if options.Sensitive != nil {
		v.Sensitive = *options.Sensitive
	}

	workspace := workspaceID

	if m.workspaces[workspace] == nil {
		m.workspaces[workspace] = &tfe.VariableList{}
	}

	vl := m.workspaces[workspace]
	vl.Items = append(vl.Items, v)

	return v, nil
}

func (m *MockVariables) Read(ctx context.Context, workspaceID string, variableID string) (*tfe.Variable, error) {
	panic("not implemented")
}

func (m *MockVariables) Update(ctx context.Context, workspaceID string, variableID string, options tfe.VariableUpdateOptions) (*tfe.Variable, error) {
	panic("not implemented")
}

func (m *MockVariables) Delete(ctx context.Context, workspaceID string, variableID string) error {
	panic("not implemented")
}

type MockWorkspaces struct {
	client         *MockClient
	workspaceIDs   map[string]*tfe.Workspace
	workspaceNames map[string]*tfe.Workspace
}

func newMockWorkspaces(client *MockClient) *MockWorkspaces {
	return &MockWorkspaces{
		client:         client,
		workspaceIDs:   make(map[string]*tfe.Workspace),
		workspaceNames: make(map[string]*tfe.Workspace),
	}
}

func (m *MockWorkspaces) List(ctx context.Context, organization string, options *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	wl := &tfe.WorkspaceList{}
	// Get all the workspaces that match the Search value
	searchValue := ""
	var ws []*tfe.Workspace
	var tags []string

	if options != nil {
		if len(options.Search) > 0 {
			searchValue = options.Search
		}
		if len(options.Tags) > 0 {
			tags = strings.Split(options.Tags, ",")
		}
	}

	for _, w := range m.workspaceIDs {
		wTags := make(map[string]struct{})
		for _, wTag := range w.Tags {
			wTags[wTag.Name] = struct{}{}
		}

		if strings.Contains(w.Name, searchValue) {
			tagsSatisfied := true
			for _, tag := range tags {
				if _, ok := wTags[tag]; !ok {
					tagsSatisfied = false
				}
			}
			if tagsSatisfied {
				ws = append(ws, w)
			}
		}
	}

	// Return an empty result if we have no matches.
	if len(ws) == 0 {
		wl.Pagination = &tfe.Pagination{
			CurrentPage: 1,
		}
		return wl, nil
	}

	numPages := (len(ws) / 20) + 1
	currentPage := 1
	if options != nil {
		if options.PageNumber != 0 {
			currentPage = options.PageNumber
		}
	}
	previousPage := currentPage - 1
	nextPage := currentPage + 1

	for i := ((currentPage - 1) * 20); i < ((currentPage-1)*20)+20; i++ {
		if i > (len(ws) - 1) {
			break
		}
		wl.Items = append(wl.Items, ws[i])
	}

	wl.Pagination = &tfe.Pagination{
		CurrentPage:  currentPage,
		NextPage:     nextPage,
		PreviousPage: previousPage,
		TotalPages:   numPages,
		TotalCount:   len(wl.Items),
	}

	return wl, nil
}

func (m *MockWorkspaces) Create(ctx context.Context, organization string, options tfe.WorkspaceCreateOptions) (*tfe.Workspace, error) {
	// for TestCloud_setUnavailableTerraformVersion
	if *options.Name == "unavailable-terraform-version" && options.TerraformVersion != nil {
		return nil, fmt.Errorf("requested Terraform version not available in this TFC instance")
	}
	if strings.HasSuffix(*options.Name, "no-operations") {
		options.Operations = tfe.Bool(false)
		options.ExecutionMode = tfe.String("local")
	} else if options.Operations == nil {
		options.Operations = tfe.Bool(true)
		options.ExecutionMode = tfe.String("remote")
	}
	w := &tfe.Workspace{
		ID:                         GenerateID("ws-"),
		Name:                       *options.Name,
		ExecutionMode:              *options.ExecutionMode,
		Operations:                 *options.Operations,
		StructuredRunOutputEnabled: false,
		Permissions: &tfe.WorkspacePermissions{
			CanQueueApply:  true,
			CanQueueRun:    true,
			CanForceDelete: tfe.Bool(true),
		},
		Organization: &tfe.Organization{
			Name: organization,
		},
	}
	if options.Project != nil {
		w.Project = options.Project
	}
	if options.AutoApply != nil {
		w.AutoApply = *options.AutoApply
	}
	if options.VCSRepo != nil {
		w.VCSRepo = &tfe.VCSRepo{}
	}

	if options.TerraformVersion != nil {
		w.TerraformVersion = *options.TerraformVersion
	} else {
		w.TerraformVersion = tfversion.String()
	}

	var tags []*tfe.Tag
	for _, tag := range options.Tags {
		tags = append(tags, tag)
		w.TagNames = append(w.TagNames, tag.Name)
	}
	w.Tags = tags
	m.workspaceIDs[w.ID] = w
	m.workspaceNames[w.Name] = w
	return w, nil
}

func (m *MockWorkspaces) Read(ctx context.Context, organization, workspace string) (*tfe.Workspace, error) {
	// custom error for TestCloud_plan500 in backend_plan_test.go
	if workspace == "network-error" {
		return nil, errors.New("I'm a little teacup")
	}

	w, ok := m.workspaceNames[workspace]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return w, nil
}

func (m *MockWorkspaces) ReadByID(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return w, nil
}

func (m *MockWorkspaces) ReadWithOptions(ctx context.Context, organization string, workspace string, options *tfe.WorkspaceReadOptions) (*tfe.Workspace, error) {
	panic("not implemented")
}

func (m *MockWorkspaces) ReadByIDWithOptions(ctx context.Context, workspaceID string, options *tfe.WorkspaceReadOptions) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	return w, nil
}

func (m *MockWorkspaces) Update(ctx context.Context, organization, workspace string, options tfe.WorkspaceUpdateOptions) (*tfe.Workspace, error) {
	w, ok := m.workspaceNames[workspace]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	err := updateMockWorkspaceAttributes(w, options)
	if err != nil {
		return nil, err
	}

	delete(m.workspaceNames, workspace)
	m.workspaceNames[w.Name] = w

	return w, nil
}

func (m *MockWorkspaces) UpdateByID(ctx context.Context, workspaceID string, options tfe.WorkspaceUpdateOptions) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}

	originalName := w.Name
	err := updateMockWorkspaceAttributes(w, options)
	if err != nil {
		return nil, err
	}

	delete(m.workspaceNames, originalName)
	m.workspaceNames[w.Name] = w

	return w, nil
}

func updateMockWorkspaceAttributes(w *tfe.Workspace, options tfe.WorkspaceUpdateOptions) error {
	// for TestCloud_setUnavailableTerraformVersion
	if w.Name == "unavailable-terraform-version" && options.TerraformVersion != nil {
		return fmt.Errorf("requested Terraform version not available in this TFC instance")
	}

	if options.Operations != nil {
		w.Operations = *options.Operations
	}
	if options.ExecutionMode != nil {
		w.ExecutionMode = *options.ExecutionMode
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

	if options.StructuredRunOutputEnabled != nil {
		w.StructuredRunOutputEnabled = *options.StructuredRunOutputEnabled
	}

	return nil
}

func (m *MockWorkspaces) Delete(ctx context.Context, organization, workspace string) error {
	if w, ok := m.workspaceNames[workspace]; ok {
		delete(m.workspaceIDs, w.ID)
	}
	delete(m.workspaceNames, workspace)
	return nil
}

func (m *MockWorkspaces) DeleteByID(ctx context.Context, workspaceID string) error {
	if w, ok := m.workspaceIDs[workspaceID]; ok {
		delete(m.workspaceIDs, w.Name)
	}
	delete(m.workspaceIDs, workspaceID)
	return nil
}

func (m *MockWorkspaces) SafeDelete(ctx context.Context, organization, workspace string) error {
	w, ok := m.client.Workspaces.workspaceNames[workspace]

	if !ok {
		return tfe.ErrResourceNotFound
	}

	if w.Locked {
		return errors.New("cannot safe delete locked workspace")
	}

	if w.ResourceCount > 0 {
		return fmt.Errorf("cannot safe delete workspace with %d resources", w.ResourceCount)
	}

	return m.Delete(ctx, organization, workspace)
}

func (m *MockWorkspaces) SafeDeleteByID(ctx context.Context, workspaceID string) error {
	w, ok := m.client.Workspaces.workspaceIDs[workspaceID]
	if !ok {
		return tfe.ErrResourceNotFound
	}

	if w.Locked {
		return errors.New("cannot safe delete locked workspace")
	}

	if w.ResourceCount > 0 {
		return fmt.Errorf("cannot safe delete workspace with %d resources", w.ResourceCount)
	}

	return m.DeleteByID(ctx, workspaceID)
}

func (m *MockWorkspaces) RemoveVCSConnection(ctx context.Context, organization, workspace string) (*tfe.Workspace, error) {
	w, ok := m.workspaceNames[workspace]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	w.VCSRepo = nil
	return w, nil
}

func (m *MockWorkspaces) RemoveVCSConnectionByID(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	w.VCSRepo = nil
	return w, nil
}

func (m *MockWorkspaces) Lock(ctx context.Context, workspaceID string, options tfe.WorkspaceLockOptions) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	if w.Locked {
		return nil, tfe.ErrWorkspaceLocked
	}
	w.Locked = true
	return w, nil
}

func (m *MockWorkspaces) Unlock(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	if !w.Locked {
		return nil, tfe.ErrWorkspaceNotLocked
	}
	w.Locked = false
	return w, nil
}

func (m *MockWorkspaces) ForceUnlock(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	w, ok := m.workspaceIDs[workspaceID]
	if !ok {
		return nil, tfe.ErrResourceNotFound
	}
	if !w.Locked {
		return nil, tfe.ErrWorkspaceNotLocked
	}
	w.Locked = false
	return w, nil
}

func (m *MockWorkspaces) AssignSSHKey(ctx context.Context, workspaceID string, options tfe.WorkspaceAssignSSHKeyOptions) (*tfe.Workspace, error) {
	panic("not implemented")
}

func (m *MockWorkspaces) UnassignSSHKey(ctx context.Context, workspaceID string) (*tfe.Workspace, error) {
	panic("not implemented")
}

func (m *MockWorkspaces) ListRemoteStateConsumers(ctx context.Context, workspaceID string, options *tfe.RemoteStateConsumersListOptions) (*tfe.WorkspaceList, error) {
	panic("not implemented")
}

func (m *MockWorkspaces) AddRemoteStateConsumers(ctx context.Context, workspaceID string, options tfe.WorkspaceAddRemoteStateConsumersOptions) error {
	panic("not implemented")
}

func (m *MockWorkspaces) RemoveRemoteStateConsumers(ctx context.Context, workspaceID string, options tfe.WorkspaceRemoveRemoteStateConsumersOptions) error {
	panic("not implemented")
}

func (m *MockWorkspaces) UpdateRemoteStateConsumers(ctx context.Context, workspaceID string, options tfe.WorkspaceUpdateRemoteStateConsumersOptions) error {
	panic("not implemented")
}

func (m *MockWorkspaces) Readme(ctx context.Context, workspaceID string) (io.Reader, error) {
	panic("not implemented")
}

func (m *MockWorkspaces) ListTags(ctx context.Context, workspaceID string, options *tfe.WorkspaceTagListOptions) (*tfe.TagList, error) {
	panic("not implemented")
}

func (m *MockWorkspaces) AddTags(ctx context.Context, workspaceID string, options tfe.WorkspaceAddTagsOptions) error {
	return nil
}

func (m *MockWorkspaces) RemoveTags(ctx context.Context, workspaceID string, options tfe.WorkspaceRemoveTagsOptions) error {
	panic("not implemented")
}

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateID(s string) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = alphanumeric[rand.Intn(len(alphanumeric))]
	}
	return s + string(b)
}
