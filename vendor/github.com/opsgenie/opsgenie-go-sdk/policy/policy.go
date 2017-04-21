/*
Copyright 2015 OpsGenie. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package policy provides requests and response structures to achieve Policy API actions.
package policy

// EnablePolicyRequest provides necessary parameter structure to Enable a policy at OpsGenie.
type EnablePolicyRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
}

// DisablePolicyRequest provides necessary parameter structure to Disable a policy at OpsGenie.
type DisablePolicyRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
}
