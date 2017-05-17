/*
Copyright 2015 OpsGenie. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package integration provides requests and response structures to achieve Integration API actions.
package integration

// EnableIntegrationRequest provides necessary parameter structure to Enable an integration at OpsGenie.
type EnableIntegrationRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
}

// DisableIntegrationRequest provides necessary parameter structure to Disable an integration at OpsGenie.
type DisableIntegrationRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
}
