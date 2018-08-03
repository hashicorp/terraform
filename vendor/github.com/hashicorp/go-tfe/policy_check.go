package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ PolicyChecks = (*policyChecks)(nil)

// PolicyChecks describes all the policy check related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/policy-checks.html
type PolicyChecks interface {
	// List all policy checks of the given run.
	List(ctx context.Context, runID string, options PolicyCheckListOptions) ([]*PolicyCheck, error)

	// Override a soft-mandatory or warning policy.
	Override(ctx context.Context, policyCheckID string) (*PolicyCheck, error)
}

// policyChecks implements PolicyChecks.
type policyChecks struct {
	client *Client
}

// PolicyScope represents a policy scope.
type PolicyScope string

// List all available policy scopes.
const (
	PolicyScopeOrganization PolicyScope = "organization"
	PolicyScopeWorkspace    PolicyScope = "workspace"
)

// PolicyStatus represents a policy check state.
type PolicyStatus string

//List all available policy check statuses.
const (
	PolicyErrored    PolicyStatus = "errored"
	PolicyHardFailed PolicyStatus = "hard_failed"
	PolicyOverridden PolicyStatus = "overridden"
	PolicyPasses     PolicyStatus = "passed"
	PolicyPending    PolicyStatus = "pending"
	PolicyQueued     PolicyStatus = "queued"
	PolicySoftFailed PolicyStatus = "soft_failed"
)

// PolicyCheck represents a Terraform Enterprise policy check..
type PolicyCheck struct {
	ID               string                  `jsonapi:"primary,policy-checks"`
	Actions          *PolicyActions          `jsonapi:"attr,actions"`
	Permissions      *PolicyPermissions      `jsonapi:"attr,permissions"`
	Result           *PolicyResult           `jsonapi:"attr,result"`
	Scope            PolicyScope             `jsonapi:"attr,source"`
	Status           PolicyStatus            `jsonapi:"attr,status"`
	StatusTimestamps *PolicyStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// PolicyActions represents the policy check actions.
type PolicyActions struct {
	IsOverridable bool `json:"is-overridable"`
}

// PolicyPermissions represents the policy check permissions.
type PolicyPermissions struct {
	CanOverride bool `json:"can-override"`
}

// PolicyResult represents the complete policy check result,
type PolicyResult struct {
	AdvisoryFailed int  `json:"advisory-failed"`
	Duration       int  `json:"duration"`
	HardFailed     int  `json:"hard-failed"`
	Passed         int  `json:"passed"`
	Result         bool `json:"result"`
	// Sentinel       *sentinel.EvalResult `json:"sentinel"`
	SoftFailed  int `json:"soft-failed"`
	TotalFailed int `json:"total-failed"`
}

// PolicyStatusTimestamps holds the timestamps for individual policy check
// statuses.
type PolicyStatusTimestamps struct {
	ErroredAt    time.Time `json:"errored-at"`
	HardFailedAt time.Time `json:"hard-failed-at"`
	PassedAt     time.Time `json:"passed-at"`
	QueuedAt     time.Time `json:"queued-at"`
	SoftFailedAt time.Time `json:"soft-failed-at"`
}

// PolicyCheckListOptions represents the options for listing policy checks.
type PolicyCheckListOptions struct {
	ListOptions
}

// List all policy checks of the given run.
func (s *policyChecks) List(ctx context.Context, runID string, options PolicyCheckListOptions) ([]*PolicyCheck, error) {
	if !validStringID(&runID) {
		return nil, errors.New("Invalid value for run ID")
	}

	u := fmt.Sprintf("runs/%s/policy-checks", url.QueryEscape(runID))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	var pcs []*PolicyCheck
	err = s.client.do(ctx, req, &pcs)
	if err != nil {
		return nil, err
	}

	return pcs, nil
}

// Override a soft-mandatory or warning policy.
func (s *policyChecks) Override(ctx context.Context, policyCheckID string) (*PolicyCheck, error) {
	if !validStringID(&policyCheckID) {
		return nil, errors.New("Invalid value for policy check ID")
	}

	u := fmt.Sprintf("policy-checks/%s/actions/override", url.QueryEscape(policyCheckID))
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	pc := &PolicyCheck{}
	err = s.client.do(ctx, req, pc)
	if err != nil {
		return nil, err
	}

	return pc, nil
}
