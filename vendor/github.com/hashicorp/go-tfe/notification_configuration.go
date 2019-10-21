package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ NotificationConfigurations = (*notificationConfigurations)(nil)

// NotificationConfigurations describes all the Notification Configuration
// related methods that the Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/notification-configurations.html
type NotificationConfigurations interface {
	// List all the notification configurations within a workspace.
	List(ctx context.Context, workspaceID string, options NotificationConfigurationListOptions) (*NotificationConfigurationList, error)

	// Create a new notification configuration with the given options.
	Create(ctx context.Context, workspaceID string, options NotificationConfigurationCreateOptions) (*NotificationConfiguration, error)

	// Read a notification configuration by its ID.
	Read(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error)

	// Update an existing notification configuration.
	Update(ctx context.Context, notificationConfigurationID string, options NotificationConfigurationUpdateOptions) (*NotificationConfiguration, error)

	// Delete a notification configuration by its ID.
	Delete(ctx context.Context, notificationConfigurationID string) error

	// Verify a notification configuration by its ID.
	Verify(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error)
}

// notificationConfigurations implements NotificationConfigurations.
type notificationConfigurations struct {
	client *Client
}

// List of available notification triggers.
const (
	NotificationTriggerCreated        string = "run:created"
	NotificationTriggerPlanning       string = "run:planning"
	NotificationTriggerNeedsAttention string = "run:needs_attention"
	NotificationTriggerApplying       string = "run:applying"
	NotificationTriggerCompleted      string = "run:completed"
	NotificationTriggerErrored        string = "run:errored"
)

// NotificationDestinationType represents the destination type of the
// notification configuration.
type NotificationDestinationType string

// List of available notification destination types.
const (
	NotificationDestinationTypeSlack   NotificationDestinationType = "slack"
	NotificationDestinationTypeGeneric NotificationDestinationType = "generic"
)

// NotificationConfigurationList represents a list of Notification
// Configurations.
type NotificationConfigurationList struct {
	*Pagination
	Items []*NotificationConfiguration
}

// NotificationConfiguration represents a Notification Configuration.
type NotificationConfiguration struct {
	ID                string                      `jsonapi:"primary,notification-configurations"`
	CreatedAt         time.Time                   `jsonapi:"attr,created-at,iso8601"`
	DeliveryResponses []*DeliveryResponse         `jsonapi:"attr,delivery-responses"`
	DestinationType   NotificationDestinationType `jsonapi:"attr,destination-type"`
	Enabled           bool                        `jsonapi:"attr,enabled"`
	Name              string                      `jsonapi:"attr,name"`
	Token             string                      `jsonapi:"attr,token"`
	Triggers          []string                    `jsonapi:"attr,triggers"`
	UpdatedAt         time.Time                   `jsonapi:"attr,updated-at,iso8601"`
	URL               string                      `jsonapi:"attr,url"`
}

// DeliveryResponse represents a notification configuration delivery response.
type DeliveryResponse struct {
	Body       string      `json:"body"`
	Code       int         `json:"code"`
	Headers    http.Header `json:"headers"`
	SentAt     time.Time   `json:"sent-at,iso8601"`
	Successful bool        `json:"successful"`
	URL        string      `json:"url"`
}

// NotificationConfigurationListOptions represents the options for listing
// notification configurations.
type NotificationConfigurationListOptions struct {
	ListOptions
}

// List all the notification configurations associated with a workspace.
func (s *notificationConfigurations) List(ctx context.Context, workspaceID string, options NotificationConfigurationListOptions) (*NotificationConfigurationList, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/notification-configurations", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	ncl := &NotificationConfigurationList{}
	err = s.client.do(ctx, req, ncl)
	if err != nil {
		return nil, err
	}

	return ncl, nil
}

// NotificationConfigurationCreateOptions represents the options for
// creating a new notification configuration.
type NotificationConfigurationCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,notification-configurations"`

	// The destination type of the notification configuration
	DestinationType *NotificationDestinationType `jsonapi:"attr,destination-type"`

	// Whether the notification configuration should be enabled or not
	Enabled *bool `jsonapi:"attr,enabled"`

	// The name of the notification configuration
	Name *string `jsonapi:"attr,name"`

	// The token of the notification configuration
	Token *string `jsonapi:"attr,token,omitempty"`

	// The destination type of the notification configuration
	Triggers []string `jsonapi:"attr,triggers,omitempty"`

	// The url of the notification configuration
	URL *string `jsonapi:"attr,url"`
}

func (o NotificationConfigurationCreateOptions) valid() error {
	if o.DestinationType == nil {
		return errors.New("destination type is required")
	}
	if o.Enabled == nil {
		return errors.New("enabled is required")
	}
	if !validString(o.Name) {
		return errors.New("name is required")
	}
	if !validString(o.URL) {
		return errors.New("url is required")
	}
	return nil
}

// Creates a notification configuration with the given options.
func (s *notificationConfigurations) Create(ctx context.Context, workspaceID string, options NotificationConfigurationCreateOptions) (*NotificationConfiguration, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/notification-configurations", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = s.client.do(ctx, req, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// Read a notitification configuration by its ID.
func (s *notificationConfigurations) Read(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error) {
	if !validStringID(&notificationConfigurationID) {
		return nil, errors.New("invalid value for notification configuration ID")
	}

	u := fmt.Sprintf("notification-configurations/%s", url.QueryEscape(notificationConfigurationID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = s.client.do(ctx, req, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// NotificationConfigurationUpdateOptions represents the options for
// updating a existing notification configuration.
type NotificationConfigurationUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,notification-configurations"`

	// Whether the notification configuration should be enabled or not
	Enabled *bool `jsonapi:"attr,enabled,omitempty"`

	// The name of the notification configuration
	Name *string `jsonapi:"attr,name,omitempty"`

	// The token of the notification configuration
	Token *string `jsonapi:"attr,token,omitempty"`

	// The destination type of the notification configuration
	Triggers []string `jsonapi:"attr,triggers,omitempty"`

	// The url of the notification configuration
	URL *string `jsonapi:"attr,url,omitempty"`
}

// Updates a notification configuration with the given options.
func (s *notificationConfigurations) Update(ctx context.Context, notificationConfigurationID string, options NotificationConfigurationUpdateOptions) (*NotificationConfiguration, error) {
	if !validStringID(&notificationConfigurationID) {
		return nil, errors.New("invalid value for notification configuration ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("notification-configurations/%s", url.QueryEscape(notificationConfigurationID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = s.client.do(ctx, req, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

// Delete a notifications configuration by its ID.
func (s *notificationConfigurations) Delete(ctx context.Context, notificationConfigurationID string) error {
	if !validStringID(&notificationConfigurationID) {
		return errors.New("invalid value for notification configuration ID")
	}

	u := fmt.Sprintf("notification-configurations/%s", url.QueryEscape(notificationConfigurationID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// Verifies a notification configuration by delivering a verification
// payload to the configured url.
func (s *notificationConfigurations) Verify(ctx context.Context, notificationConfigurationID string) (*NotificationConfiguration, error) {
	if !validStringID(&notificationConfigurationID) {
		return nil, errors.New("invalid value for notification configuration ID")
	}

	u := fmt.Sprintf(
		"notification-configurations/%s/actions/verify", url.QueryEscape(notificationConfigurationID))
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	nc := &NotificationConfiguration{}
	err = s.client.do(ctx, req, nc)
	if err != nil {
		return nil, err
	}

	return nc, nil
}
