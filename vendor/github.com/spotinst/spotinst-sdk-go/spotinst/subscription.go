package spotinst

import (
	"fmt"
	"net/http"
)

// SubscriptionService handles communication with the Subscriptio related
// methods of the Spotinst API.
type SubscriptionService struct {
	client *Client
}

type Subscription struct {
	ID         *string                `json:"id,omitempty"`
	ResourceID *string                `json:"resourceId,omitempty"`
	EventType  *string                `json:"eventType,omitempty"`
	Protocol   *string                `json:"protocol,omitempty"`
	Endpoint   *string                `json:"endpoint,omitempty"`
	Format     map[string]interface{} `json:"eventFormat,omitempty"`
}

type SubscriptionResponse struct {
	Response struct {
		Errors []Error         `json:"errors"`
		Items  []*Subscription `json:"items"`
	} `json:"response"`
}

type subscriptionWrapper struct {
	Subscription Subscription `json:"subscription"`
}

// Get an existing subscription configuration.
func (s *SubscriptionService) Get(args ...string) ([]*Subscription, *http.Response, error) {
	var id string
	if len(args) > 0 {
		id = args[0]
	}

	path := fmt.Sprintf("events/subscription/%s", id)
	req, err := s.client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	var retval SubscriptionResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, err
}

// Create a new subscription.
func (s *SubscriptionService) Create(subscription *Subscription) ([]*Subscription, *http.Response, error) {
	path := "events/subscription"

	req, err := s.client.NewRequest("POST", path, subscriptionWrapper{Subscription: *subscription})
	if err != nil {
		return nil, nil, err
	}

	var retval SubscriptionResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, nil
}

// Update an existing subscription.
func (s *SubscriptionService) Update(subscription *Subscription) ([]*Subscription, *http.Response, error) {
	id := (*subscription).ID
	(*subscription).ID = nil
	path := fmt.Sprintf("events/subscription/%s", *id)

	req, err := s.client.NewRequest("PUT", path, subscriptionWrapper{Subscription: *subscription})
	if err != nil {
		return nil, nil, err
	}

	var retval SubscriptionResponse
	resp, err := s.client.Do(req, &retval)
	if err != nil {
		return nil, resp, err
	}

	return retval.Response.Items, resp, nil
}

// Delete an existing subscription.
func (s *SubscriptionService) Delete(subscription *Subscription) (*http.Response, error) {
	id := (*subscription).ID
	(*subscription).ID = nil
	path := fmt.Sprintf("events/subscription/%s", *id)

	req, err := s.client.NewRequest("DELETE", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// String creates a reasonable string representation.
func (s Subscription) String() string {
	return Stringify(s)
}