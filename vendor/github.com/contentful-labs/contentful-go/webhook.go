package contentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// WebhooksService service
type WebhooksService service

// Webhook model
type Webhook struct {
	Sys               *Sys             `json:"sys,omitempty"`
	Name              string           `json:"name,omitempty"`
	URL               string           `json:"url,omitempty"`
	Topics            []string         `json:"topics,omitempty"`
	HTTPBasicUsername string           `json:"httpBasicUsername,omitempty"`
	HTTPBasicPassword string           `json:"httpBasicPassword,omitempty"`
	Headers           []*WebhookHeader `json:"headers,omitempty"`
}

// WebhookHeader model
type WebhookHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetVersion returns entity version
func (webhook *Webhook) GetVersion() int {
	version := 1
	if webhook.Sys != nil {
		version = webhook.Sys.Version
	}

	return version
}

// List returns webhooks collection
func (service *WebhooksService) List(spaceID string) *Collection {
	path := fmt.Sprintf("/spaces/%s/webhook_definitions", spaceID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return &Collection{}
	}

	col := NewCollection(&CollectionOptions{})
	col.c = service.c
	col.req = req

	return col
}

// Get returns a single webhook entity
func (service *WebhooksService) Get(spaceID, webhookID string) (*Webhook, error) {
	path := fmt.Sprintf("/spaces/%s/webhook_definitions/%s", spaceID, webhookID)
	method := "GET"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := service.c.do(req, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// Upsert updates or creates a new entity
func (service *WebhooksService) Upsert(spaceID string, webhook *Webhook) error {
	bytesArray, err := json.Marshal(webhook)
	if err != nil {
		return err
	}

	var path string
	var method string

	if webhook.Sys != nil && webhook.Sys.CreatedAt != "" {
		path = fmt.Sprintf("/spaces/%s/webhook_definitions/%s", spaceID, webhook.Sys.ID)
		method = "PUT"
	} else {
		path = fmt.Sprintf("/spaces/%s/webhook_definitions", spaceID)
		method = "POST"
	}

	req, err := service.c.newRequest(method, path, nil, bytes.NewReader(bytesArray))
	if err != nil {
		return err
	}

	req.Header.Set("X-Contentful-Version", strconv.Itoa(webhook.GetVersion()))

	if ok := service.c.do(req, webhook); ok != nil {
		return ok
	}

	return nil
}

// Delete the webhook
func (service *WebhooksService) Delete(spaceID string, webhook *Webhook) error {
	path := fmt.Sprintf("/spaces/%s/webhook_definitions/%s", spaceID, webhook.Sys.ID)
	method := "DELETE"

	req, err := service.c.newRequest(method, path, nil, nil)
	if err != nil {
		return err
	}

	version := strconv.Itoa(webhook.Sys.Version)
	req.Header.Set("X-Contentful-Version", version)

	if err = service.c.do(req, nil); err != nil {
		return err
	}

	return nil
}
