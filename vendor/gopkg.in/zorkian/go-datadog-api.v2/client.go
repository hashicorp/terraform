/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// Client is the object that handles talking to the Datadog API. This maintains
// state information for a particular application connection.
type Client struct {
	apiKey, appKey string

	//The Http Client that is used to make requests
	HttpClient   *http.Client
	RetryTimeout time.Duration
}

// valid is the struct to unmarshal validation endpoint responses into.
type valid struct {
	Errors  []string `json:"errors"`
	IsValid bool     `json:"valid"`
}

// NewClient returns a new datadog.Client which can be used to access the API
// methods. The expected argument is the API key.
func NewClient(apiKey, appKey string) *Client {
	return &Client{
		apiKey:       apiKey,
		appKey:       appKey,
		HttpClient:   http.DefaultClient,
		RetryTimeout: time.Duration(60 * time.Second),
	}
}

// SetKeys changes the value of apiKey and appKey.
func (c *Client) SetKeys(apiKey, appKey string) {
	c.apiKey = apiKey
	c.appKey = appKey
}

// Validate checks if the API and application keys are valid.
func (client *Client) Validate() (bool, error) {
	var bodyreader io.Reader
	var out valid
	req, err := http.NewRequest("GET", client.uriForAPI("/v1/validate"), bodyreader)

	if err != nil {
		return false, err
	}
	if bodyreader != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	var resp *http.Response
	resp, err = client.HttpClient.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	// Only care about 200 OK or 403 which we'll unmarshal into struct valid. Everything else is of no interest to us.
	if resp.StatusCode != 200 && resp.StatusCode != 403 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		return false, fmt.Errorf("API error %s: %s", resp.Status, body)
	}

	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &out)
	if err != nil {
		return false, err
	}

	return out.IsValid, nil
}
