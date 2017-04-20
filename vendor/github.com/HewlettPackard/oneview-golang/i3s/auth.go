/*
(c) Copyright [2015] Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package I3S for working with HPE ImageStreamer
package i3s

import (
	"encoding/json"
	"strconv"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/docker/machine/libmachine/log"
)

// AuthHeader Marshal a json into a auth header
type AuthHeader struct {
	ContentType string `json:"Content-Type,omitempty"`
	XAPIVersion int    `json:"X-API-Version,omitempty"`
	Auth        string `json:"auth,omitempty"`
}

// GetAuthHeaderMap Generate an auth Header map
func (c *I3SClient) GetAuthHeaderMap() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json; charset=utf-8",
		"X-API-Version": strconv.Itoa(c.APIVersion),
		"auth":          c.APIKey,
	}
}

// GetAuthHeaderMapNoVer generate header without version
func (c *I3SClient) GetAuthHeaderMapNoVer() map[string]string {
	return map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		"auth":         c.APIKey,
	}
}

// Session struct
type Session struct {
	ID string `json:"sessionID,omitempty"`
}

// TimeOut structure
type TimeOut struct {
	IdleTimeout int64 `json:"idleTimeout"`
}

// GetIdleTimeout gets the current timeout for the logged on session
// returns timeout in milliseconds, or error when it fails
func (c *I3SClient) GetIdleTimeout() (int64, error) {
	var (
		uri     = "/rest/sessions/idle-timeout"
		timeout TimeOut
		header  map[string]string
	)
	log.Debugf("Calling idel-timeout get for header -> %+v", c.GetAuthHeaderMap())
	header = c.GetAuthHeaderMap()
	header["Session-ID"] = header["auth"]
	c.SetAuthHeaderOptions(header)
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return -1, err
	}
	log.Debugf("Timeout data %s", data)
	if err := json.Unmarshal([]byte(data), &timeout); err != nil {
		return -1, err
	}
	return timeout.IdleTimeout, nil
}

// SetIdleTimeout sets the current timeout
func (c *I3SClient) SetIdleTimeout(thetime int64) error {
	var (
		uri     = "/rest/sessions/idle-timeout"
		timeout TimeOut
		header  map[string]string
	)
	timeout.IdleTimeout = thetime
	log.Debugf("Calling idel-timeout POST for header -> %+v", c.GetAuthHeaderMap())
	header = c.GetAuthHeaderMap()
	header["Session-ID"] = header["auth"]
	c.SetAuthHeaderOptions(header)
	_, err := c.RestAPICall(rest.POST, uri, timeout)
	if err != nil {
		return err
	}
	return nil
}
