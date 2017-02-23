package spotinst

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/spotinst/spotinst-sdk-go/spotinst/util/uritemplates"
)

// Subscription is an interface for interfacing with the Subscription
// endpoints of the Spotinst API.
type SubscriptionService interface {
	List(*ListSubscriptionInput) (*ListSubscriptionOutput, error)
	Create(*CreateSubscriptionInput) (*CreateSubscriptionOutput, error)
	Read(*ReadSubscriptionInput) (*ReadSubscriptionOutput, error)
	Update(*UpdateSubscriptionInput) (*UpdateSubscriptionOutput, error)
	Delete(*DeleteSubscriptionInput) (*DeleteSubscriptionOutput, error)
}

// SubscriptionServiceOp handles communication with the balancer related methods
// of the Spotinst API.
type SubscriptionServiceOp struct {
	client *Client
}

var _ SubscriptionService = &SubscriptionServiceOp{}

type Subscription struct {
	ID         *string                `json:"id,omitempty"`
	ResourceID *string                `json:"resourceId,omitempty"`
	EventType  *string                `json:"eventType,omitempty"`
	Protocol   *string                `json:"protocol,omitempty"`
	Endpoint   *string                `json:"endpoint,omitempty"`
	Format     map[string]interface{} `json:"eventFormat,omitempty"`
}

type ListSubscriptionInput struct{}

type ListSubscriptionOutput struct {
	Subscriptions []*Subscription `json:"subscriptions,omitempty"`
}

type CreateSubscriptionInput struct {
	Subscription *Subscription `json:"subscription,omitempty"`
}

type CreateSubscriptionOutput struct {
	Subscription *Subscription `json:"subscription,omitempty"`
}

type ReadSubscriptionInput struct {
	ID *string `json:"subscriptionId,omitempty"`
}

type ReadSubscriptionOutput struct {
	Subscription *Subscription `json:"subscription,omitempty"`
}

type UpdateSubscriptionInput struct {
	Subscription *Subscription `json:"subscription,omitempty"`
}

type UpdateSubscriptionOutput struct {
	Subscription *Subscription `json:"subscription,omitempty"`
}

type DeleteSubscriptionInput struct {
	ID *string `json:"subscriptionId,omitempty"`
}

type DeleteSubscriptionOutput struct{}

func subscriptionFromJSON(in []byte) (*Subscription, error) {
	b := new(Subscription)
	if err := json.Unmarshal(in, b); err != nil {
		return nil, err
	}
	return b, nil
}

func subscriptionsFromJSON(in []byte) ([]*Subscription, error) {
	var rw responseWrapper
	if err := json.Unmarshal(in, &rw); err != nil {
		return nil, err
	}
	out := make([]*Subscription, len(rw.Response.Items))
	if len(out) == 0 {
		return out, nil
	}
	for i, rb := range rw.Response.Items {
		b, err := subscriptionFromJSON(rb)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

func subscriptionsFromHttpResponse(resp *http.Response) ([]*Subscription, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return subscriptionsFromJSON(body)
}

func (s *SubscriptionServiceOp) List(input *ListSubscriptionInput) (*ListSubscriptionOutput, error) {
	r := s.client.newRequest("GET", "/events/subscription")

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	gs, err := subscriptionsFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	return &ListSubscriptionOutput{Subscriptions: gs}, nil
}

func (s *SubscriptionServiceOp) Create(input *CreateSubscriptionInput) (*CreateSubscriptionOutput, error) {
	r := s.client.newRequest("POST", "/events/subscription")
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ss, err := subscriptionsFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	output := new(CreateSubscriptionOutput)
	if len(ss) > 0 {
		output.Subscription = ss[0]
	}

	return output, nil
}

func (s *SubscriptionServiceOp) Read(input *ReadSubscriptionInput) (*ReadSubscriptionOutput, error) {
	path, err := uritemplates.Expand("/events/subscription/{subscriptionId}", map[string]string{
		"subscriptionId": StringValue(input.ID),
	})
	if err != nil {
		return nil, err
	}

	r := s.client.newRequest("GET", path)
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ss, err := subscriptionsFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	output := new(ReadSubscriptionOutput)
	if len(ss) > 0 {
		output.Subscription = ss[0]
	}

	return output, nil
}

func (s *SubscriptionServiceOp) Update(input *UpdateSubscriptionInput) (*UpdateSubscriptionOutput, error) {
	path, err := uritemplates.Expand("/events/subscription/{subscriptionId}", map[string]string{
		"subscriptionId": StringValue(input.Subscription.ID),
	})
	if err != nil {
		return nil, err
	}

	// We do not need the ID anymore so let's drop it.
	input.Subscription.ID = nil

	r := s.client.newRequest("PUT", path)
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ss, err := subscriptionsFromHttpResponse(resp)
	if err != nil {
		return nil, err
	}

	output := new(UpdateSubscriptionOutput)
	if len(ss) > 0 {
		output.Subscription = ss[0]
	}

	return output, nil
}

func (s *SubscriptionServiceOp) Delete(input *DeleteSubscriptionInput) (*DeleteSubscriptionOutput, error) {
	path, err := uritemplates.Expand("/events/subscription/{subscriptionId}", map[string]string{
		"subscriptionId": StringValue(input.ID),
	})
	if err != nil {
		return nil, err
	}

	r := s.client.newRequest("DELETE", path)
	r.obj = input

	_, resp, err := requireOK(s.client.doRequest(r))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &DeleteSubscriptionOutput{}, nil
}
