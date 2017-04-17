package dnsimple

import (
	"fmt"
)

// WebhooksService handles communication with the webhook related
// methods of the DNSimple API.
//
// See https://developer.dnsimple.com/v2/webhooks
type WebhooksService struct {
	client *Client
}

// Webhook represents a DNSimple webhook.
type Webhook struct {
	ID  int    `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

// WebhookResponse represents a response from an API method that returns a Webhook struct.
type WebhookResponse struct {
	Response
	Data *Webhook `json:"data"`
}

// WebhookResponse represents a response from an API method that returns a collection of Webhook struct.
type WebhooksResponse struct {
	Response
	Data []Webhook `json:"data"`
}

// webhookPath generates the resource path for given webhook.
func webhookPath(accountID string, webhookID int) (path string) {
	path = fmt.Sprintf("/%v/webhooks", accountID)
	if webhookID != 0 {
		path = fmt.Sprintf("%v/%v", path, webhookID)
	}
	return
}

// ListWebhooks lists the webhooks for an account.
//
// See https://developer.dnsimple.com/v2/webhooks#list
func (s *WebhooksService) ListWebhooks(accountID string, _ *ListOptions) (*WebhooksResponse, error) {
	path := versioned(webhookPath(accountID, 0))
	webhooksResponse := &WebhooksResponse{}

	resp, err := s.client.get(path, webhooksResponse)
	if err != nil {
		return webhooksResponse, err
	}

	webhooksResponse.HttpResponse = resp
	return webhooksResponse, nil
}

// CreateWebhook creates a new webhook.
//
// See https://developer.dnsimple.com/v2/webhooks#create
func (s *WebhooksService) CreateWebhook(accountID string, webhookAttributes Webhook) (*WebhookResponse, error) {
	path := versioned(webhookPath(accountID, 0))
	webhookResponse := &WebhookResponse{}

	resp, err := s.client.post(path, webhookAttributes, webhookResponse)
	if err != nil {
		return nil, err
	}

	webhookResponse.HttpResponse = resp
	return webhookResponse, nil
}

// GetWebhook fetches a webhook.
//
// See https://developer.dnsimple.com/v2/webhooks#get
func (s *WebhooksService) GetWebhook(accountID string, webhookID int) (*WebhookResponse, error) {
	path := versioned(webhookPath(accountID, webhookID))
	webhookResponse := &WebhookResponse{}

	resp, err := s.client.get(path, webhookResponse)
	if err != nil {
		return nil, err
	}

	webhookResponse.HttpResponse = resp
	return webhookResponse, nil
}

// DeleteWebhook PERMANENTLY deletes a webhook from the account.
//
// See https://developer.dnsimple.com/v2/webhooks#delete
func (s *WebhooksService) DeleteWebhook(accountID string, webhookID int) (*WebhookResponse, error) {
	path := versioned(webhookPath(accountID, webhookID))
	webhookResponse := &WebhookResponse{}

	resp, err := s.client.delete(path, nil, nil)
	if err != nil {
		return nil, err
	}

	webhookResponse.HttpResponse = resp
	return webhookResponse, nil
}
