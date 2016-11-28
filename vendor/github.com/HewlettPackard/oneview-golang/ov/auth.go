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

// Package ov for working with HP OneView
package ov

import (
	"encoding/json"
	"strconv"
	"strings"

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
func (c *OVClient) GetAuthHeaderMap() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json; charset=utf-8",
		"X-API-Version": strconv.Itoa(c.APIVersion),
		"auth":          c.APIKey,
	}
}

// GetAuthHeaderMapNoVer generate header without version
func (c *OVClient) GetAuthHeaderMapNoVer() map[string]string {
	return map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		"auth":         c.APIKey,
	}
}

// Session struct
type Session struct {
	ID string `json:"sessionID,omitempty"`
}

// Auth structure
type Auth struct {
	UserName string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
	Domain   string `json:"authLoginDomain,omitempty"`
}

// TimeOut structure
type TimeOut struct {
	IdleTimeout int64 `json:"idleTimeout"`
}

// RefreshLogin Refresh login authkey
// Should make sure we have a valid APIKey
func (c *OVClient) RefreshLogin() error {
	if c.APIKey == "" || len(strings.TrimSpace(c.APIKey)) == 0 || c.APIKey == "none" {
		log.Debugf("Getting new session id")
		s, err := c.SessionLogin()
		if err != nil {
			return err
		}
		c.APIKey = s.ID
	}
	// check it we are getting 404 Not Found from GetIdleTimeout, this means the Session-ID is no good
	_, err := c.GetIdleTimeout()
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		s, err := c.SessionLogin()
		if err != nil {
			return err
		}
		c.APIKey = s.ID
	}
	return nil
}

// SessionLogin Login to OneView and get a session ID
// returns Session structure
func (c *OVClient) SessionLogin() (Session, error) {
	var (
		uri     = "/rest/login-sessions"
		body    = Auth{UserName: c.User, Password: c.Password, Domain: c.Domain}
		session Session
	)

	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.POST, uri, body)
	if err != nil {
		return session, err
	}

	log.Debugf("SessionLogin %s", data)
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return session, err
	}
	// Update APIKey
	return session, err
}

// SessionLogout Logout to OneView and get a session ID
// returns Session structure
func (c *OVClient) SessionLogout() error {
	var (
		uri = "/rest/login-sessions"
	)
	log.Debugf("Calling logout for header -> %+v", c.GetAuthHeaderMap())
	if c.APIKey == "none" {
		log.Debugf("already logged out")
		return nil
	}
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	_, err := c.RestAPICall(rest.DELETE, uri, nil)
	if err != nil {
		log.Debugf("Error from %s :-> %+v", uri, err)
		return err
	}
	c.APIKey = "none"
	return nil
}

// GetIdleTimeout gets the current timeout for the logged on session
// returns timeout in milliseconds, or error when it fails
func (c *OVClient) GetIdleTimeout() (int64, error) {
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
func (c *OVClient) SetIdleTimeout(thetime int64) error {
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
