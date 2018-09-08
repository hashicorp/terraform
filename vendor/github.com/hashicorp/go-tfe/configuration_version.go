package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	slug "github.com/hashicorp/go-slug"
)

// Compile-time proof of interface implementation.
var _ ConfigurationVersions = (*configurationVersions)(nil)

// ConfigurationVersions describes all the configuration version related
// methods that the Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/configuration-versions.html
type ConfigurationVersions interface {
	// List returns all configuration versions of a workspace.
	List(ctx context.Context, workspaceID string, options ConfigurationVersionListOptions) (*ConfigurationVersionList, error)

	// Create is used to create a new configuration version. The created
	// configuration version will be usable once data is uploaded to it.
	Create(ctx context.Context, workspaceID string, options ConfigurationVersionCreateOptions) (*ConfigurationVersion, error)

	// Read a configuration version by its ID.
	Read(ctx context.Context, cvID string) (*ConfigurationVersion, error)

	// Upload packages and uploads Terraform configuration files. It requires
	// the upload URL from a configuration version and the full path to the
	// configuration files on disk.
	Upload(ctx context.Context, url string, path string) error
}

// configurationVersions implements ConfigurationVersions.
type configurationVersions struct {
	client *Client
}

// ConfigurationStatus represents a configuration version status.
type ConfigurationStatus string

//List all available configuration version statuses.
const (
	ConfigurationErrored  ConfigurationStatus = "errored"
	ConfigurationPending  ConfigurationStatus = "pending"
	ConfigurationUploaded ConfigurationStatus = "uploaded"
)

// ConfigurationSource represents a source of a configuration version.
type ConfigurationSource string

// List all available configuration version sources.
const (
	ConfigurationSourceAPI       ConfigurationSource = "tfe-api"
	ConfigurationSourceBitbucket ConfigurationSource = "bitbucket"
	ConfigurationSourceGithub    ConfigurationSource = "github"
	ConfigurationSourceGitlab    ConfigurationSource = "gitlab"
	ConfigurationSourceTerraform ConfigurationSource = "terraform"
)

// ConfigurationVersionList represents a list of configuration versions.
type ConfigurationVersionList struct {
	*Pagination
	Items []*ConfigurationVersion
}

// ConfigurationVersion is a representation of an uploaded or ingressed
// Terraform configuration in TFE. A workspace must have at least one
// configuration version before any runs may be queued on it.
type ConfigurationVersion struct {
	ID               string              `jsonapi:"primary,configuration-versions"`
	AutoQueueRuns    bool                `jsonapi:"attr,auto-queue-runs"`
	Error            string              `jsonapi:"attr,error"`
	ErrorMessage     string              `jsonapi:"attr,error-message"`
	Source           ConfigurationSource `jsonapi:"attr,source"`
	Speculative      bool                `jsonapi:"attr,speculative "`
	Status           ConfigurationStatus `jsonapi:"attr,status"`
	StatusTimestamps *CVStatusTimestamps `jsonapi:"attr,status-timestamps"`
	UploadURL        string              `jsonapi:"attr,upload-url"`
}

// CVStatusTimestamps holds the timestamps for individual configuration version
// statuses.
type CVStatusTimestamps struct {
	FinishedAt time.Time `json:"finished-at"`
	QueuedAt   time.Time `json:"queued-at"`
	StartedAt  time.Time `json:"started-at"`
}

// ConfigurationVersionListOptions represents the options for listing
// configuration versions.
type ConfigurationVersionListOptions struct {
	ListOptions
}

// List returns all configuration versions of a workspace.
func (s *configurationVersions) List(ctx context.Context, workspaceID string, options ConfigurationVersionListOptions) (*ConfigurationVersionList, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/configuration-versions", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	cvl := &ConfigurationVersionList{}
	err = s.client.do(ctx, req, cvl)
	if err != nil {
		return nil, err
	}

	return cvl, nil
}

// ConfigurationVersionCreateOptions represents the options for creating a
// configuration version.
type ConfigurationVersionCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,configuration-versions"`

	// When true, runs are queued automatically when the configuration version
	// is uploaded.
	AutoQueueRuns *bool `jsonapi:"attr,auto-queue-runs,omitempty"`

	// When true, this configuration version can only be used for planning.
	Speculative *bool `jsonapi:"attr,speculative,omitempty"`
}

// Create is used to create a new configuration version. The created
// configuration version will be usable once data is uploaded to it.
func (s *configurationVersions) Create(ctx context.Context, workspaceID string, options ConfigurationVersionCreateOptions) (*ConfigurationVersion, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("Invalid value for workspace ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/configuration-versions", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	cv := &ConfigurationVersion{}
	err = s.client.do(ctx, req, cv)
	if err != nil {
		return nil, err
	}

	return cv, nil
}

// Read a configuration version by its ID.
func (s *configurationVersions) Read(ctx context.Context, cvID string) (*ConfigurationVersion, error) {
	if !validStringID(&cvID) {
		return nil, errors.New("Invalid value for configuration version ID")
	}

	u := fmt.Sprintf("configuration-versions/%s", url.QueryEscape(cvID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	cv := &ConfigurationVersion{}
	err = s.client.do(ctx, req, cv)
	if err != nil {
		return nil, err
	}

	return cv, nil
}

// Upload packages and uploads Terraform configuration files. It requires the
// upload URL from a configuration version and the path to the configuration
// files on disk.
func (s *configurationVersions) Upload(ctx context.Context, url, path string) error {
	body := bytes.NewBuffer(nil)

	_, err := slug.Pack(path, body)
	if err != nil {
		return err
	}

	req, err := s.client.newRequest("PUT", url, body)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
