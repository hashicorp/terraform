/*
Copyright 2016. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package escalation provides requests and response structures to achieve Escalation API actions.
package escalation

// Rule defines the structure for each escalation rule definition
type Rule struct {
	Delay int `json:"delay"`
	Notify string `json:"notify,omitempty"`
	NotifyType string `json:"notifyType,omitempty"`
	NotifyCondition string `json:"notifyCondition,omitempty"`
}

// CreateEscalationRequest provides necessary parameter structure for creating escalation
type CreateEscalationRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Name   string `json:"name,omitempty"`
        Rules []Rule `json:"rules,omitempty"`
}

// UpdateEscalationRequest provides necessary parameter structure for updating an escalation
type UpdateEscalationRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Id     string `json:"id,omitempty"`
        Name   string `json:"name,omitempty"`
        Rules []Rule `json:"rules,omitempty"`
}

// DeleteEscalationRequest provides necessary parameter structure for deleting an escalation
type DeleteEscalationRequest struct {
	APIKey	string `url:"apiKey,omitempty"`
	ID      string `url:"id,omitempty"`
	Name	string `url:"name,omitempty"`
}

// GetEscalationRequest provides necessary parameter structure for requesting escalation information
type GetEscalationRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id     string `url:"id,omitempty"`
        Name   string `url:"name,omitempty"`
}

// ListEscalationRequest provides necessary parameter structure for listing escalations
type ListEscalationsRequest struct {
	APIKey string `url:"apiKey,omitempty"`
}
